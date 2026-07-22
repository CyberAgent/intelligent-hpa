# IHPA Controller 実装仕様書

## 概要

IntelligentHorizontalPodAutoscalerReconciler（IHPA Controller）は、IHPAカスタムリソースを監視し、予測ベースのオートスケーリングに必要なすべてのKubernetesリソースを生成・管理するコントローラーです。

## アーキテクチャ

### コントローラーの責務

IHPA Controllerは以下のリソースのライフサイクルを管理します：

1. **HorizontalPodAutoscaler (HPA)** - 標準のHPAに予測メトリクスを追加
2. **FittingJob** - 各メトリクスに対する機械学習ジョブ
3. **Estimator** - 各メトリクスに対する予測値送信リソース
4. **RBAC Resources** - ServiceAccount、Role、RoleBinding

### コンポーネント構成

```
IntelligentHorizontalPodAutoscalerReconciler
├── Client (Kubernetes API Client)
├── Logger (構造化ログ)
├── Scheme (Kubernetes スキーマ)
└── fittingJobMap (削除時のクリーンアップ用マップ)
```

## Reconcileループの詳細フロー

### 1. リソース取得フェーズ

```go
func (r *IntelligentHorizontalPodAutoscalerReconciler) Reconcile(req ctrl.Request)
```

**処理内容:**
- IHPAリソースをNamespacedNameで取得
- リソースが見つからない場合（削除された場合）、fittingJobMapからエントリを削除
- エラーは`client.IgnoreNotFound(err)`で処理（NotFoundエラーは無視）

**検証対象要件:** 要件 1.3（削除時のクリーンアップ）

### 2. ジェネレーター初期化フェーズ

```go
g, err := NewIntelligentHorizontalPodAutoscalerGenerator(&ihpa, r, ctx)
```

**ジェネレーター初期化処理:**

1. **kube-systemのUID取得** - クラスタの一意性を保証
2. **ScaleTargetの取得** - Deployment/StatefulSet/ReplicaSetからコンテナ情報を取得
3. **リソースリクエストの集計** - 全コンテナのリソースリクエストを合計

**検証対象要件:** 要件 1.1（リソース生成の基盤）

### 3. HPA生成・更新フェーズ

```go
hpaResource, err := g.HorizontalPodAutoscalerResource()
```

**HPA生成ロジック:**

1. **標準メトリクスの抽出** - ExtendedMetricSpecから標準のMetricSpecを抽出
2. **予測メトリクスの生成** - 各メトリクスに対応する予測メトリクスを生成
3. **メトリクスの結合** - 標準メトリクス + 予測メトリクスを結合

**予測メトリクス生成の詳細:**

- **Resource メトリクス（Utilization）の処理:**
  - コンテナのリソースリクエスト合計を取得
  - Utilization（パーセンテージ）を実際の値に変換
  - メトリクスプロバイダのスケール（例: Datadogのnanocoreスケール）に調整
  - CPUの場合、ミリコア単位の調整（スケール調整 +3）

- **External メトリクスの処理:**
  - AverageValue または Value を抽出
  - 予測メトリクスは AverageValue として設定（レプリカ数で除算されるため）

- **メトリクス命名規則:**
  - プレフィックス: `ake.ihpa.forecasted_`
  - 例: `kubernetes.cpu.usage.total` → `ake.ihpa.forecasted_kubernetes_cpu_usage_total`

**リソース作成/更新:**
- 存在しない場合: `r.Create(ctx, hpa)`
- 存在する場合: Specを更新して `r.Update(ctx, hpa)`

**検証対象要件:** 要件 1.1, 1.2, 1.5（HPA生成と複数メトリクスサポート）

### 4. RBAC リソース生成フェーズ

```go
saResource, roleResource, roleBindingResource, err := g.RBACResources()
```

**RBAC生成ロジック:**

1. **ServiceAccount:**
   - 名前: `ihpa-<ihpa_name>`
   - 名前空間: IHPAと同じ名前空間

