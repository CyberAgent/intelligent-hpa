# Intelligent HPA

Intelligent HPA is a Kubernetes custom controller for scaling replicas by prediction of metrics which is refered by HPA.

- [Prerequisite](#prerequisite)
- [Usage](#usage)
- [Installation](#installation)
- [Fitting Job](#fitting-job)
- [Other docs](#other-docs)

## Prerequisite

Select metrics provider for storing predicted metrics.

### Datadog

Install Datadog Agent and Datadog Cluster Agent.

```
helm repo add stable https://kubernetes-charts.storage.googleapis.com

KUBE_SYSTEM_UID=$(kubectl get namespace kube-system -o jsonpath='{.metadata.uid}')

helm install datadog stable/datadog \
  --set datadog.apiKey=xxx \
  --set datadog.appKey=yyy \
  --set clusterAgent.enabled=true \
  --set clusterAgent.metricsProvider.enabled=true \
  --set datadog.tags={"kube_system_uid:${KUBE_SYSTEM_UID}"}
```

If you already installed them, you have to add a tag of `kube-system` Namespace UID for unique identification of your cluster. The tag is needed for identification of **Resource** metrics such as `cpu` and `memory`. Datadog Agent will be restarted by running below commands.

```sh
TMP_DD_TAGS=$(kubectl get ds <your_datadog_agent> -o jsonpath='{.spec.template.spec.containers[*].env[?(@.name == "DD_TAGS")].value}')
KUBE_SYSTEM_UID=$(kubectl get namespace kube-system -o jsonpath='{.metadata.uid}')

TMP_DD_TAGS="${TMP_DD_TAGS} kube_system_uid:${KUBE_SYSTEM_UID}"

kubectl patch ds <your_datadog_agent> -p "{\"spec\":{\"template\":{\"spec\":{\"containers\":[{\"name\":\"agent\",\"env\":[{\"name\":\"DD_TAGS\",\"value\":\"${TMP_DD_TAGS}\"}]}]}}}}"

# if error occurred, please change containers[0].name replace to "datadog"
```

### Prometheus

not yet implemented

## Usage

IHPA manifest has some field below:

- `estimator`
    - Settings for Estimator resource
    - `gapMinutes`
        - Time to slide predictive metrics timestamp (in minute)
        - Generally set time to ready of your application
        - e.g.) The deployment scales based on the metrics 5 minutes ahead if you set `5`.
            - Generated HPA refers to actual metrics too, so there is no scale-in when the predictive metrics are reduced but the actual load is high
        - Default: `5`
    - `mode`
        - Adjustment mode sending predictive metrics to providers
        - `adjust`: Adjust predictive metrics based on difference between previous predictive metrics and actual metrics
        - `raw`: No adjustment
        - Allowable: `adjust`, `raw` (default: `adjust`)
- `metricProvider`
    - Provider for sending and fetching metrics
    - Only Datadog is supported now
- `template`
    - Almost same template as HorizontalPodAutoscaler
    - You can copy/paste HPA manifests to this field
    - Only `Resource` and `External` metrics are supported now
- `fittingJob`
    - Settings for FittingJob resource
    - This can be set in `.spec.template.spec.metrics`
    - `seasonality`
        - Seasonality of metrics
        - Allowable: `auto`, `daily`, `weekly`, `yearly` (default: `auto`)
    - `executeOn`
        - Time to execute fittingJob (CronJob)
        - e.g.) The fittingJob is executed at 12:XX (XX is random) if you set `12`.
        - default: `4`
    - `changePointDetectionConfig`
        - Parameters for change point detection
          - `percentageThreshold` (default: `50`)
          - `windowSize` (default: `100`)
          - `trajectoryRows` (default: `50`)
          - `trajectoryFeatures` (default: `5`)
          - `testRows` (default: `50`)
          - `testFeatures` (default: `5`)
          - `lag` (default: `288`)
        - See [FittingJob section](#fitting-job) for details
    - `customConfig`
        - Arbitrary string passed to fittingJob
    - `image`
        - Container image name for fittingJob
        - default: `cyberagentoss/intelligent-hpa-fittingjob:latest`
    - `imagePullSecrets`
        - Secret for pulling fittingJob image
    - See [this struct](https://github.com/cyberagent-oss/intelligent-hpa/blob/master/ihpa-controller/api/v1beta2/fittingjob_types.go#L63-L86) for other parameters

```yaml
---
apiVersion: ihpa.ake.cyberagent.co.jp/v1beta2
kind: IntelligentHorizontalPodAutoscaler
metadata:
  name: nginx
spec:
  estimator:
    gapMinutes: 10
    mode: adjust
  metricProvider:
    name: datadog
    datadog:
      apikey: xxx
      appkey: yyy
  template:
    spec:
      scaleTargetRef:
        apiVersion: apps/v1
        kind: Deployment
        name: nginx
      minReplicas: 1
      maxReplicas: 5
      metrics:
      - type: Resource
        resource:
          name: cpu
          target:
            type: Utilization
            averageUtilization: 50
        fittingJob: 
          seasonality: daily
          executeOn: 12
          changePointDetectionConfig:
            percentageThreshold: 50
            windowSize: 100
            trajectoryRows: 50
            trajectoryFeatures: 5
            testRows: 50
            testFeatures: 5
            lag: 288
          customConfig: '{"hello":"world"}'
          image: your-job-image:v1
          imagePullSecrets:
          - name: pull-secret
```

You can see metrics status by `kubectl describe hpa.v2beta2.autoscaling xxx`. A predictive metric has `ake.ihpa` prefix. The metric sometimes shows weird value but it is not problem. This is caused by workaround for issue that the HPA interprets the provider's values without scale unit information.

```yaml
Name:                                                                       ihpa-nginx
Namespace:                                                                  loadtest
Labels:                                                                     <none>
Annotations:                                                                <none>
CreationTimestamp:                                                          Mon, 30 Mar 2020 16:05:29 +0900
Reference:                                                                  Deployment/nginx
Metrics:                                                                    ( current / target )
  "nginx.net.request_per_s" (target average value):                         26657m / 50
  "ake.ihpa.forecasted_kubernetes_cpu_usage_total" (target average value):  2936862750m / 50M
  "ake.ihpa.forecasted_nginx_net_request_per_s" (target average value):     43532m / 50
  resource cpu on pods  (as a percentage of request):                       4% (4m) / 50%
Min replicas:                                                               1
Max replicas:                                                               30
Deployment pods:                                                            2 current / 2 desired
```

Immediately after apply manifest, there are no predictive metrics. The fittingJob process starts on time of `executeOn`. You can start CronJob manually too as follows if you want. Please change the name of CronJob to match your environment.

```sh
kubectl create job -n loadtest --from cronjob/ihpa-nginx-nginx-net-request-per-s manual-train
```

## Installation

Create manifest directly. FittingJob CRD is very large, so if you use `apply`, you will be stuck with the capacity limit of manifest size.

```
kubectl create -f manifests/intelligent-hpa.yaml
```

## Fitting Job

Default fittingJob image does time series prediction using Prophet. This library can predict mertics well without tuning parameters.

Besides, fittingJob has change point detection. This feature is used for selection of training data. Because of this, IHPA can drop data which have bad influence if the workload has changed. These parameters required tuning, so you don't have to use it unless you want to use it. These parameters is below:

|parameter          |description|
|:-----------------:|:----------|
|percentageThreshold|Threshold of rate of change in percentage. (Allowable: 1-99)|
|windowSize         |Width (Columns) of matrix of sub time series. The smaller the range, the more sensitive it is to change, as it calculates anomalies by comparing them in the range.|
|trajectoryRows     |Number of columns of trajectory matrix (sub time series)|
|trajectoryFeatures |Number of features of SVDed trajectory matrix. This removes noise and improves accuracy. This value must be smaller than `windowSize`.|
|testRows           |Number of columns of test matrix (sub time series)|
|testFeatures       |Number of features of SVDed trajectory matrix|
|lag                |Shift width between trajectory and test matrics. It's probably a good idea to match `Seasonality` because we can avoid false anomaly detection by comparing the same time period. Basically, I recommend to use 288. That is a number of metrics of one day because we get metrics at 5 minute intervals.|

You can tune these parameters by [Jupyter Notebook](./fittingjob/change_point_detection_tuning.ipynb).

## Other docs (Japanese Only)

- [Architecture](./docs/architecture.md)
- [How Estimator works](./docs/estimator.md)
- [How FittingJob works](./docs/fittingjob.md)
- [For developers](./docs/developer.md)
