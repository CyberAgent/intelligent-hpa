# FittingJob Controller 実装仕様書

## 概要

FittingJobReconciler（FittingJob Controller）は、FittingJobカスタムリソースを監視し、機械学習モデルの訓練と予測を実行するために必要なKubernetesリソースを生成・管理するコントローラーです。

## アーキテクチャ

### コントローラーの責務

FittingJob Controllerは以下のリソースのライフサイクルを管理します：

1. **ConfigMap（設定用）** - FittingJobコンテナの設定を保存
2. **CronJob** - 定期的にFittingJobを実行

### コンポーネント構成

```
FittingJobReconciler
├── Client (Kubernetes API Client)
├── Logger (構造化ログ)
└── Scheme (Kubernetes スキーマ)
```

## Reconcileループの詳細フロー

### 1. リソース取得フェーズ

```go
func (r *FittingJobReconciler) Reconcile(req ctrl.Request)
```

**処理内容:**
- FittingJobリソースをNamespacedNameで取得
- リソースが見つからない場合（削除された場合）、エラーを無視して終了
- エラーは`client.IgnoreNotFound(err)`で処理

**検証対象要件:** 要件 6.1（FittingJobリソース管理）

### 2. ジェネレーター初期化フェーズ

```go
g, err := NewFittingJobGenerator(&fj)
```

**ジェネレーター初期化処理:**
- FittingJobリソースを受け取り、ジェネレーターインスタンスを作成
- エラーチェック（現在の実装では常に成功）

**検証対象要件:** 要件 6.1（リソース生成の基盤）

### 3. ConfigMap生成・更新フェーズ

```go
cmResource, err := g.ConfigMapResource()
```

**ConfigMap生成ロジック:**

1. **メトリクスプロバイダ設定の変換:**
   ```go
   mpConfig := mpconfig.ConvertMetricProvider(&g.fj.Spec.Provider)
   ```

2. **FittingJobConfig構造体の構築:**
   ```go
   fittingJobConfig := &FittingJobConfig{
       MetricProvider:             *mpConfig,
       TargetMetricsName:          mpConfig.ActiveProvider().AddSumAggregator(g.fj.Spec.TargetMetric.Name),
       TargetTags:                 g.fj.Spec.TargetMetric.Selector.MatchLabels,
       Seasonality:                g.fj.Spec.Seasonality,
       ChangePointDetectionConfig: g.fj.Spec.ChangePointDetectionConfig,
       CustomConfig:               g.fj.Spec.CustomConfig,
       DataConfigMapName:          g.fj.Spec.DataConfigMap.Name,
       DataConfigMapNamespace:     g.fj.GetNamespace(),
   }
   ```

3. **JSON形式へのシリアライズ:**
   ```go
   configData, err := json.Marshal(fittingJobConfig)
   ```

4. **ConfigMapリソースの構築:**
   - 名前: `<fittingjob_name>-config`
   - データ: `config.json` キーにJSON設定を保存

**メトリクス名の処理:**
- `AddSumAggregator`メソッドで集約方法を追加
- 例: `kubernetes.cpu.usage.total` → `sum:kubernetes.cpu.usage.total`
- 目的: 訓練データ取得時に適切な集約を適用

**リソース作成/更新:**
- 存在しない場合: `r.Create(ctx, cm)`
- 存在する場合: DataとBinaryDataを更新して `r.Update(ctx, cm)`

**検証対象要件:** 要件 6.1, 8.2, 8.3（設定管理とカスタム設定）

### 4. CronJob生成・更新フェーズ

```go
cjResource, err := g.CronJobResource()
```

**CronJob生成ロジック:**

1. **JobSpecの生成:**
   ```go
   jobSpec := *g.fj.Spec.JobPatchSpec.GenerateJobSpec()
   ```
   
   **JobPatchSpecからJobSpecへの変換:**
   - Job関連: ActiveDeadlineSeconds, BackoffLimit, Completions
   - Pod関連: Affinity, ImagePullSecrets, NodeSelector, ServiceAccountName, Tolerations, Volumes
   - Container関連: Args, Command, Env, EnvFrom, Image, ImagePullPolicy, Resources