2. **Role:**
   - 名前: `ihpa-<ihpa_name>`
   - 権限: ConfigMapの `get`, `update`, `patch`
   - ResourceNames: すべてのメトリクスに対応するConfigMap名

3. **RoleBinding:**
   - 名前: `ihpa-<ihpa_name>`
   - ServiceAccountとRoleをバインド

**リソース作成:**
- 各リソースは存在しない場合のみ作成（`apierrors.IsNotFound(err)`でチェック）
- OwnerReferenceを設定してIHPAとの親子関係を確立

**検証対象要件:** 要件 7.1, 7.2, 7.5（RBAC設定と権限分離）

### 5. FittingJob リソース生成フェーズ

```go
fittingJobResources, err := g.FittingJobResources()
```

**FittingJob生成ロジック（メトリクスごと）:**

1. **メトリクス識別子の変換:**
   - Resource メトリクス: プロバイダ固有の名前に変換（例: `cpu` → `kubernetes.cpu.usage.total`）
   - External メトリクス: そのまま使用

2. **FittingJobSpec の構築:**
   - `TargetMetric`: 変換されたメトリクス識別子
   - `DataConfigMap`: 予測データ保存用ConfigMap名
   - `Provider`: メトリクスプロバイダ設定
   - `ServiceAccountName`: RBAC用ServiceAccount名
   - `Seasonality`, `ExecuteOn`, `ChangePointDetectionConfig`: ユーザー設定

3. **一意性の保証:**
   - アノテーション `ihpa.ake.cyberagent.co.jp/fittingjob-id` にMD5ハッシュを設定
   - ハッシュ計算要素: メトリクス名、名前空間、ScaleTarget種別、ScaleTarget名

**リソース作成/更新:**
- 存在しない場合: 新規作成
- 存在する場合: Specを更新

**検証対象要件:** 要件 1.1, 1.5, 6.1（FittingJob生成と複数メトリクス）

### 6. Estimator リソース生成フェーズ

```go
estimatorResources, err := g.EstimatorResources()
```

**Estimator生成ロジック（メトリクスごと）:**

1. **メトリクスタグの生成:**
   - 一意性フィルタから生成: `kube_system_uid`, `kube_namespace`, `kube_<kind>`
   - ソート済み配列として保存

2. **ベースメトリクスタグの検索:**
   - External メトリクスの場合: 元のメトリクスのセレクタから抽出
   - その他のメトリクス: 一意性フィルタから生成

3. **EstimatorSpec の構築:**
   - `DataConfigMap`: FittingJobと同じConfigMap
   - `Provider`: メトリクスプロバイダ設定
   - `MetricName`: 予測メトリクス名（`ake.ihpa.forecasted_` プレフィックス付き）
   - `MetricTags`: 予測メトリクス用タグ
   - `BaseMetricName`: 元のメトリクス名
   - `BaseMetricTags`: 元のメトリクス用タグ
   - `Mode`: 調整モード（raw/adjust）
   - `GapMinutes`: タイムスタンプ調整用の分数

**リソース作成/更新:**
- 存在しない場合: 新規作成
- 存在する場合: Specを更新

**検証対象要件:** 要件 1.1, 1.5, 4.1（Estimator生成と調整機能）

### 7. リソース追跡フェーズ

**現在のFittingJob IDの収集:**
```go
currentIds := make(map[string]struct{}, len(fittingJobResources))
for _, fj := range fittingJobResources {
    id, ok := fj.GetAnnotations()[fittingJobIDAnnotation]
    if ok {
        currentIds[id] = struct{}{}
        r.fittingJobMap[ihpaNamespacedName][id] = struct{}{}
    }
}
```

**目的:**
- 現在のメトリクス設定に対応するFittingJob IDを追跡
- 削除時のクリーンアップに使用

**検証対象要件:** 要件 1.4（リソース追跡）

### 8. 不要リソース削除フェーズ

