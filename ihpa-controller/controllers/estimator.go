package controllers

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cyberagent-oss/intelligent-hpa/ihpa-controller/controllers/metricprovider"
	"github.com/go-logr/logr"
)

const (
	EstimateUnknown = EstimateOperator(iota)
	EstimateAdd
	EstimateRemove
	EstimateUpdate

	AdjustMode = EstimateMode("adjust")
	RawMode    = EstimateMode("raw")

	EstimateTargetsBuffer = 20

	TimeStampLabel = "timestamp"
	YHatLabel      = "yhat"
	YHatUpperLabel = "yhat_upper"
	YHatLowerLabel = "yhat_lower"
)

type EstimateOperator int

type EstimateMode string

type EstimateTarget struct {
	ID             string
	EstimateMode   string
	GapMinutes     int
	DataCh         <-chan []byte
	MetricProvider metricprovider.MetricProvider
	MetricName     string
	MetricTags     []string
	BaseMetricName string
	BaseMetricTags []string

	estimatorStopCh chan struct{}
	logr.Logger
}

type EstimateOperation struct {
	Operator EstimateOperator
	Target   EstimateTarget
}

type EstimateDatum struct {
	UnixTime         int64
	EstimateUnixTime int64
	YHat             float64
	UpperYHat        float64
	LowerYHat        float64
}

func (d *EstimateDatum) String() string {
	return fmt.Sprintf("{org_time: %s, est_time: %s, yhat: %.1f, yhat_upper: %.1f, yhat_lower: %.1f}",
		time.Unix(d.UnixTime, 0), time.Unix(d.EstimateUnixTime, 0), d.YHat, d.UpperYHat, d.LowerYHat)
}

type PastEstimateDatumQueue []EstimateDatum

func (q *PastEstimateDatumQueue) enqueue(d *EstimateDatum) {
	if d == nil {
		return
	}
	*q = append(*q, *d)
}
func (q *PastEstimateDatumQueue) dequeue() *EstimateDatum {
	if len(*q) == 0 {
		return nil
	}
	var d EstimateDatum
	*q, d = (*q)[1:], (*q)[0]
	return &d
}
func (q *PastEstimateDatumQueue) peek() *EstimateDatum {
	if len(*q) == 0 {
		return nil
	}
	var d EstimateDatum
	d = (*q)[0]
	return &d
}
func (q *PastEstimateDatumQueue) seekByUnixTime(u int64) *EstimateDatum {
	var d *EstimateDatum
	for {
		if len(*q) == 0 {
			return d
		}
		if pd := q.peek(); pd.UnixTime > u {
			return d
		}
		d = q.dequeue()
	}
}
func (q *PastEstimateDatumQueue) String() string {
	s := "["
	for _, v := range *q {
		s += fmt.Sprintf("%s,", v.String())
	}
	s = strings.TrimRight(s, ",")
	s += "]"
	return s
}

// adjustYHat adjust current data YHat based on previous data and actual metric.
func (currEd *EstimateDatum) adjustYHat(prevEd *EstimateDatum, actualValue float64) float64 {
	adjusted := currEd.YHat

	// check data integrity
	if !(prevEd.UpperYHat >= prevEd.YHat &&
		prevEd.YHat >= prevEd.LowerYHat &&
		currEd.UpperYHat >= currEd.YHat &&
		currEd.YHat >= currEd.LowerYHat) {
		return adjusted
	}

	if actualValue > prevEd.YHat {
		upperWidth := float64(prevEd.UpperYHat - prevEd.YHat)
		// get min to avoid response anormaly metric
		mag := math.Min(upperWidth, float64(actualValue-prevEd.YHat)) / upperWidth
		adjusted += mag * float64(currEd.UpperYHat-currEd.YHat)
	} else {
		lowerWidth := float64(prevEd.YHat - prevEd.LowerYHat)
		mag := math.Min(lowerWidth, float64(prevEd.YHat-actualValue)) / lowerWidth
		adjusted -= mag * float64(currEd.YHat-currEd.LowerYHat)
	}
	return adjusted
}

// estimatorHandler handle estimate request
func estimatorHandler(opeCh <-chan *EstimateOperation, log logr.Logger) {
	log.V(LogicMessageLogLevel).Info("start estimator handler")
	estimateTargets := make([]EstimateTarget, 0, EstimateTargetsBuffer)
	for {
		select {
		case ope := <-opeCh:
			switch ope.Operator {
			case EstimateAdd:
				et := ope.Target
				log.V(LogicMessageLogLevel).Info("create estimator", "id", et.ID)
				et.estimatorStopCh = make(chan struct{})
				et.Logger = log
				go et.estimator()
				estimateTargets = append(estimateTargets, et)

			case EstimateUpdate:
				patch := ope.Target
				log.V(LogicMessageLogLevel).Info("update estimator", "id", patch.ID)
				for i, et := range estimateTargets {
					if et.ID == patch.ID {
						if err := et.updateEstimateTarget(&patch); err != nil {
							log.V(LogicMessageLogLevel).Info("update estimator error", "error_msg", err)
						}

						close(et.estimatorStopCh)
						et.estimatorStopCh = make(chan struct{})
						go et.estimator()
						estimateTargets[i] = et

						// break search loop, not handler loop
						break
					}
				}

			case EstimateRemove:
				for i, et := range estimateTargets {
					if et.ID == ope.Target.ID {
						log.V(LogicMessageLogLevel).Info("stop estimating", "id", et.ID)
						close(et.estimatorStopCh)
						estimateTargets = append(estimateTargets[:i], estimateTargets[i+1:]...)
						// break search loop, not handler loop
						break
					}
				}
			}
		}
	}
}