2. **設定用Volumeの追加:**
   ```go
   volume := corev1.Volume{
       Name: "fittingjob-config",
       VolumeSource: corev1.VolumeSource{
           ConfigMap: &corev1.ConfigMapVolumeSource{
               LocalObjectReference: corev1.LocalObjectReference{Name: g.configMapName()},
           },
       },
   }
   volumeMount := corev1.VolumeMount{Name: volume.Name, MountPath: "/" + volume.Name}
   jobSpec.Template.Spec.Volumes = append(jobSpec.Template.Spec.Volumes, volume)
   ```

3. **RestartPolicyの設定:**
   ```go
   jobSpec.Template.Spec.RestartPolicy = corev1.RestartPolicyOnFailure
   ```
   
   **理由:**
   - Jobは失敗時に再試行が必要
   - OnFailureポリシーで自動リトライ

4. **Containerの設定:**
   ```go
   fittingContainer := jobSpec.Template.Spec.Containers[0]
   fittingContainer.Name = "fittingjob"
   if fittingContainer.Image == "" {
       fittingContainer.Image = DefaultImage
   }
   fittingContainer.VolumeMounts = append(fittingContainer.VolumeMounts, volumeMount)
   ```
   
   **デフォルトイメージ:**
   - `cyberagentoss/intelligent-hpa-fittingjob:latest`
   - ユーザーがカスタムイメージを指定しない場合に使用

5. **CronJobスケジュールの設定:**
   ```go
   Schedule: randomMinuteCronFormat(int(g.fj.Spec.ExecuteOn))
   ```
   
   **スケジュール生成ロジック:**
   ```go
   func randomMinuteCronFormat(hour int) string {
       if hour < 0 {
           hour = 0
       }
       hour %= 24
       rand.Seed(time.Now().UnixNano())
       return fmt.Sprintf("%d %d * * *", rand.Intn(60), hour)
   }
   ```
   
   **特徴:**
   - 指定された時刻（ExecuteOn）に実行
   - 分はランダム化（0-59分）
   - 目的: 複数のFittingJobが同時実行されるのを避ける
   - 例: ExecuteOn=4 → `23 4 * * *`（毎日4時23分に実行）

**リソース作成/更新:**
- 存在しない場合: `r.Create(ctx, cj)`
- 存在する場合: Specを更新して `r.Update(ctx, cj)`

**検証対象要件:** 要件 6.1, 6.2, 6.3（CronJob作成とスケジューリング）

## FittingJobConfig構造体の詳細

### 構造体定義

```go
type FittingJobConfig struct {
    MetricProvider             mpconfig.MetricProviderConfig          `json:"provider"`
    DumpPath                   string                                 `json:"dumpPath,omitempty"`
    TargetMetricsName          string                                 `json:"targetMetricsName"`
    TargetTags                 map[string]string                      `json:"targetTags"`
    Seasonality                string                                 `json:"seasonality"`
    DataConfigMapName          string                                 `json:"dataConfigMapName"`
    DataConfigMapNamespace     string                                 `json:"dataConfigMapNamespace"`
    ChangePointDetectionConfig ihpav1beta2.ChangePointDetectionConfig `json:"changePointDetection"`
    CustomConfig               string                                 `json:"customConfig"`
}
```

### フィールド説明

**MetricProvider:**
- メトリクスプロバイダの設定（Datadog、Prometheusなど）
- APIキー、APPキー、エンドポイントなどの認証情報を含む

**TargetMetricsName:**
- 訓練データ取得対象のメトリクス名
- 集約方法（sum）が追加される
- 例: `sum:kubernetes.cpu.usage.total`

**TargetTags:**
- メトリクスを特定するためのタグ
- 例: `{"kube_namespace": "default", "kube_deployment": "myapp"}`
- 目的: 複数のリソースから特定のリソースのメトリクスを取得

**Seasonality:**
- 時系列の季節性パターン
- 値: `daily`, `weekly`, `yearly`, `auto`
- デフォルト: `auto`（Prophetが自動検出）

**DataConfigMapName / DataConfigMapNamespace:**
- 予測結果を保存するConfigMapの名前と名前空間
- FittingJobコンテナがこのConfigMapに予測データを書き込む

**ChangePointDetectionConfig:**
- 変化点検知のパラメータ
- SST（Singular Spectrum Transformation）アルゴリズムの設定