**削除対象の特定:**
```go
if fjIdsStr, ok := annotations[fittingJobIDsAnnotation]; ok {
    previousIds := strings.Split(fjIdsStr, ",")
    deleteIds := make(map[string]struct{}, len(previousIds))
    for _, id := range previousIds {
        if _, ok := currentIds[id]; !ok {
            deleteIds[id] = struct{}{}
        }
    }
}
```

**削除処理:**

1. **FittingJobの削除:**
   - すべてのFittingJobをリスト
   - アノテーションのIDが削除対象に含まれる場合、削除

2. **Estimatorの削除:**
   - すべてのEstimatorをリスト
   - アノテーションのIDが削除対象に含まれる場合、削除

3. **fittingJobMapからの削除:**
   - 削除されたIDをfittingJobMapから除去

**削除シナリオ:**
- IHPAのメトリクス設定が変更され、一部のメトリクスが削除された場合
- 以前のアノテーションに存在するが、現在の設定に存在しないIDが削除対象

**検証対象要件:** 要件 1.2, 1.3（リソース更新と削除）

### 9. アノテーション更新フェーズ

**FittingJob IDsアノテーションの更新:**
```go
var fjIdsStr string
for k, _ := range currentIds {
    fjIdsStr += k + ","
}
fjIdsStr = strings.TrimRight(fjIdsStr, ",")
annotations[fittingJobIDsAnnotation] = fjIdsStr
ihpa.SetAnnotations(annotations)
```

**アノテーション形式:**
- キー: `ihpa.ake.cyberagent.co.jp/fittingjob-ids`
- 値: カンマ区切りのFittingJob IDリスト
- 例: `a1b2c3d4e5f6,f6e5d4c3b2a1`

**目的:**
- 次回のReconcileループで削除対象を特定するため
- IHPAリソースに現在のメトリクス設定を記録

**検証対象要件:** 要件 1.4（リソース追跡）

## リソース生成の詳細

### HPA生成の詳細

**generateForecastedMetricSpec メソッド:**

このメソッドは標準のMetricSpecから予測用のExternalMetricSpecを生成します。

**Resource メトリクス（Utilization）の変換:**

1. **リソースリクエストの取得:**
   ```go
   if q, ok := g.scaleTargetRequests[metric.Resource.Name]; ok {
       // CPU: ミリコア単位で取得
       // Memory/Storage: バイト単位で取得
   }
   ```

2. **スケール調整の計算:**
   ```go
   // CPUの場合
   qty = resource.NewQuantity(q.MilliValue(), resource.DecimalSI)
   scaleAdjust = 3  // ミリコア → ナノコアへの調整
   
   // Memory/Storageの場合
   qty = resource.NewQuantity(q.Value(), resource.DecimalSI)
   scaleAdjust = 0
   ```

3. **Utilization値の適用:**
   ```go
   percentage := float64(*(metricTarget.AverageUtilization)) / 100.0
   requestTotal := float64(qty.ScaledValue(resource.Scale(mi.GetScale() + scaleAdjust)))
   avg = int64(requestTotal * percentage)
   ```

**例: CPU Utilization 50%の変換:**
- コンテナリクエスト: 1000m (1 core)
- Utilization: 50%
- Datadogスケール: -9 (nanocore)
- 計算: 1000m * 50% = 500m = 500,000,000 nanocore
- HPA表示: 500,000,000（単位なし）

**External メトリクスの変換:**

1. **AverageValue または Value の抽出**
2. **AverageValue への統一:**
   - 予測メトリクスは単一データポイント
   - HPAはレプリカ数で除算するため、AverageValueとして設定

**メトリクス識別子の生成:**

```go
metricIdentifier.Name = correspondForecastedMetricName(metricIdentifier.Name)
metricIdentifier.Selector = &metav1.LabelSelector{MatchLabels: g.uniqueMetricFilters()}
```

**一意性フィルタ:**
```go
{
    "kube_system_uid": "<kube-system namespace UID>",
    "kube_namespace": "<IHPA namespace>",
    "kube_<kind>": "<ScaleTarget name>"
}
```

### FittingJob生成の詳細

**メトリクス識別子の変換:**