// flow
//    receive data (time series metrics)
// -> shift data time stamp by gap
// -> cut down data until now
// -> see first elements of current data
// -> wait time to send data to provider
// -> adjust yhat based on previous actual metric
// -> send data to provider
func (et *EstimateTarget) estimator() {
	et.V(LogicMessageLogLevel).Info("start estimator", "id", et.ID)

	waitTime := time.Duration(5)
	position := 0
	// this should be sorted
	data := make([]EstimateDatum, 0)
	pastDatumQueue := PastEstimateDatumQueue(make([]EstimateDatum, 0, 288)) // 5 minutes interval 1 day capacity

estimatorLoop:
	for {
		select {
		case <-et.estimatorStopCh:
			et.V(LogicMessageLogLevel).Info("receive stop request", "id", et.ID)
			break estimatorLoop
		case <-time.After(waitTime * time.Second):
			// have no data or reach end of data
			if len(data) == 0 || position > len(data)-1 {
				// fall through to check watcherDataCh
				break
			}

			adjustedYHat := data[position].YHat
			// ignore first prediction because we cannot see before data.
			if position != 0 && et.EstimateMode == string(AdjustMode) {
				currData := data[position]
				// look up previous datum which has actual value
				var prevData EstimateDatum
				var prevY float64
				et.V(LogicMessageLogLevel).Info("search data", "time", time.Now())
				if d := pastDatumQueue.seekByUnixTime(time.Now().Unix()); d != nil {
					prevData = *d
					var err error
					prevY, err = et.MetricProvider.Fetch(
						et.MetricProvider.AddSumAggregator(et.BaseMetricName),
						prevData.UnixTime,
						et.BaseMetricTags,
						nil,
					)
					if err != nil {
						et.V(LogicMessageLogLevel).Info("failed to fetch previous data", "error_msg", err)
						prevY = prevData.YHat
					}
					et.V(2).Info("match data", "prevY", prevY, "d", d.String())
				} else {
					et.V(LogicMessageLogLevel).Info("valid previous datum is not found", "past_queue", pastDatumQueue.String())
					prevData = currData
				}
				// adopt only upper adjust
				if yhat := currData.adjustYHat(&prevData, prevY); yhat > adjustedYHat {
					adjustedYHat = yhat
				}
			}

			et.V(LogicMessageLogLevel).Info(
				"send metrics",
				"metricName", et.MetricName,
				"timestamp", time.Unix(data[position].EstimateUnixTime, 0).String(),
				"yhat", data[position].YHat,
				"adjusted_yhat", adjustedYHat,
				"upper_yhat", data[position].UpperYHat,
				"lower_yhat", data[position].LowerYHat,
				"tags", et.MetricTags,
			)

			sendMap := map[string]float64{
				et.MetricName:            adjustedYHat,
				et.MetricName + ".raw":   data[position].YHat,
				et.MetricName + ".upper": data[position].UpperYHat,
				et.MetricName + ".lower": data[position].LowerYHat,
			}
			for metricName, datapoint := range sendMap {
				if err := et.MetricProvider.Send(
					metricName,
					data[position].EstimateUnixTime,
					datapoint,
					et.MetricTags,
					map[string]interface{}{"metricUnitReference": et.BaseMetricName},
				); err != nil {
					et.V(LogicMessageLogLevel).Info("failed to send metric data", "metric_name", metricName, "error_msg", err)
				}
			}
			pastDatumQueue.enqueue(&data[position])

			position++
			if len(data) > position {
				now := time.Now().Unix()
				waitTime = time.Duration(data[position].EstimateUnixTime - now)
			} else {
				// completed to reading all data
				waitTime = time.Duration(5)
			}
		}

		// data input check is separated from sending metrics
		// because if proceed to watcherDataCh case while waiting time.After(),
		// the wait time until now becomes meaningless and new wait time is set.
		select {
		case d := <-et.DataCh:
			et.V(LogicMessageLogLevel).Info("receive data", "id", et.ID)
			newData, err := readEstimateDataAsCSV(bytes.NewReader(d))
			if err != nil {
				et.V(LogicMessageLogLevel).Info("failed to read data", "error_msg", err)
				continue
			}

			// shift by gap
			for i := range newData {
				newData[i].EstimateUnixTime = newData[i].UnixTime - int64(et.GapMinutes)*60
			}

			tmpData := joinEstimateData(newData, data)
			now := time.Now().Unix()

			// cut down old data
			for i, ed := range tmpData {
				if ed.EstimateUnixTime > now {
					data = tmpData[i:]
					position = 0
					break
				}
			}
			waitTime = time.Duration(data[position].EstimateUnixTime - now)
		default:
		}

		if len(data) > position {
			et.V(LogicMessageLogLevel).Info(
				"next time to send metric",
				"metric_name", et.MetricName,
				"remain_time", waitTime*time.Second,
				"next_time", time.Unix(data[position].EstimateUnixTime, 0).String(),
			)
		}
	}

	et.V(LogicMessageLogLevel).Info("stop estimator", "id", et.ID)
}