**CustomConfig:**
- ユーザー定義のカスタム設定文字列
- 特殊な予測アルゴリズムのための追加設定

### 設定の使用方法

FittingJobコンテナは起動時に以下の手順で設定を読み込みます：

1. **ConfigMapのマウント:**
   - `/fittingjob-config/config.json` にマウント

2. **JSON設定の読み込み:**
   - Pythonコード内でJSONファイルをパース

3. **メトリクスプロバイダへの接続:**
   - 認証情報を使用してAPIに接続

4. **訓練データの取得:**
   - TargetMetricsNameとTargetTagsを使用してメトリクスを取得

5. **モデルの訓練:**
   - Seasonalityとその他のパラメータを使用してProphetモデルを訓練

6. **予測結果の保存:**
   - DataConfigMapNameで指定されたConfigMapに予測データを書き込む

## ChangePointDetectionConfig の詳細

### 構造体定義

```go
type ChangePointDetectionConfig struct {
    PercentageThreshold int32 `json:"percentageThreshold,omitempty"` // デフォルト: 50
    WindowSize          int32 `json:"windowSize,omitempty"`          // デフォルト: 100
    TrajectoryRows      int32 `json:"trajectoryRows,omitempty"`      // デフォルト: 50
    TrajectoryFeatures  int32 `json:"trajectoryFeatures,omitempty"`  // デフォルト: 5
    TestRows            int32 `json:"testRows,omitempty"`            // デフォルト: 50
    TestFeatures        int32 `json:"testFeatures,omitempty"`        // デフォルト: 5
    Lag                 int32 `json:"lag,omitempty"`                 // デフォルト: 288
}
```

### パラメータ説明

**PercentageThreshold (1-99):**
- 異常検知の閾値（パーセンテージ）
- この閾値を超えるデータポイントは訓練データから除外される
- 高い値: より多くのデータを保持（保守的）
- 低い値: より多くのデータを除外（積極的）

**WindowSize (≥1):**
- スライディングウィンドウの幅
- 部分時系列の長さを決定
- 大きい値: より長期的なパターンを検出
- 小さい値: より短期的な変化を検出

**TrajectoryRows (≥1):**
- 軌跡行列の行数
- SST分析の基礎となる行列のサイズ

**TrajectoryFeatures (≥1):**
- 左特異ベクトルの軌跡特徴数
- 主成分分析のような次元削減

**TestRows (≥1):**
- テスト行列の行数
- 変化点検知のためのテストウィンドウサイズ

**TestFeatures (≥1):**
- 左特異ベクトルのテスト特徴数
- テストデータの次元削減

**Lag (≥1):**
- 軌跡とテストのギャップ
- 時系列データのラグを考慮
- デフォルト288: 5分間隔で24時間分（288 = 24 * 60 / 5）

### 変化点検知の目的

**問題:**
- メトリクスデータには異常な期間が含まれる可能性がある
- デプロイ、障害、メンテナンスなどによる一時的な変化
- これらのデータで訓練すると予測精度が低下

**解決策:**
- SST（Singular Spectrum Transformation）アルゴリズムを使用
- 異常な期間を自動的に検出
- 訓練データから除外して品質を向上

**検証対象要件:** 要件 3.1, 3.2, 3.3, 3.4（変化点検知機能）

## JobPatchSpec の詳細

### 構造体定義

```go
type JobPatchSpec struct {
    // Job関連
    ActiveDeadlineSeconds *int64 `json:"activeDeadlineSeconds,omitempty"`
    BackoffLimit          *int32 `json:"backoffLimit,omitempty"`
    Completions           *int32 `json:"completions,omitempty"`

    // Pod関連
    Affinity           *corev1.Affinity              `json:"affinity,omitempty"`
    ImagePullSecrets   []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
    NodeSelector       map[string]string             `json:"nodeSelector,omitempty"`
    ServiceAccountName string                        `json:"serviceAccountName,omitempty"`
    Tolerations        []corev1.Toleration           `json:"tolerations,omitempty"`
    Volumes            []corev1.Volume               `json:"volumes,omitempty"`

    // Container関連
    Args            []string                    `json:"args,omitempty"`
    Command         []string                    `json:"command,omitempty"`
    Env             []corev1.EnvVar             `json:"env,omitempty"`
    EnvFrom         []corev1.EnvFromSource      `json:"envFrom,omitempty"`
    Image           string                      `json:"image,omitempty"`
    ImagePullPolicy corev1.PullPolicy           `json:"imagePullPolicy,omitempty"`
    Resources       corev1.ResourceRequirements `json:"resources,omitempty"`
}
```

