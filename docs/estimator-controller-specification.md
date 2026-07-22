# Estimator Controller 実装仕様書

## 概要

EstimatorReconciler（Estimator Controller）は、Estimatorカスタムリソースを監視し、予測データをメトリクスプロバイダに送信するコントローラーです。ConfigMapに保存された予測データを読み込み、適切なタイミングで調整を行い、メトリクスとして送信します。

## アーキテクチャ

### コントローラーの責務

Estimator Controllerは以下の機能を提供します：

1. **ConfigMap管理** - 予測データ保存用のConfigMapを作成
2. **データ監視** - ConfigMapの変更を監視して新しい予測データを検出
3. **メトリクス送信** - 予測データをメトリクスプロバイダに送信
4. **調整機能** - 過去の予測精度に基づいて予測値を調整

### コンポーネント構成

```
EstimatorReconciler
├── Client (Kubernetes API Client)
├── Logger (構造化ログ)
├── Scheme (Kubernetes スキーマ)
├── opeCh (操作チャネル)
└── estimatorChs (Estimatorごとのデータチャネル)
    └── estimatorHandler (バックグラウンドハンドラー)
        └── EstimateTarget (Estimatorごとの実行インスタンス)
```

## Reconcileループの詳細フロー

### 1. リソース取得フェーズ

```go
func (r *EstimatorReconciler) Reconcile(req ctrl.Request)
```

**処理内容:**
- EstimatorリソースをNamespacedNameで取得
- リソースが見つからない場合（削除された場合）、クリーンアップ処理を実行

**削除時のクリーンアップ:**
```go
if apierrors.IsNotFound(err) {
    log.V(LogicMessageLogLevel).Info("cleanup estimator", "target", req.String())
    delete(r.estimatorChs, req.String())
    r.opeCh <- &EstimateOperation{
        Operator: EstimateRemove,
        Target:   EstimateTarget{ID: req.String()},
    }
}
```

**クリーンアップ処理:**
1. estimatorChsマップからエントリを削除
2. EstimateRemove操作をopeCh経由でestimatorHandlerに送信
3. バックグラウンドのEstimatorゴルーチンを停止

**検証対象要件:** 要件 1.3（削除時のクリーンアップ）

### 2. ジェネレーター初期化フェーズ

```go
g, err := NewEstimatorGenerator(&est, r)
```

**ジェネレーター初期化処理:**
- Estimatorリソースとコントローラーを受け取り、ジェネレーターインスタンスを作成
- Schemeを保存（OwnerReference設定に使用）

**検証対象要件:** 要件 4.1（Estimatorリソース管理）

### 3. ConfigMap生成フェーズ

```go
cmResource, err := g.ConfigMapResource()
```

**ConfigMap生成ロジック:**

```go
func (g *estimatorGeneratorImpl) ConfigMapResource() (*corev1.ConfigMap, error) {
    cm := corev1.ConfigMap{
        ObjectMeta: metav1.ObjectMeta{
            Name:      g.est.Spec.DataConfigMap.Name,
            Namespace: g.est.GetNamespace(),
        },
    }
    
    if err := ctrlutil.SetControllerReference(g.est, &cm, g.scheme); err != nil {
        return nil, err
    }
    
    return &cm, nil
}
```

**特徴:**
- 空のConfigMapを生成（データはFittingJobが書き込む）
- ControllerReferenceを設定（Estimatorが削除されるとConfigMapも削除）
- 名前はEstimatorSpecで指定された名前を使用

**リソース作成:**
```go
if err := r.Get(ctx, types.NamespacedName{...}, cm); err != nil {
    if apierrors.IsNotFound(err) {
        cm = cmResource.DeepCopy()
        if err := r.Create(ctx, cm); err != nil {
            return ctrl.Result{}, fmt.Errorf("failed to create configmap: %w", err)
        }
    }
}
```

**重要な注意点:**
- ConfigMapが既に存在する場合、更新しない
- 理由: FittingJobがConfigMapを更新するため、Estimatorは上書きしない
- 存在チェックのみ実行

**検証対象要件:** 要件 10.1, 10.2（ConfigMap管理とデータ監視）

### 4. Estimator起動/更新フェーズ

