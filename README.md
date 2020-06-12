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

Datadog Agent と Datadog Cluster Agent をインストールします。すでにインストールしている人はこの手順を飛ばしてください。

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

すでにインストールしている場合はクラスタの一意識別性のために `kube-system` Namespace の UID をタグに追加してください。これは `cpu`, `memory` といった Resource 系のメトリクスに一意性を与えるために必要になります (External 系を使用する場合はユーザーがタグを付与するので不要ですがつけておいた方が無難です)。タグを追加すると Datadog Agent が再起動します。

```sh
TMP_DD_TAGS=$(kubectl get ds <your_datadog_agent> -o jsonpath='{.spec.template.spec.containers[*].env[?(@.name == "DD_TAGS")].value}')
KUBE_SYSTEM_UID=$(kubectl get namespace kube-system -o jsonpath='{.metadata.uid}')

TMP_DD_TAGS="${TMP_DD_TAGS} kube_system_uid:${KUBE_SYSTEM_UID}"

kubectl patch ds <your_datadog_agent> -p "{\"spec\":{\"template\":{\"spec\":{\"containers\":[{\"name\":\"agent\",\"env\":[{\"name\":\"DD_TAGS\",\"value\":\"${TMP_DD_TAGS}\"}]}]}}}}"

# if error occurred, please change containers[0].name replace to "datadog"
```

### Prometheus

いつか

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
    - 現在は Datadog のみ対応しています
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

Apply したあと、該当の HPA を describe することでメトリクスの状況がわかります (デフォルトでは v1 の HPA が見えてしまうため `kubectl describe hpa.v2beta2.autoscaling xxx` のようにして確認する必要があります)。`ake.ihpa` で始まるメトリクス名が IHPA で予測しているメトリクス名になります。メトリクスによってはプロバイダのスケール値によっては異質な値になりますが問題ありません (HPA へメトリクスが渡される際にプロバイダのスケール値が考慮されない問題への対処です)。

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

マニフェストを Apply します。

```
kubectl apply -f manifests/intelligent-hpa.yaml
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