### フィールド説明

**Job関連フィールド:**

- **ActiveDeadlineSeconds:**
  - Jobの最大実行時間（秒）
  - この時間を超えるとJobは終了される
  - 長時間実行を防ぐ

- **BackoffLimit:**
  - 失敗時の再試行回数
  - デフォルト: 6回
  - 一時的なエラーからの復旧

- **Completions:**
  - 成功とみなすために必要な完了数
  - デフォルト: 1回
  - FittingJobは通常1回の完了で十分

**Pod関連フィールド:**

- **Affinity:**
  - Podのアフィニティルール
  - 特定のノードへの配置制御

- **ImagePullSecrets:**
  - プライベートレジストリの認証情報
  - カスタムイメージ使用時に必要

- **NodeSelector:**
  - ノード選択のラベルセレクタ
  - 特定のノードタイプへの配置

- **ServiceAccountName:**
  - 使用するServiceAccount
  - ConfigMapへのアクセス権限に必要

- **Tolerations:**
  - ノードのTaintを許容
  - 特殊なノードでの実行を可能にする

- **Volumes:**
  - 追加のVolume定義
  - カスタムデータやシークレットのマウント

**Container関連フィールド:**

- **Args / Command:**
  - コンテナの引数とコマンド
  - カスタム実行ロジックの指定

- **Env / EnvFrom:**
  - 環境変数の設定
  - メトリクスプロバイダのAPIキーなど

- **Image:**
  - コンテナイメージ
  - デフォルト: `cyberagentoss/intelligent-hpa-fittingjob:latest`

- **ImagePullPolicy:**
  - イメージプルポリシー
  - Always, IfNotPresent, Never

- **Resources:**
  - リソースリクエストとリミット
  - CPU、メモリの制限

### GenerateJobSpec メソッド

**Volume自動マウント:**
```go
volumeMounts := make([]corev1.VolumeMount, 0, len(jps.Volumes))
for _, volume := range jps.Volumes {
    volumeMount := corev1.VolumeMount{
        Name:      volume.Name,
        MountPath: "/" + volume.Name,
    }
    volumeMounts = append(volumeMounts, volumeMount)
}
```

**特徴:**
- Volumeの名前をマウントパスとして使用
- 例: Volume名 `my-data` → マウントパス `/my-data`
- 簡潔な設定が可能

**検証対象要件:** 要件 7.4, 8.2（imagePullSecretsとカスタムイメージ）

## エラーハンドリング

### リソース取得エラー

**FittingJobリソースが見つからない場合:**
```go
if err := r.Get(ctx, req.NamespacedName, &fj); err != nil {
    log.V(ResourceMessageLogLevel).Info("failed to fetch FittingJob", "error_message", err)
    return ctrl.Result{}, client.IgnoreNotFound(err)
}
```

**処理:**
- エラーをログに記録
- NotFoundエラーは無視して正常終了
- OwnerReferenceによって子リソースは自動削除される

### リソース生成エラー

**ジェネレーター初期化エラー:**
```go
g, err := NewFittingJobGenerator(&fj)
if err != nil {
    return ctrl.Result{}, fmt.Errorf("failed to create fittingjob resource generator: %w", err)
}
```

**現在の実装:**
- 常に成功（エラーを返さない）
- 将来の拡張のためのエラーハンドリング

**ConfigMap生成エラー:**
```go
cmResource, err := g.ConfigMapResource()
if err != nil {
    return ctrl.Result{}, fmt.Errorf("failed to generate configmap resource: %w", err)
}
```

**発生原因:**
- JSON marshaling エラー
- 設定データの不正

**CronJob生成エラー:**
```go
cjResource, err := g.CronJobResource()
if err != nil {
    return ctrl.Result{}, fmt.Errorf("failed to generate cronjob resource: %w", err)
}
```