**新規Estimatorの起動:**
```go
if _, ok := r.estimatorChs[req.String()]; !ok {
    dataCh := make(chan []byte, 5)
    
    r.opeCh <- &EstimateOperation{
        Operator: EstimateAdd,
        Target: EstimateTarget{
            ID:             req.String(),
            EstimateMode:   est.Spec.Mode,
            GapMinutes:     int(est.Spec.GapMinutes),
            DataCh:         dataCh,
            MetricName:     est.Spec.MetricName,
            MetricTags:     est.Spec.MetricTags,
            BaseMetricName: est.Spec.BaseMetricName,
            BaseMetricTags: est.Spec.BaseMetricTags,
            MetricProvider: mpconfig.ConvertMetricProvider(est.Spec.Provider.DeepCopy()).ActiveProvider(),
        },
    }
    r.estimatorChs[req.String()] = dataCh
}
```

**処理内容:**
1. **データチャネルの作成** - バッファサイズ5のチャネル
2. **EstimateAdd操作の送信** - estimatorHandlerに新しいEstimatorを登録
3. **チャネルの保存** - estimatorChsマップに保存

**既存Estimatorの更新:**
```go
else {
    r.opeCh <- &EstimateOperation{
        Operator: EstimateUpdate,
        Target: EstimateTarget{
            ID:             req.String(),
            EstimateMode:   est.Spec.Mode,
            GapMinutes:     int(est.Spec.GapMinutes),
            MetricName:     est.Spec.MetricName,
            MetricTags:     est.Spec.MetricTags,
            BaseMetricName: est.Spec.BaseMetricName,
            BaseMetricTags: est.Spec.BaseMetricTags,
            MetricProvider: mpconfig.ConvertMetricProvider(est.Spec.Provider.DeepCopy()).ActiveProvider(),
        },
    }
}
```

**処理内容:**
- EstimateUpdate操作を送信
- 既存のEstimatorの設定を更新
- データチャネルは再利用

**検証対象要件:** 要件 4.1, 4.4（Estimator起動と調整モード）

### 5. データ送信フェーズ

**ConfigMapからのデータ読み込み:**
```go
key := est.Spec.BaseMetricName
var in interface{}
if d, ok := cm.BinaryData[key]; ok {
    in = d
}
if d, ok := cm.Data[key]; ok {
    in = d
}
var data []byte
switch v := in.(type) {
case string:
    data = []byte(v)
}
```

**データ送信:**
```go
if data != nil {
    log.V(LogicMessageLogLevel).Info("new data stored", "name", req.String(), "key", key)
    dataCh := r.estimatorChs[req.String()]
    dataCh <- data
}
```

**処理内容:**
1. ConfigMapからBaseMetricName（元のメトリクス名）をキーとしてデータを取得
2. BinaryDataまたはDataフィールドから読み込み
3. データが存在する場合、データチャネルに送信
4. バックグラウンドのEstimatorゴルーチンがデータを受信

**検証対象要件:** 要件 10.2（ConfigMap監視）

## EstimatorHandler の詳細

### ハンドラーの役割

**estimatorHandler関数:**
```go
func estimatorHandler(opeCh <-chan *EstimateOperation, log logr.Logger)
```

**処理内容:**
- 操作チャネル（opeCh）から操作を受信
- EstimateTargetのリストを管理
- 各EstimateTargetに対してゴルーチンを起動

### 操作の種類

**EstimateAdd（追加）:**
```go
case EstimateAdd:
    et := ope.Target
    et.estimatorStopCh = make(chan struct{})
    et.Logger = log
    go et.estimator()
    estimateTargets = append(estimateTargets, et)
```

**処理:**
1. 停止チャネルを作成
2. ロガーを設定
3. estimatorゴルーチンを起動
4. リストに追加

**EstimateUpdate（更新）:**
```go
case EstimateUpdate:
    patch := ope.Target
    for i, et := range estimateTargets {
        if et.ID == patch.ID {
            if err := et.updateEstimateTarget(&patch); err != nil {
                log.V(LogicMessageLogLevel).Info("update estimator error", "error_msg", err)
            }
            
            close(et.estimatorStopCh)
            et.estimatorStopCh = make(chan struct{})
            go et.estimator()
            estimateTargets[i] = et
            break
        }
    }
}
```

**処理:**
1. IDで対象のEstimateTargetを検索
2. 設定を更新
3. 既存のゴルーチンを停止
4. 新しい停止チャネルを作成
5. 新しいゴルーチンを起動

**EstimateRemove（削除）:**
```go
case EstimateRemove:
    for i, et := range estimateTargets {
        if et.ID == ope.Target.ID {
            close(et.estimatorStopCh)
            estimateTargets = append(estimateTargets[:i], estimateTargets[i+1:]...)
            break
        }
    }
}
```

**処理:**
1. IDで対象のEstimateTargetを検索
2. 停止チャネルをクローズ（ゴルーチンに停止を通知）
3. リストから削除

