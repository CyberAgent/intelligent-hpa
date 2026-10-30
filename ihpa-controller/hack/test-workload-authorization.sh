#!/usr/bin/env bash
# Run against a disposable cluster after installing config/crd and config/admission.
# Usage: ./hack/test-workload-authorization.sh kind-ihpa-security
set -euo pipefail
context=${1:?Specify a disposable Kubernetes context}
namespace=ihpa-authz-regression
user=ihpa-authz-regression
root=$(cd "$(dirname "$0")/.." && pwd)
k=(kubectl --context "$context")
"${k[@]}" create namespace "$namespace"
trap '"${k[@]}" delete namespace "$namespace" --wait=false' EXIT
"${k[@]}" create role editor -n "$namespace" --verb=create,update,patch,get --resource=fittingjobs.ihpa.ake.cyberagent.co.jp,intelligenthorizontalpodautoscalers.ihpa.ake.cyberagent.co.jp
"${k[@]}" create rolebinding editor -n "$namespace" --role=editor --user="$user"
for version in v1beta1 v1beta2; do
  for resource in fittingjob ihpa_simple; do
    fixture="$root/config/samples/${resource}_${version}.yaml"
    if [[ "$resource/$version" == fittingjob/v1beta1 ]]; then fixture="$root/hack/testdata/fittingjob_v1beta1.yaml"; fi
    if output=$("${k[@]}" --as="$user" create -n "$namespace" -f "$fixture" --dry-run=server 2>&1); then
      echo "ERROR: CR-only editor was allowed: $fixture"; exit 1
    fi
    [[ "$output" == *"requires permission to create Pods"* ]] || { echo "$output"; exit 1; }
  done
done
"${k[@]}" create role workload -n "$namespace" --verb=create --resource=pods
"${k[@]}" create rolebinding workload -n "$namespace" --role=workload --user="$user"
for version in v1beta1 v1beta2; do
  for resource in fittingjob ihpa_simple; do
    fixture="$root/config/samples/${resource}_${version}.yaml"
    if [[ "$resource/$version" == fittingjob/v1beta1 ]]; then fixture="$root/hack/testdata/fittingjob_v1beta1.yaml"; fi
    "${k[@]}" --as="$user" create -n "$namespace" -f "$fixture" --dry-run=server
  done
done
# Persist one resource, revoke Pod rights, and prove edits are denied too.
"${k[@]}" --as="$user" create -n "$namespace" -f "$root/config/samples/fittingjob_v1beta2.yaml"
"${k[@]}" delete rolebinding workload -n "$namespace"
if output=$("${k[@]}" --as="$user" patch fittingjob fittingjob -n "$namespace" --type=merge -p '{"spec":{"image":"untrusted/image:latest"}}' 2>&1); then
  echo 'ERROR: edit was allowed after revoking Pod rights'; exit 1
fi
[[ "$output" == *"requires permission to create Pods"* ]] || { echo "$output"; exit 1; }
echo 'Workload authorization regression checks passed.'