**発生原因:**
- JobSpec生成エラー
- Volume設定の不正

### リソース作成/更新エラー

**ConfigMap作成/更新エラー:**
```go
if err := r.Create(ctx, cm); err != nil {
    return ctrl.Result{}, fmt.Errorf("failed to create configmap: %w", err)
}
```

**発生原因:**
- Kubernetes APIエラー
- 権限不足
- ConfigMapサイズ制限超過

**CronJob作成/更新エラー:**
```go
if err := r.Create(ctx, cj); err != nil {
    return ctrl.Result{}, fmt.Errorf("failed to create cronjob: %w", err)
}
```

**発生原因:**
- Kubernetes APIエラー
- 権限不足
- CronJobスケジュール形式エラー

**処理:**
- すべてのエラーでReconcileループが再試行
- エクスポネンシャルバックオフが適用される
- 一時的なエラーから自動復旧

## ログ出力

### ログレベル

**定数定義:**
```go
const (
    ResourceMessageLogLevel = 1
    LogicMessageLogLevel    = 1
)
```

### ログメッセージ

**リソース取得エラーログ:**
```go
log.V(ResourceMessageLogLevel).Info("failed to fetch FittingJob", "error_message", err)
```

**ConfigMap操作ログ:**
```go
log.V(ResourceMessageLogLevel).Info("initialize configmap", "name", cmResource.GetName())
log.V(ResourceMessageLogLevel).Info("successed to create/update configmap", "kind", cm.GetObjectKind().GroupVersionKind(), "name", cm.GetName())
```

**CronJob操作ログ:**
```go
log.V(ResourceMessageLogLevel).Info("initialize cronjob", "name", cjResource.GetName())
log.V(ResourceMessageLogLevel).Info("successed to create/update cronjob", "kind", cj.GetObjectKind().GroupVersionKind(), "name", cj.GetName())
```

**ログの特徴:**
- 構造化ログ（key-value形式）
- リソース名、種別を含む
- 操作の成功/失敗を明確に記録

## OwnerReference管理

### OwnerReferenceの設定

**ConfigMapへの設定:**
```go
addOwnerReference(&(g.fj.TypeMeta), &(g.fj.ObjectMeta), &configMap)
```

**CronJobへの設定:**
```go
addOwnerReference(&(g.fj.TypeMeta), &(g.fj.ObjectMeta), &cj)
```

**効果:**
- FittingJobが削除されると、ConfigMapとCronJobが自動的に削除される
- Kubernetes Garbage Collectorによって管理される
- 手動での削除処理が不要

**適用対象リソース:**
- ConfigMap（設定用）
- CronJob

## 命名規則

### ConfigMap名

**生成ロジック:**
```go
func (g *fittingJobGeneratorImpl) configMapName() string {
    return g.fj.GetName() + "-config"
}
```

**例:**
- FittingJob名: `ihpa-myapp-kubernetes-cpu-usage-total`
- ConfigMap名: `ihpa-myapp-kubernetes-cpu-usage-total-config`

**目的:**
- FittingJobとConfigMapの関連を明確にする
- 名前の衝突を避ける

### CronJob名

**生成ロジック:**
```go
ObjectMeta: metav1.ObjectMeta{
    Name:      g.fj.GetName(),
    Namespace: g.fj.GetNamespace(),
}
```

**特徴:**
- FittingJobと同じ名前を使用
- 1対1の関係が明確

## SetupWithManager

### コントローラー登録

```go
func (r *FittingJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
        For(&ihpav1beta2.FittingJob{}).
        Complete(r)
}
```

**処理内容:**
- FittingJobリソースを監視対象として登録

**監視対象:**
- FittingJob リソースの作成、更新、削除イベント

**トリガー:**
- FittingJobリソースに変更があるたびにReconcileループが実行される

## RBAC権限

### コントローラーに必要な権限

```go
// +kubebuilder:rbac:groups=ihpa.ake.cyberagent.co.jp,resources=fittingjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ihpa.ake.cyberagent.co.jp,resources=fittingjobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=cronjobs/status,verbs=get
```

**権限の分類:**

1. **FittingJobリソース:**
   - 完全な管理権限（CRUD + status更新）