**検証対象要件:** 要件 4.1（Estimatorライフサイクル管理）

## Estimator実行ロジックの詳細

### estimatorメソッドのフロー

```
1. データ受信（時系列予測データ）
   ↓
2. タイムスタンプをGapMinutesだけシフト
   ↓
3. 現在時刻より前のデータを削除
   ↓
4. 最初の要素を確認
   ↓
5. 送信時刻まで待機
   ↓
6. 過去の実際のメトリクスに基づいてyhatを調整（adjustモードの場合）
   ↓
7. メトリクスプロバイダにデータを送信
   ↓
8. 次のデータポイントへ
```

### データ受信と処理

**CSV形式のデータ読み込み:**
```go
newData, err := readEstimateDataAsCSV(bytes.NewReader(d))
```

**CSV形式:**
```csv
timestamp,yhat,yhat_upper,yhat_lower
1582253873,176.89,244.83,50.89
1582253933,126.80,251.26,8.17
```

**EstimateDatum構造体:**
```go
type EstimateDatum struct {
    UnixTime         int64   // 元のタイムスタンプ
    EstimateUnixTime int64   // 調整後のタイムスタンプ（GapMinutesだけシフト）
    YHat             float64 // 予測値
    UpperYHat        float64 // 予測値の上限
    LowerYHat        float64 // 予測値の下限
}
```

**タイムスタンプのシフト:**
```go
for i := range newData {
    newData[i].EstimateUnixTime = newData[i].UnixTime - int64(et.GapMinutes)*60
}
```

**目的:**
- 予測データは未来の時刻を持つ
- GapMinutesだけ前にシフトして、現在時刻に近づける
- 例: GapMinutes=10の場合、10分前にシフト
  - 元のタイムスタンプ: 2024-01-15 10:00:00
  - シフト後: 2024-01-15 09:50:00

**データのマージ:**
```go
tmpData := joinEstimateData(newData, data)
```

**joinEstimateData関数:**
- 新しいデータと既存のデータをマージ
- 新しいデータが優先される（古いデータは上書き）
- タイムスタンプでソート

**古いデータの削除:**
```go
now := time.Now().Unix()
for i, ed := range tmpData {
    if ed.EstimateUnixTime > now {
        data = tmpData[i:]
        position = 0
        break
    }
}
```

**検証対象要件:** 要件 2.4, 8.1, 10.3（メトリクス送信タイミングとデータマージ）

### メトリクス調整メカニズム

**adjustYHatメソッド:**
```go
func (currEd *EstimateDatum) adjustYHat(prevEd *EstimateDatum, actualValue float64) float64
```

**調整ロジック:**

1. **データ整合性チェック:**
   ```go
   if !(prevEd.UpperYHat >= prevEd.YHat &&
        prevEd.YHat >= prevEd.LowerYHat &&
        currEd.UpperYHat >= currEd.YHat &&
        currEd.YHat >= currEd.LowerYHat) {
       return adjusted
   }
   ```

2. **実際の値が予測値より高い場合（上方向調整）:**
   ```go
   if actualValue > prevEd.YHat {
       upperWidth := float64(prevEd.UpperYHat - prevEd.YHat)
       mag := math.Min(upperWidth, float64(actualValue-prevEd.YHat)) / upperWidth
       adjusted += mag * float64(currEd.UpperYHat-currEd.YHat)
   }
   ```
   
   **計算式:**
   - `upperWidth`: 前回の予測値と上限の差
   - `mag`: 調整の大きさ（0.0〜1.0）
   - `adjusted`: 現在の予測値 + (mag × 現在の予測値と上限の差)

3. **実際の値が予測値より低い場合（下方向調整）:**
   ```go
   else {
       lowerWidth := float64(prevEd.YHat - prevEd.LowerYHat)
       mag := math.Min(lowerWidth, float64(prevEd.YHat-actualValue)) / lowerWidth
       adjusted -= mag * float64(currEd.YHat-currEd.LowerYHat)
   }
   ```

**調整の適用:**
```go
if position != 0 && et.EstimateMode == string(AdjustMode) {
    currData := data[position]
    
    // 過去のデータを検索
    if d := pastDatumQueue.seekByUnixTime(time.Now().Unix()); d != nil {
        prevData = *d
        prevY, err = et.MetricProvider.Fetch(
            et.MetricProvider.AddSumAggregator(et.BaseMetricName),
            prevData.UnixTime,
            et.BaseMetricTags,
            nil,
        )
    }
    
    // 上方向の調整のみ適用
    if yhat := currData.adjustYHat(&prevData, prevY); yhat > adjustedYHat {
        adjustedYHat = yhat
    }
}
```

