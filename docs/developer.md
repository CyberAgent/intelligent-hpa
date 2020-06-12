# 開発者向けドキュメント

## マニフェストの生成

マニフェストを生成するときは下記のようにします。Namespace を `kube-system` で作るように設定していますが、マニフェストの先頭に `ihpa-system` という定義が追加されてしまうため不要なものを入れたくない場合は消してください。

```bash
IMG="cyberagentoss/intelligent-hpa-controller:latest"

cd ihpa-controller/config/manager && kustomize edit set image controller=${IMG}
cd - && kustomize build ihpa-controller/config/default > intelligent-hpa.yaml
```

#### NOTES

Kubernetes 1.18 から`x-kubernetes-list-type` の Validation が入るようになりました ([本家](https://github.com/kubernetes/kubernetes/blob/master/CHANGELOG/CHANGELOG-1.18.md#other-api-changes), [Qiita](https://qiita.com/Ladicle/items/bbe2a62aba85d083283d#other))。チェック内容はフィールドの一意性やデフォルト値が設定されているかどうかなどです。そのため古い Kubebuilder (少なくとも v2.3.1) だと `make manifests` による CRD 生成時に Validation の通らないマニフェストが生成されてしまいます。本環境では `containers` や `initContainers` 内にある Port のプロトコルを指定する部分のデフォルト値が指定されていないためエラーとなります。

```
The CustomResourceDefinition "fittingjobs.ihpa.ake.cyberagent.co.jp" is invalid:
* spec.validation.openAPIV3Schema.properties[spec].properties[template].properties[spec].properties[template].properties[spec].properties[containers].items.properties[ports].items.properties[protocol].default: Required value: this property is in x-kubernetes-list-map-keys, so it must have a default or be a required property
* spec.validation.openAPIV3Schema.properties[spec].properties[template].properties[spec].properties[template].properties[spec].properties[initContainers].items.properties[ports].items.properties[protocol].default: Required value: this property is in x-kubernetes-list-map-keys, so it must havea default or be a required property
```

これを回避するために下記のようにデフォルト値を設定します。上記のエラーだと 2 箇所こういった箇所が存在するので同じように直します。

```
                                          protocol:
---------------------->                     default: TCP
                                            description: Protocol for port. Must be
                                              UDP, TCP, or SCTP. Defaults to "TCP".
                                            type: string
                                        required:
                                        - containerPort
                                        type: object
                                      type: array
                                      x-kubernetes-list-map-keys:
                                      - containerPort
                                      - protocol
                                      x-kubernetes-list-type: map
```

この問題は [PR](https://github.com/kubernetes-sigs/controller-tools/pull/440) が作られているためこれがマージされれば直るはずです。

## Fitting Job

Datadog のメトリクス取得や、モデルの学習は全て `./fittingjob` で行っています。下記のような設定ファイルを用意して `./train.py config.json` で実行します。

```json
{
    "provider":{
        "datadog":{
            "apikey":"xxx",
            "appkey":"yyy"
        }
    },
    "targetMetricsName":"nginx.net.request_per_s",
    "targetTags":{
        "kube_namespace":"loadtest",
        "kube_deployment":"nginx"
    },
    "seasonality":"auto",
    "dataConfigMapName":"target-cm",
    "dataConfigMapNamespace":"loadtest",
    "changePointDetection":{
        "percentageThreshold":50,
        "windowSize":100,
        "trajectoryRows":50,
        "trajectoryFeatures":5,
        "testRows":50,
        "testFeatures":5,
        "lag":288
    },
    "customConfig":""
}
```

この設定ファイルは IHPA リソースを作成すると自動で生成されるため基本的に気にする必要はありません。カスタムイメージを実装する場合はこの設定を使用して予測・ConfigMap (`dataConfigMapName` と `dataConfigMapNamespace` に指定されているもの) への書き込みを行ってください。

現在のモデルにはメトリクスの傾向が変わった点を検知して学習データに含まないようにする機能があります。この変化点検知は `changePointDetection` の項目によって調整されます。

### Build

ビルドは下記のように行います。

```
make fittingjob
```

### Development

Prophet に必要なパッケージをインストールします。

```bash
apt install -y build-essential python-dev python3-dev
```

依存パッケージをインストールします。

```bash
pip install pipenv

cd ./fittingjob
pipenv install
```

IHPA は下記のようなカラムを含んだ CSV が ConfigMap に入れられることを期待しています。カラムは順不同で不要なものは無視されます。

```csv
timestamp,yhat,yhat_upper,yhat_lower
1582253873,176.89,244.83,50.89
1582253933,126.80,251.26,8.17
1582253993,134.48,268.67,75.97
```

`timestamp` と `yhat` の値の組がプロバイダに送信されます。そのとき現在のメトリクスと、それを予測したメトリクスとのズレに応じて `yhat_upper`, `yhat_lower` 間で調整されます。

## IHPA Controller

IHPA の Custom Controller は `./ihpa-controller` 内にあります。フレームワークは Kubebuilder を使用しているので実装の作法や Makefile などの各種機能は[公式ドキュメント](https://book.kubebuilder.io/)を参照してください。

### Build

ビルドは下記のように行います。

```
make controller
```

### Kubebuilder

scaffold の生成は下記コマンドで行いました。

```bash
kubebuilder init --domain ake.cyberagent.co.jp --owner "SIA Platform Team" --repo github.com/cyberagent-oss/intelligent-hpa/ihpa-controller
kubebuilder create api --group ihpa --version v1beta2 --kind IntelligentHorizontalPodAutoscaler
kubebuilder create api --group ihpa --version v1beta2 --kind FittingJob
kubebuilder create api --group ihpa --version v1beta2 --kind Estimator
```