2. **CronJobリソース:**
   - 完全な管理権限（CRUD）
   - ステータス読み取り

3. **ConfigMapリソース:**
   - IHPA Controllerから継承された権限を使用
   - FittingJob Controllerは直接ConfigMapの権限を宣言しない

## 実装の特徴

### 宣言的リソース管理

**特徴:**
- 現在の状態と望ましい状態を比較
- 差分を検出して必要な操作を実行
- 冪等性を保証（同じ入力で同じ結果）

**実装パターン:**
```go
if err := r.Get(ctx, types.NamespacedName{...}, resource); apierrors.IsNotFound(err) {
    // リソースが存在しない → 作成
    if err := r.Create(ctx, resource); err != nil {
        return ctrl.Result{}, err
    }
} else {
    // リソースが存在する → 更新
    resource.Data = newData
    if err := r.Update(ctx, resource); err != nil {
        return ctrl.Result{}, err
    }
}
```

### スケジューリングのランダム化

**目的:**
- 複数のFittingJobが同時実行されるのを避ける
- メトリクスプロバイダAPIの負荷分散
- Kubernetes APIサーバーの負荷分散

**実装:**
```go
func randomMinuteCronFormat(hour int) string {
    rand.Seed(time.Now().UnixNano())
    return fmt.Sprintf("%d %d * * *", rand.Intn(60), hour)
}
```

**効果:**
- 同じ時刻（ExecuteOn）を指定しても、実際の実行時刻は分単位でずれる
- 例: 10個のFittingJobがExecuteOn=4を指定
  - 実行時刻: 4:03, 4:17, 4:28, 4:41, 4:52, ...（ランダム）

### 設定の分離

**ConfigMapによる設定管理:**
- コンテナイメージと設定を分離
- 設定変更時にイメージの再ビルドが不要
- 同じイメージで異なる設定を使用可能

**利点:**
- デプロイの柔軟性
- 設定の可視性（kubectl get configmap で確認可能）
- 設定の再利用性

### カスタマイズ性

**複数のカスタマイズポイント:**

1. **コンテナイメージ:**
   - デフォルトイメージまたはカスタムイメージ
   - 特殊な予測アルゴリズムの実装

2. **Seasonality:**
   - daily, weekly, yearly, auto
   - ワークロードパターンに応じた選択

3. **ChangePointDetectionConfig:**
   - 7つのパラメータで細かく調整
   - データ品質の最適化

4. **CustomConfig:**
   - 任意の文字列設定
   - 将来の拡張に対応

5. **JobPatchSpec:**
   - リソース制限、アフィニティ、Tolerationなど
   - Kubernetes標準の設定をすべてサポート

**検証対象要件:** 要件 8.1, 8.2, 8.3, 8.4（設定とカスタマイズ）

## 設計上の考慮事項

### スケーラビリティ

**FittingJob数の増加:**
- 各FittingJobが独立して処理される
- コントローラーの並列実行が可能
- O(1)の複雑度（各FittingJobに対して）

**CronJobの実行:**
- Kubernetesのスケジューラーが管理
- 複数のFittingJobが並列実行可能
- ランダム化によって負荷分散

### 信頼性

**エラーリカバリ:**
- すべてのエラーでReconcileループが再試行
- エクスポネンシャルバックオフで負荷を軽減
- 一時的なエラーから自動復旧

**Job失敗時の再試行:**
- RestartPolicy: OnFailure
- BackoffLimitで再試行回数を制御
- 一時的な障害からの復旧

**リソースクリーンアップ:**
- OwnerReferenceによる自動削除
- 孤立リソースの防止

### 保守性

**コードの分離:**
- Reconcilerはオーケストレーションのみ
- Generatorがリソース生成ロジックを担当
- 関心の分離が明確

**テスタビリティ:**
- Generatorインターフェースでモック可能
- 各生成メソッドが独立してテスト可能
- ユニットテストが容易

**設定の可視性:**
- ConfigMapで設定を管理
- kubectl コマンドで設定を確認可能
- デバッグが容易

## 手動実行のサポート

### kubectl create job コマンド

**実行方法:**
```bash
kubectl create job --from=cronjob/<fittingjob-name> <job-name>
```