**重要な特徴:**
- **上方向のみの調整**: プロアクティブスケーリングを維持するため
- **過去データの使用**: PastEstimateDatumQueueから適切なデータを検索
- **実際のメトリクス取得**: MetricProviderから実際の値を取得

**検証対象要件:** 要件 4.1, 4.2, 4.3, 4.5（メトリクス調整メカニズムと上方向のみの調整）

### メトリクス送信

**送信するメトリクス:**
```go
sendMap := map[string]float64{
    et.MetricName:            adjustedYHat,
    et.MetricName + ".raw":   data[position].YHat,
    et.MetricName + ".upper": data[position].UpperYHat,
    et.MetricName + ".lower": data[position].LowerYHat,
}
```

**メトリクス名の例:**
- `ake.ihpa.forecasted_kubernetes_cpu_usage_total`: 調整後の予測値
- `ake.ihpa.forecasted_kubernetes_cpu_usage_total.raw`: 元の予測値
- `ake.ihpa.forecasted_kubernetes_cpu_usage_total.upper`: 予測値の上限
- `ake.ihpa.forecasted_kubernetes_cpu_usage_total.lower`: 予測値の下限

**送信処理:**
```go
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
```

**送信パラメータ:**
- `metricName`: メトリクス名
- `EstimateUnixTime`: タイムスタンプ（GapMinutesだけシフト済み）
- `datapoint`: メトリクス値
- `MetricTags`: タグ（一意性フィルタ）
- `metricUnitReference`: 元のメトリクス名（単位の参照用）

**過去データキューへの追加:**
```go
pastDatumQueue.enqueue(&data[position])
```

**目的:**
- 次回の調整計算で使用
- 最大288個のデータポイントを保持（5分間隔で24時間分）

**検証対象要件:** 要件 2.4, 5.4（メトリクス送信と命名規則）

### 待機時間の計算

**次のデータポイントまでの待機:**
```go
if len(data) > position {
    now := time.Now().Unix()
    waitTime = time.Duration(data[position].EstimateUnixTime - now)
}
```

**特徴:**
- 次のデータポイントのEstimateUnixTimeまで待機
- 正確なタイミングでメトリクスを送信
- 例: 次のデータポイントが10分後の場合、10分待機

**データがない場合:**
```go
else {
    waitTime = time.Duration(5)
}
```

**目的:**
- 新しいデータが到着するまで5秒ごとにチェック
- データチャネルからの受信を確認

## PastEstimateDatumQueue の詳細

### キューの役割

**目的:**
- 過去の予測データを保存
- 調整計算で使用する適切なデータポイントを検索

**容量:**
- 288個のデータポイント（5分間隔で24時間分）

### キューの操作

**enqueue（追加）:**
```go
func (q *PastEstimateDatumQueue) enqueue(d *EstimateDatum) {
    if d == nil {
        return
    }
    *q = append(*q, *d)
}
```

**dequeue（取り出し）:**
```go
func (q *PastEstimateDatumQueue) dequeue() *EstimateDatum {
    if len(*q) == 0 {
        return nil
    }
    var d EstimateDatum
    *q, d = (*q)[1:], (*q)[0]
    return &d
}
```

**peek（先頭を確認）:**
```go
func (q *PastEstimateDatumQueue) peek() *EstimateDatum {
    if len(*q) == 0 {
        return nil
    }
    var d EstimateDatum
    d = (*q)[0]
    return &d
}
```

**seekByUnixTime（時刻で検索）:**
```go
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
```

**処理内容:**
- 指定された時刻より前の最後のデータポイントを返す
- 古いデータポイントをキューから削除
- 例: 現在時刻が10:00の場合、09:55のデータポイントを返す

**使用例:**
```go
if d := pastDatumQueue.seekByUnixTime(time.Now().Unix()); d != nil {
    prevData = *d
    prevY, err = et.MetricProvider.Fetch(...)
}
```

## データマージロジック

### joinEstimateData関数

```go
func joinEstimateData(newEds, oldEds []EstimateDatum) []EstimateDatum
```

**処理内容:**

1. **ソート:**
   ```go
   sort.Slice(newEds, func(i, j int) bool {
       return newEds[i].EstimateUnixTime < newEds[j].EstimateUnixTime
   })
   sort.Slice(oldEds, func(i, j int) bool {
       return oldEds[i].EstimateUnixTime < oldEds[j].EstimateUnixTime
   })
   ```