```go
func (g *ihpaGeneratorImpl) convertMetricSpecToIdentifier(metric *autoscalingv2beta2.MetricSpec)
```

**Resource メトリクスの変換:**
- メトリクスプロバイダの変換関数を使用
- 例: Datadogの場合
  - `cpu` → `kubernetes.cpu.usage.total`
  - `memory` → `kubernetes.memory.usage`

**External メトリクスの変換:**
- そのまま使用（変換不要）

**ConfigMap名の生成:**
```go
func (g *ihpaGeneratorImpl) configMapName(metric *ihpav1beta2.ExtendedMetricSpec) string {
    return g.ihpaMetricString(metric.MetricSpec())
}
```

**命名規則:**
- フォーマット: `ihpa-<ihpa_name>-<metric_name>`
- サニタイズ: `.` と `_` を `-` に変換
- 例: `ihpa-myapp-kubernetes-cpu-usage-total`

**一意性ID の生成:**

```go
func (g *ihpaGeneratorImpl) uniqueMetricID(metric *autoscalingv2beta2.MetricSpec) string {
    hash := g.uniqueMetricHash(metric)
    dst := make([]byte, hex.EncodedLen(len(hash)))
    hex.Encode(dst, hash)
    return string(dst)
}
```

**ハッシュ計算要素:**
- メトリクス名
- 予測メトリクス名
- 名前空間
- ScaleTarget種別（Deployment/StatefulSet/ReplicaSet）
- ScaleTarget名

**目的:**
- 同じメトリクス設定に対して常に同じIDを生成
- メトリクス設定の変更を検出
- 削除対象の特定に使用

### Estimator生成の詳細

**メトリクスタグの生成:**

```go
tagMap := g.uniqueMetricFilters()
tags := make([]string, 0, len(tagMap))
for k, v := range tagMap {
    tags = append(tags, k+":"+v)
}
sort.Strings(tags)
```

**タグ形式:**
- `key:value` 形式
- ソート済み（一貫性のため）
- 例: `["kube_deployment:myapp", "kube_namespace:default", "kube_system_uid:abc123"]`

**ベースメトリクスタグの検索:**

External メトリクスの場合、元のメトリクスのセレクタから抽出:
```go
if metric.Type == "External" {
    m := metric.External.Metric.Selector.MatchLabels
    baseMetricTags = make([]string, 0, len(m))
    for k, v := range m {
        baseMetricTags = append(baseMetricTags, k+":"+v)
    }
}
```

**目的:**
- 予測メトリクスと元のメトリクスを区別
- 調整モードで元のメトリクスを取得するため

## エラーハンドリング

### リソース取得エラー

**IHPAリソースが見つからない場合:**
```go
if apierrors.IsNotFound(err) {
    log.V(LogicMessageLogLevel).Info("cleanup estimator", "target", ihpaNamespacedName)
    delete(r.fittingJobMap, ihpaNamespacedName)
}
return ctrl.Result{}, client.IgnoreNotFound(err)
```

**処理:**
- fittingJobMapからエントリを削除
- エラーを無視して正常終了
- OwnerReferenceによって子リソースは自動削除される

### リソース生成エラー

**ジェネレーター初期化エラー:**
```go
g, err := NewIntelligentHorizontalPodAutoscalerGenerator(&ihpa, r, ctx)
if err != nil {
    return ctrl.Result{}, fmt.Errorf("failed to create ihpa manager: %w", err)
}
```

**発生原因:**
- kube-system名前空間が見つからない
- ScaleTargetリソース（Deployment/StatefulSet/ReplicaSet）が見つからない

**処理:**
- エラーをラップして返す
- Reconcileループが再試行

### リソース作成/更新エラー

**各リソースの作成/更新エラー:**
```go
if err := r.Create(ctx, hpa); err != nil {
    return ctrl.Result{}, fmt.Errorf("failed to create hpa: %w", err)
}
```

**発生原因:**
- Kubernetes APIエラー
- 権限不足
- リソース競合

**処理:**
- エラーをラップして返す
- Reconcileループが再試行
- エクスポネンシャルバックオフが適用される