**例:**
```bash
kubectl create job --from=cronjob/ihpa-myapp-kubernetes-cpu-usage-total manual-run-1
```

**効果:**
- CronJobから即座にJobを作成
- スケジュールを待たずに実行
- テストやデバッグに有用

**検証対象要件:** 要件 6.3（手動実行サポート）

## データフロー

### 全体的なデータフロー

```
1. FittingJob Controller
   ↓ ConfigMap作成（設定）
   
2. CronJob
   ↓ スケジュール実行
   
3. FittingJob Container
   ↓ ConfigMap読み込み
   ↓ メトリクスプロバイダからデータ取得
   ↓ Prophet モデル訓練
   ↓ 変化点検知（オプション）
   ↓ 予測実行
   ↓ ConfigMap書き込み（予測データ）
   
4. Estimator Controller
   ↓ ConfigMap監視
   ↓ 予測データ読み込み
   ↓ メトリクスプロバイダに送信
   
5. HPA
   ↓ 予測メトリクス取得
   ↓ スケーリング判断
```

### ConfigMapの役割

**設定用ConfigMap（<fittingjob-name>-config）:**
- FittingJob Controllerが作成
- FittingJob Containerが読み込み
- 内容: FittingJobConfig（JSON形式）

**データ用ConfigMap（<ihpa-name>-<metric-name>）:**
- IHPA Controllerが作成（空）
- FittingJob Containerが書き込み
- Estimator Controllerが読み込み
- 内容: 予測データ（CSV形式）

**検証対象要件:** 要件 10.1, 10.2（データ永続化）

## 制限事項と今後の課題

### 現在の制限事項

**サポートされているスケジュール:**
- 日次実行のみ（毎日指定時刻に実行）
- 週次、月次などのスケジュールは未サポート

**CronJobのバージョン:**
- batch/v1beta1 を使用
- batch/v1 への移行が推奨される（Kubernetes 1.21+）

**設定の検証:**
- ConfigMapの内容検証は実装されていない
- 不正な設定でもConfigMapは作成される
- FittingJob Container実行時にエラーが発生

### 今後の改善案

**スケジュール機能の拡張:**
1. **週次実行** - 特定の曜日に実行
2. **月次実行** - 月初や月末に実行
3. **カスタムCron式** - ユーザー定義のCron式をサポート

**設定検証:**
1. **Webhook検証** - FittingJob作成時に設定を検証
2. **デフォルト値の適用** - 必須フィールドの自動設定
3. **設定テンプレート** - 一般的な設定のプリセット

**パフォーマンス最適化:**
1. **ConfigMapキャッシング** - 頻繁な読み書きを最適化
2. **バッチ更新** - 複数の設定変更を一度に適用
3. **並列実行制御** - 同時実行数の制限

**監視とデバッグ:**
1. **ステータスレポート** - FittingJobステータスにJob実行状態を反映
2. **イベント発行** - 重要な操作でKubernetesイベントを発行
3. **メトリクス露出** - Prometheusメトリクスでコントローラーの動作を監視

## まとめ

FittingJob Controllerは、機械学習モデルの訓練と予測を実行するために必要なKubernetesリソースを生成・管理します。

**主要な責務:**
1. ConfigMap生成（FittingJob設定）
2. CronJob生成（定期実行）
3. リソースライフサイクル管理

**設計の特徴:**
- 宣言的リソース管理
- スケジューリングのランダム化
- 設定の分離
- 高いカスタマイズ性
- 自動クリーンアップ

**検証対象要件:**
- 要件 6.1: FittingJobリソース作成とCronJob生成
- 要件 6.2: スケジュール実行（ExecuteOnとランダム化）
- 要件 6.3: 手動実行サポート
- 要件 8.1, 8.2, 8.3: 設定とカスタマイズ
- 要件 3.1, 3.2, 3.3, 3.4: 変化点検知設定
- 要件 7.4: imagePullSecretsサポート
- 要件 10.1, 10.2: データ永続化（ConfigMap）

**FittingJob Containerとの連携:**
- ConfigMapを通じて設定を渡す
- FittingJob Containerが予測データをConfigMapに書き込む
- Estimator Controllerが予測データを読み込む
- エンドツーエンドのデータフローを実現
