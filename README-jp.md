# Intelligent HPA

Intelligent HPA は HPA の参照しているワークロードのメトリクスを予測して事前にスケーリングさせるカスタムコントローラーです。

- [Prerequisite](#prerequisite)
- [Usage](#usage)
- [Installation](#installation)
- [Fitting Job](#fitting-job)
- [Other docs](#other-docs)

## Prerequisite

メトリクスを予測・格納するためのプロバイダーを選択します。

### Datadog

Datadog Agent と Datadog Cluster Agent をインストールします。詳細は [Datadog 公式 Helm チャートのドキュメント](https://docs.datadoghq.com/ja/containers/kubernetes/installation/?tab=helm) を参照してください。すでに Datadog がインストールされている場合は、この後の upgrade 手順を実行してください。

```sh
KUBE_SYSTEM_UID=$(kubectl get namespace kube-system -o jsonpath='{.metadata.uid}')

helm repo add datadog https://helm.datadoghq.com
helm repo update

helm install datadog datadog/datadog \
  --set datadog.apiKey=xxx \
  --set datadog.appKey=yyy \
  --set clusterAgent.metricsProvider.enabled=true \
  --set datadog.checksCardinality=orchestrator \
  --set-string "datadog.tags[0]=kube_system_uid:${KUBE_SYSTEM_UID}"
```

IHPA では外部メトリクスプロバイダー、API キー、メトリクス読み取り権限を持つ APP キーが必要です。Agent の認証情報とは別に、[Usage](#usage) に従って IHPA の `metricProvider` にも Datadog 認証情報を設定してください。

すでに Datadog がインストールされている場合は以下の upgrade コマンドを使ってください。 `--reuse-values` が現在の credentials を保持し、追加の設定を反映します。

```sh
KUBE_SYSTEM_UID=$(kubectl get namespace kube-system -o jsonpath='{.metadata.uid}')

helm repo update

helm upgrade datadog datadog/datadog \
  --reuse-values \
  --set clusterAgent.metricsProvider.enabled=true \
  --set datadog.checksCardinality=orchestrator \
  --set-string "datadog.tags[0]=kube_system_uid:${KUBE_SYSTEM_UID}"
```

`datadog.tags` は list のため、`--reuse-values` を使っても既存のタグは引き継がれず、指定した値で list 全体が置き換わります。既存のタグがある場合は、upgrade の前に現在の値を確認し、`tags[0]`, `tags[1]`, ... として全て指定してください。

```sh
helm get values datadog -o yaml
```

タグが多い場合は、一度 yaml ファイルに書き出してから追記することも検討してください。

また、設定変更により Agent は再起動する場合があります。Helm 管理でない Agent は、管理元の設定に同等の値を反映するか、対応する patch を適用してください。

IHPA の Datadog 接続先は、controller manager と FittingJob の学習コンテナの両方に `DD_SITE`（例: `us5.datadoghq.com`）を設定して選択します。Job は controller の環境変数を引き継がないため、FittingJob / IHPA の job patch 設定にも `env` を指定してください。未指定または空の場合は US1（`datadoghq.com`）を使用します。

指定できる値は `datadoghq.com`、`us3.datadoghq.com`、`us5.datadoghq.com`、`datadoghq.eu`、`ap1.datadoghq.com`、`ap2.datadoghq.com`、`uk1.datadoghq.com` です。GovCloud を含むその他の値は HTTP リクエスト前に拒否します。`DD_SITE` を設定する前に、controller と FittingJob の両方のイメージを再ビルドしてください。

### Prometheus

IHPAはメトリクスプロバイダとしてPrometheusをサポートしています。`metricProvider`セクションに`prometheus`プロバイダソースを設定します：

```yaml
metricProvider:
  name: prometheus
  prometheus:
    url: "http://prometheus-server:9090"
    pushgatewayUrl: "http://prometheus-pushgateway:9091"
```

- `url`: メトリクスのクエリに使用するPrometheusサーバーのURL
- `pushgatewayUrl`: 予測メトリクスの送信に使用するPrometheus PushgatewayのURL

### Workload editor authorization

The default installation requires Kubernetes 1.30 or newer and installs a
fail-closed ValidatingAdmissionPolicy and binding. Creating or updating a
FittingJob or IHPA (either API version) requires `create` on core `pods` in the
same namespace, in addition to the CR permission. The supplied CR editor roles
alone no longer permit workload creation. Grant Pod creation only to trusted
workload operators; these resources accept executable images, commands,
service accounts and volumes. The controller also needs this permission to
create FittingJobs on behalf of an authorized IHPA writer.

Upgrade by applying the complete `manifests/intelligent-hpa.yaml` (or
`config/default`), including both admission resources, before allowing CR
writers access. CRD-only installation does not install this protection.
Review existing FittingJob/IHPA resources and their CronJobs before restarting
the controller: admission checks only new writes and cannot establish who
previously authored a stored template. Custom installations must retain both
the policy and its Deny binding.


## Usage

Intelligent HPA のマニフェストについて説明します。

- `estimator`
    - Estimator リソースに関係する設定をします
    - `gapMinutes`
        - 予測値をずらす時間 (分) を指定します
        - 何分先のメトリクスを予測するかどうかと同義です
        - 例えば `5` にすると 5 分先のメトリクスをもとに Pod がスケールします
            - 現在のメトリクスも参照されるため、実負荷が高いときに予測メトリクスが減ってもスケールインは発生しません
        - default: `5`
    - `mode`
        - 予測メトリクスをプロバイダに送信する際の調整モードを指定します
        - `adjust` にすると直前の予測のずれをもとに現在のメトリクスを調整して送信します
        - `raw` にすると与えられた予測値をそのまま送信します
        - 許容値: `adjust`, `raw` (default: `adjust`)
- `metricProvider`
    - メトリクスを取得・送信するプロバイダを設定します
    - Datadog と Prometheus に対応しています
    - `aggregation`
        - メトリクスのクエリ時に使用する集計関数を指定します
        - 許容値: `sum`, `min`, `max`, `count`, `avg` (default: `sum`)
- `template`
    - HPA のマニフェストを記述します
    - HPA から移行する場合はそのままここにコピーしてください
    - 現在は `Resource`, `External` メトリクスのみに対応しています
- `fittingJob`
    - FittingJob リソースに関係する設定をします
    - `.spec.template.spec.metrics` 内に記述できます
    - `seasonality`
        - メトリクスの季節性を固定します
        - 未指定の場合は自動で判別されるため基本的には指定する必要はありません
        - 許容値: `auto`, `daily`, `weekly`, `yearly` (default: `auto`)
    - `executeOn`
        - 学習ジョブを実行する時刻を指定します
        - 例えば `12` にすると 12 時 N 分 (N はランダム) にジョブが実行されます
        - default: `4`
    - `changePointDetectionConfig`
        - 学習ジョブが訓練データを選択する際の変化点検知に使用されるしきい値などのパラメータを指定します
          - `percentageThreshold` (default: `50`)
          - `windowSize` (default: `100`)
          - `trajectoryRows` (default: `50`)
          - `trajectoryFeatures` (default: `5`)
          - `testRows` (default: `50`)
          - `testFeatures` (default: `5`)
          - `lag` (default: `288`)
        - パラメータの詳細は [Fitting Job](#fitting-job) を参照してください
    - `customConfig`
        - 学習ジョブに渡すことのできる任意の文字列です
    - `image`
        - 学習ジョブに使用するコンテナイメージを指定します
        - default: `cyberagentoss/intelligent-hpa-fittingjob:latest`
    - `imagePullSecrets`
        - 学習ジョブのイメージを Pull する際の Secret を指定します
    - その他パラメータについては[こちら](https://github.com/cyberagent-oss/intelligent-hpa/blob/master/ihpa-controller/api/v1beta2/fittingjob_types.go#L63-L86)を確認してください

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
    aggregation: sum
    datadog:
      apikey: xxx
      appkey: yyy
      # SecretまたはConfigMapからキーを参照することも可能です:
      # keysFrom:
      #   - secretRef:
      #       name: datadog-keys
      # Secret/ConfigMapには "APIKey" と "APPKey" というキーを含める必要があります
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

Apply したあと、該当の HPA を describe することでメトリクスの状況がわかります (デフォルトでは v1 の HPA が見えてしまうため `kubectl describe hpa.v2.autoscaling xxx` のようにして確認する必要があります)。`ake.ihpa` で始まるメトリクス名が IHPA で予測しているメトリクス名になります。メトリクスによってはプロバイダのスケール値によっては異質な値になりますが問題ありません (HPA へメトリクスが渡される際にプロバイダのスケール値が考慮されない問題への対処です)。

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

学習は `executeOn` に指定した時間に実行されるため Apply 直後は予測メトリクスがありません。最大 1 日待機すると学習が始まりますが、すでに十分なメトリクスがあり、直ちに学習ジョブを実行したい場合は下記のように手動で CronJob を起動することが可能です。CronJob 名は環境に合わせて変更してください。

```sh
kubectl create job -n loadtest --from cronjob/ihpa-nginx-nginx-net-request-per-s manual-train
```

## Installation

マニフェストを直接 `create` します (現在 FittingJob の CRD 定義が大きすぎるため `apply` だと容量制限に引っかかります)。

```
kubectl create -f manifests/intelligent-hpa.yaml
```

## Fitting Job

デフォルトの学習イメージでは Prophet を用いた時系列予測を行っています。チューニング無しでも基本的な予測ができます。

また変化点検知も組み込んでいます。これはワークロードが変化するなどメトリクスの傾向が大きく変わった際に、その学習データをドロップする目的で使用しています。こちらはチューニングが必須なのでこだわりがない場合は設定しなくても構いません。パラメータの詳細は下記になります。

|parameter|description|
|:-:|:-|
|percentageThreshold|変化したと判断するしきい値を設定します。1-99 の範囲のパーセンテージ表記です|
|windowSize|サブ時系列の行列を作る際の幅 (列) を指定します。これが小さいとその範囲を比べて異常値を計算するため変化に敏感になります|
|trajectoryRows|履歴行列 (サブ時系列) の行数を指定します。|
|trajectoryFeatures|履歴行列を特異値分解した際の特徴量選択数を指定します。これによりノイズとなっている情報を取り除いて精度を高めることができます。この値は windowSize よりも小さい必要があります。|
|testRows|テスト行列 (サブ時系列) の行数を指定します。|
|testFeatures|テスト行列を特異値分解した際の特徴量選択数を指定します。|
|lag|履歴行列とテスト行列をずらす幅を指定します。同じ時間帯を比較できると誤った異常検知を防げるはずなので Seasonality に合わせるのが良いと思われます。基本的には Daily 想定で 288 (5 分間隔でメトリクスを取得するのでその 1 日分) ずらせば良いです。|

チューニング用の [Jupyter Notebook](./fittingjob/change_point_detection_tuning.ipynb) を用意しています。

## Other docs

- [アーキテクチャ](./docs/architecture.md)
- [Estimator とメトリクス調整の仕組み](./docs/estimator.md)
- [学習ジョブの仕組み](./docs/fittingjob.md)
- [開発者向け](./docs/developer.md)

### Datadog の認証情報の参照

インラインの `apikey` / `appkey` より `datadog.keysFrom.secretRef` を使用してください。
生成する FittingJob / Estimator には参照を保持し、Estimator は実行用のコピーでのみ
値を解決します。学習 Pod には `secretKeyRef` / `configMapKeyRef` で `APIKey` と
`APPKey` だけを渡し、参照元の他のキーは渡しません。キーごとに、そのキーを含む
最後の参照元を優先し、参照元の optional 設定を維持します。学習設定全体の出力は
削除しています。controller と学習イメージの両方を再ビルド・更新してください。

派生リソースを再生成する前に、Secret 参照を使う既存の全 IHPA からインラインの
`apikey` / `appkey` フィールドを削除し、元のソースマニフェストに基づく Secret
参照だけの設定に戻してください。旧 controller は解決した認証情報を IHPA 自体に
保存する場合があり、そのフィールドを残した live オブジェクトから再生成すると
平文の認証情報が再びコピーされます。修正した IHPA から FittingJob / Estimator /
学習設定 ConfigMap を再生成し、認証情報が残る可能性のある古い Job / Pod / ログを
削除してください。以前に露出した Datadog キーもローテーションしてください。
再調整で派生リソースの spec と設定は上書きされますが、過去のログや API 監査記録は
削除されません。インラインの認証情報は互換性のため利用可能ですが、その保存先で
読み取り可能になります。認証情報の秘匿には Secret 参照を使用してください。