### リソース削除エラー

**FittingJob/Estimator削除エラー:**
```go
if err := r.Delete(ctx, &fj); err != nil {
    return ctrl.Result{}, fmt.Errorf("failed to delete fittingjob: %w", err)
}
```

**処理:**
- エラーをラップして返す
- Reconcileループが再試行
- 次回のReconcileで再度削除を試行

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

**リソース操作ログ:**
```go
log.V(ResourceMessageLogLevel).Info("initialize hpa", "name", hpaResource.GetName())
log.V(ResourceMessageLogLevel).Info("successed to create/update hpa", "kind", hpa.GetObjectKind().GroupVersionKind(), "name", hpa.GetName())
```

**ロジック操作ログ:**
```go
log.V(LogicMessageLogLevel).Info("cleanup estimator", "target", ihpaNamespacedName)
```

**削除操作ログ:**
```go
log.V(ResourceMessageLogLevel).Info("successed to delete fitting job", "kind", fj.GetObjectKind(), "name", fj.GetName(), "id", id)
log.V(ResourceMessageLogLevel).Info("successed to delete estimator", "kind", est.GetObjectKind(), "name", est.GetName(), "id", id)
```

**アノテーション更新ログ:**
```go
log.V(ResourceMessageLogLevel).Info("successed to apply annotation", fittingJobIDsAnnotation, fjIdsStr)
```

## OwnerReference管理

### OwnerReferenceの設定

**addOwnerReference関数:**
```go
func addOwnerReference(ownerTypeMeta *metav1.TypeMeta, ownerObjectMeta *metav1.ObjectMeta, dependent metav1.Object)
```

**設定内容:**
- `APIVersion`: IHPAのAPIバージョン
- `Controller`: true（このリソースがコントローラー）
- `BlockOwnerDeletion`: true（所有者が削除されるまで削除をブロック）
- `Kind`: IHPAのKind
- `Name`: IHPAの名前
- `UID`: IHPAのUID

**効果:**
- IHPAが削除されると、すべての子リソースが自動的に削除される
- Kubernetes Garbage Collectorによって管理される
- 手動での削除処理が不要

**適用対象リソース:**
- HorizontalPodAutoscaler
- FittingJob
- Estimator
- ServiceAccount
- Role
- RoleBinding

## 命名規則とサニタイズ

### リソース名の生成

**基本パターン:**
```go
func (g *ihpaGeneratorImpl) ihpaString() string {
    s := fmt.Sprintf("ihpa-%s", strings.ToLower(g.ihpa.GetName()))
    return sanitizeForKubernetesResourceName(s)
}
```

**メトリクス固有のリソース名:**
```go
func (g *ihpaGeneratorImpl) ihpaMetricString(metric *autoscalingv2beta2.MetricSpec) string {
    metricName, _ := extractScopedMetricInfo(metric)
    s := fmt.Sprintf("%s-%s", g.ihpaString(), strings.ToLower(metricName))
    return sanitizeForKubernetesResourceName(s)
}
```

### サニタイズルール

**Kubernetesリソース名用:**
```go
func sanitizeForKubernetesResourceName(s string) string {
    sanitizeMap := map[string]string{".": "-", "_": "-"}
    return replaceStrings(s, sanitizeMap)
}
```

**変換例:**
- `kubernetes.cpu.usage.total` → `kubernetes-cpu-usage-total`
- `my_custom_metric` → `my-custom-metric`

**予測メトリクス名用:**
```go
func correspondForecastedMetricName(metricName string) string {
    sanitizeMap := map[string]string{".": "_", "-": "_"}
    return MetricPath + ".forecasted_" + replaceStrings(metricName, sanitizeMap)
}
```

**変換例:**
- `kubernetes.cpu.usage.total` → `ake.ihpa.forecasted_kubernetes_cpu_usage_total`
- `my-custom-metric` → `ake.ihpa.forecasted_my_custom_metric`