func (base *EstimateTarget) updateEstimateTarget(patch *EstimateTarget) error {
	if base.ID != patch.ID {
		return fmt.Errorf("target id is not match: base=%s, patch=%s", base.ID, patch.ID)
	}

	// TODO: look for a smart way to overwrite by only not zero value
	if patch.EstimateMode != "" {
		base.EstimateMode = patch.EstimateMode
	}
	if patch.GapMinutes != 0 {
		base.GapMinutes = patch.GapMinutes
	}
	if patch.MetricProvider != nil {
		base.MetricProvider = patch.MetricProvider
	}
	if patch.MetricName != "" {
		base.MetricName = patch.MetricName
	}
	if patch.MetricTags != nil {
		base.MetricTags = patch.MetricTags
	}
	if patch.BaseMetricName != "" {
		base.BaseMetricName = patch.BaseMetricName
	}
	if patch.BaseMetricTags != nil {
		base.BaseMetricTags = patch.BaseMetricTags
	}

	return nil
}

// readEstimateDataAsCSV parse csv data to an array of EstimateDatum.
// This function expects that csv data has header and required
// some column name (TimeStampLabel, YHatLabel, YHatUpperLabel and YHatLowerLabel).
func readEstimateDataAsCSV(r io.Reader) ([]EstimateDatum, error) {
	csvr := csv.NewReader(r)
	records, err := csvr.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(records) <= 1 {
		return nil, fmt.Errorf("not enough data (%v)", records)
	}

	requiredColumnSize := 4
	columnMap := make(map[string]int, requiredColumnSize)
	for i, column := range records[0] {
		switch column {
		case TimeStampLabel:
			columnMap[TimeStampLabel] = i
		case YHatLabel:
			columnMap[YHatLabel] = i
		case YHatUpperLabel:
			columnMap[YHatUpperLabel] = i
		case YHatLowerLabel:
			columnMap[YHatLowerLabel] = i
		}
	}
	if len(columnMap) != requiredColumnSize {
		return nil, fmt.Errorf("column not satisfied requirements (%v)", columnMap)
	}

	data := make([]EstimateDatum, 0, len(records))
	// exclude header
	for _, record := range records[1:] {
		var ed EstimateDatum
		if unixtime, err := strconv.Atoi(record[columnMap[TimeStampLabel]]); err != nil {
			continue
		} else {
			ed.UnixTime = int64(unixtime)
		}
		if yHat, err := strconv.ParseFloat(record[columnMap[YHatLabel]], 64); err != nil {
			continue
		} else {
			ed.YHat = yHat
		}
		if upperYHat, err := strconv.ParseFloat(record[columnMap[YHatUpperLabel]], 64); err != nil {
			continue
		} else {
			ed.UpperYHat = upperYHat
		}
		if lowerYHat, err := strconv.ParseFloat(record[columnMap[YHatLowerLabel]], 64); err != nil {
			continue
		} else {
			ed.LowerYHat = lowerYHat
		}

		data = append(data, ed)
	}

	return data, nil
}

// joinEstimateData join newEds and range of oldEds which older than newEds.
func joinEstimateData(newEds, oldEds []EstimateDatum) []EstimateDatum {
	sort.Slice(newEds, func(i, j int) bool {
		return newEds[i].EstimateUnixTime < newEds[j].EstimateUnixTime
	})
	sort.Slice(oldEds, func(i, j int) bool {
		return oldEds[i].EstimateUnixTime < oldEds[j].EstimateUnixTime
	})

	if len(newEds) == 0 {
		return oldEds
	}
	if len(oldEds) == 0 {
		return newEds
	}

	if newEds[0].EstimateUnixTime > oldEds[len(oldEds)-1].EstimateUnixTime {
		return append(oldEds, newEds...)
	}

	frontEds := oldEds
	newFirstEstimateUnixTime := newEds[0].EstimateUnixTime
	for i := range oldEds {
		if oldEds[i].EstimateUnixTime >= newFirstEstimateUnixTime {
			frontEds = oldEds[:i]
			break
		}
	}
	return append(frontEds, newEds...)
}