2. **空チェック:**
   ```go
   if len(newEds) == 0 {
       return oldEds
   }
   if len(oldEds) == 0 {
       return newEds
   }
   ```

3. **重複しない場合:**
   ```go
   if newEds[0].EstimateUnixTime > oldEds[len(oldEds)-1].EstimateUnixTime {
       return append(oldEds, newEds...)
   }
   ```

4. **重複する場合:**
   ```go
   frontEds := oldEds
   newFirstEstimateUnixTime := newEds[0].EstimateUnixTime
   for i := range oldEds {
       if oldEds[i].EstimateUnixTime >= newFirstEstimateUnixTime {
           frontEds = oldEds[:i]
           break
       }
   }
   return append(frontEds, newEds...)
   ```

**マージの原則:**
- 新しいデータが優先される
- 古いデータのうち、新しいデータより前の部分のみ保持
- 重複する部分は新しいデータで上書き

**検証対象要件:** 要件 10.3（データマージロジック）

## エラーハンドリング

### リソース取得エラー
- NotFoundエラー: クリーンアップ処理を実行して正常終了
- その他のエラー: エラーを返してReconcileループが再試行

### データ読み込みエラー
- CSV解析エラー: ログに記録して次のデータを待機
- 不正なデータ形式: スキップして処理を継続

### メトリクス送信エラー
- 送信失敗: ログに記録して次のメトリクスを送信
- 一時的なエラーから自動復旧

## ログ出力

### 主要なログメッセージ

**Estimator追加:**
```go
log.V(LogicMessageLogLevel).Info("estimator added", "id", req.String(), "mode", est.Spec.Mode,
    "gapMinutes", est.Spec.GapMinutes, "metricName", est.Spec.MetricName)
```

**メトリクス送信:**
```go
et.V(LogicMessageLogLevel).Info("send metrics",
    "metricName", et.MetricName,
    "timestamp", time.Unix(data[position].EstimateUnixTime, 0).String(),
    "yhat", data[position].YHat,
    "adjusted_yhat", adjustedYHat)
```

## SetupWithManager

```go
func (r *EstimatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
    r.estimatorChs = make(map[string]chan<- []byte)
    
    opeCh := make(chan *EstimateOperation)
    r.opeCh = opeCh
    go estimatorHandler(opeCh, r.Log.WithName("Estimator"))
    
    return ctrl.NewControllerManagedBy(mgr).
        For(&ihpav1beta2.Estimator{}).
        Owns(&corev1.ConfigMap{}).
        Complete(r)
}
```

**処理内容:**
1. estimatorChsマップを初期化
2. 操作チャネルを作成
3. estimatorHandlerゴルーチンを起動
4. Estimatorリソースを監視
5. ConfigMapの変更も監視（Owns）

**Ownsの効果:**
- ConfigMapが変更されるとReconcileループがトリガーされる
- FittingJobが予測データを書き込むと自動的に検出

## RBAC権限

```go
// +kubebuilder:rbac:groups=ihpa.ake.cyberagent.co.jp,resources=estimators,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ihpa.ake.cyberagent.co.jp,resources=estimators/status,verbs=get;update;patch
```

**権限:**
- Estimatorリソースの完全な管理権限
- ConfigMapの権限はIHPA Controllerから継承

## 実装の特徴

### 非同期処理
- バックグラウンドゴルーチンでメトリクス送信
- Reconcileループをブロックしない
- 複数のEstimatorが並列実行

### チャネルベースの通信
- データチャネルで予測データを受け渡し
- 操作チャネルでライフサイクル管理
- ゴルーチン間の安全な通信

### 調整メカニズムの柔軟性
- rawモード: 調整なし
- adjustモード: 過去の精度に基づく調整
- 上方向のみの調整でプロアクティブスケーリングを維持

## まとめ

Estimator Controllerは、予測データをメトリクスプロバイダに送信し、過去の精度に基づいて調整を行います。

**主要な責務:**
1. ConfigMap管理（予測データ保存用）
2. データ監視（ConfigMapの変更検出）
3. メトリクス送信（適切なタイミング）
4. 調整機能（過去の精度に基づく）

**設計の特徴:**
- 非同期処理
- チャネルベースの通信
- 柔軟な調整メカニズム
- 自動クリーンアップ

**検証対象要件:**
- 要件 4.1, 4.2, 4.3: メトリクス調整メカニズム
- 要件 4.4: 調整モードサポート
- 要件 4.5: 上方向のみの調整
- 要件 2.4: メトリクス配信タイミング
- 要件 8.1: GapMinutes設定
- 要件 10.1, 10.2, 10.3: データ永続化と管理