**理由:**
- Kubernetesリソース名: DNS-1123準拠（英数字、`-`、`.`のみ）
- メトリクス名: プロバイダの命名規則に準拠（Datadogは`.`と`_`を使用）

## SetupWithManager

### コントローラー登録

```go
func (r *IntelligentHorizontalPodAutoscalerReconciler) SetupWithManager(mgr ctrl.Manager) error {
    r.fittingJobMap = make(map[string]map[string]struct{})

    return ctrl.NewControllerManagedBy(mgr).
        For(&ihpav1beta2.IntelligentHorizontalPodAutoscaler{}).
        Complete(r)
}
```

**処理内容:**
1. **fittingJobMapの初期化** - 削除時のクリーンアップ用マップ
2. **コントローラーの登録** - IHPAリソースを監視

**監視対象:**
- IntelligentHorizontalPodAutoscaler リソースの作成、更新、削除イベント

**トリガー:**
- IHPAリソースに変更があるたびにReconcileループが実行される

## RBAC権限

### コントローラーに必要な権限

```go
// +kubebuilder:rbac:groups=ihpa.ake.cyberagent.co.jp,resources=intelligenthorizontalpodautoscalers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ihpa.ake.cyberagent.co.jp,resources=intelligenthorizontalpodautoscalers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers/status,verbs=get
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list
// +kubebuilder:rbac:groups=core,resources=secrets/status,verbs=get
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps/status,verbs=get
// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
```

**権限の分類:**

1. **IHPAリソース:**
   - 完全な管理権限（CRUD + status更新）

2. **HPAリソース:**
   - 完全な管理権限（CRUD）
   - ステータス読み取り

3. **ConfigMap:**
   - 完全な管理権限（CRUD + watch）
   - FittingJobとEstimatorのデータ保存用

4. **RBAC リソース:**
   - ServiceAccount、Role、RoleBindingの完全な管理権限

5. **Secrets:**
   - 読み取り専用（メトリクスプロバイダの認証情報用）

6. **Namespaces:**
   - 読み取り専用（kube-system UIDの取得用）

7. **Deployments:**
   - 読み取り専用（ScaleTargetのリソースリクエスト取得用）

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
    resource.Spec = newSpec
    if err := r.Update(ctx, resource); err != nil {
        return ctrl.Result{}, err
    }
}
```

### メトリクスごとの独立性

**設計原則:**
- 各メトリクスに対して独立したFittingJobとEstimatorを生成
- メトリクスの追加・削除が他のメトリクスに影響しない
- 並列実行が可能

**実装:**
```go
for i := range metrics {
    fjs[i], err = g.fittingJobResource(&metrics[i])
    ests[i], err = g.estimatorResource(&metrics[i])
}
```

### 一意性の保証

**クラスタレベルの一意性:**
- kube-system名前空間のUIDを使用
- 複数クラスタで同じメトリクス名が衝突しない

**名前空間レベルの一意性:**
- 名前空間名を含める
- 同じクラスタ内の異なる名前空間で衝突しない

**リソースレベルの一意性:**
- ScaleTarget種別と名前を含める
- 同じ名前空間内の異なるリソースで衝突しない

**メトリクスレベルの一意性:**
- メトリクス名を含める
- 同じリソースの異なるメトリクスで衝突しない

**実装:**
```go
func generateMetricUniqueFilter(kubeSystemUID, namespace, kind, name string) map[string]string {
    return map[string]string{
        "kube_system_uid":               kubeSystemUID,
        "kube_namespace":                namespace,
        "kube_" + strings.ToLower(kind): name,
    }
}
```

### リソースリクエストの集計

**目的:**
- Resource メトリクスのUtilizationを実際の値に変換
- 全コンテナのリソースリクエストを合計

**実装:**
```go
func totalResourceList(containers []corev1.Container) *corev1.ResourceList {
    resourceLists := make([]corev1.ResourceList, 0, len(containers))
    for _, c := range containers {
        if rl := c.Resources.Requests; rl != nil {
            resourceLists = append(resourceLists, rl)
        }
    }
    return sumUpResourceLists(resourceLists)
}
```

**集計ロジック:**
```go
func sumUpResourceLists(resourceLists []corev1.ResourceList) *corev1.ResourceList {
    baserl := corev1.ResourceList{}
    for _, rl := range resourceLists {
        for k, v := range rl {
            if baseValue, ok := baserl[k]; ok {
                baseValue.Add(v)
                baserl[k] = baseValue
            } else {
                baserl[k] = v
            }
        }
    }
    return &baserl
}
```

**例:**
- Container 1: CPU 500m, Memory 512Mi
- Container 2: CPU 300m, Memory 256Mi
- 合計: CPU 800m, Memory 768Mi

## 設計上の考慮事項

### スケーラビリティ

**メトリクス数の増加:**
- 各メトリクスが独立したリソースを生成
- O(n)の複雑度（nはメトリクス数）
- 大量のメトリクスでもパフォーマンスが線形

**IHPA数の増加:**
- 各IHPAが独立して処理される
- コントローラーの並列実行が可能
- Kubernetes APIの負荷が主なボトルネック

### 信頼性

**エラーリカバリ:**
- すべてのエラーでReconcileループが再試行
- エクスポネンシャルバックオフで負荷を軽減
- 一時的なエラーから自動復旧

**リソースクリーンアップ:**
- OwnerReferenceによる自動削除
- fittingJobMapによる追跡
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

## 制限事項と今後の課題

### 現在の制限事項

**TODOコメント:**
```go
// TODO: (low) fetch datadog key from env
// TODO: (low) determine sum/min/max/count/avg from IHPA property
// TODO: (low) consider selector of hpa object type
// TODO: (low) write any status to configmap
```

**サポートされていない機能:**
1. **環境変数からのDatadogキー取得** - 現在はSecretから取得のみ
2. **集約方法の動的決定** - 現在は`sum`固定
3. **HPAオブジェクトタイプのセレクタ** - 現在は全HPAを対象
4. **ConfigMapへのステータス書き込み** - 現在は実装なし

**サポートされているScaleTarget:**
- Deployment
- StatefulSet
- ReplicaSet

**サポートされていないScaleTarget:**
- DaemonSet
- Job
- CronJob
- カスタムリソース

### 今後の改善案

**パフォーマンス最適化:**
1. **バッチ処理** - 複数のリソース作成を一度に実行
2. **キャッシング** - ScaleTargetのリソースリクエストをキャッシュ
3. **並列処理** - メトリクスごとの処理を並列化

**機能拡張:**
1. **ステータスレポート** - IHPAステータスに子リソースの状態を反映
2. **イベント発行** - 重要な操作でKubernetesイベントを発行
3. **メトリクス露出** - Prometheusメトリクスでコントローラーの動作を監視

**エラーハンドリング改善:**
1. **リトライ戦略** - エラーの種類に応じたリトライ間隔
2. **エラー分類** - 一時的エラーと永続的エラーの区別
3. **アラート** - 重大なエラーでアラートを発行

## まとめ

IHPA Controllerは、Kubernetesの宣言的リソース管理パターンに従い、IHPAリソースから必要なすべての子リソースを生成・管理します。

**主要な責務:**
1. HPA生成（標準メトリクス + 予測メトリクス）
2. FittingJob生成（メトリクスごと）
3. Estimator生成（メトリクスごと）
4. RBAC リソース生成
5. リソースライフサイクル管理
6. 不要リソースのクリーンアップ

**設計の特徴:**
- 宣言的リソース管理
- メトリクスごとの独立性
- 一意性の保証
- 自動クリーンアップ
- エラーリカバリ

**検証対象要件:**
- 要件 1.1: IHPAリソース作成時のリソース生成
- 要件 1.2: IHPAリソース更新時のリソース更新
- 要件 1.3: IHPAリソース削除時のクリーンアップ
- 要件 1.4: リソース追跡とアノテーション管理
- 要件 1.5: 複数メトリクスのサポート
- 要件 7.1, 7.2, 7.5: RBAC設定と権限分離
