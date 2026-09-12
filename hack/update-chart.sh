#!/usr/bin/env bash
# SPDX-License-Identifier: MIT

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CHART_DIR="${REPO_ROOT}/charts/garm-operator"
CHART_CRD_DIR="${CHART_DIR}/templates/crd"
CHART_RBAC_DIR="${CHART_DIR}/templates/rbac"
CHART_WEBHOOK_DIR="${CHART_DIR}/templates/webhook"
CHART_MONITORING_DIR="${CHART_DIR}/templates/monitoring"
TMP_DIR="${REPO_ROOT}/tmp/chart-sync"
SLICED_DIR="${TMP_DIR}/sliced"
LOCALBIN="${REPO_ROOT}/bin"
KUSTOMIZE="${LOCALBIN}/kustomize"
SLICE="${LOCALBIN}/kubectl-slice"

if [[ ! -x "${KUSTOMIZE}" ]]; then
  echo "kustomize not found at ${KUSTOMIZE}. Please run 'make kustomize'." >&2
  exit 1
fi

if [[ ! -x "${SLICE}" ]]; then
  echo "kubectl-slice not found at ${SLICE}. Please run 'make slice'." >&2
  exit 1
fi

mkdir -p "${SLICED_DIR}" "${CHART_CRD_DIR}" "${CHART_RBAC_DIR}" "${CHART_WEBHOOK_DIR}" "${CHART_MONITORING_DIR}"
trap 'rm -rf "${TMP_DIR}"' EXIT

echo "1. Synchronizing Custom Resource Definitions (CRDs)..."
"${KUSTOMIZE}" build "${REPO_ROOT}/config/default" > "${TMP_DIR}/all.yaml"

"${SLICE}" -f "${TMP_DIR}/all.yaml" \
  --include-kind CustomResourceDefinition \
  --template "{{ .metadata.name }}.yaml" \
  -o "${SLICED_DIR}/"

for crd in "${SLICED_DIR}"/*.yaml; do
  name="$(basename "${crd}")"

  python3 - "${crd}" "${CHART_CRD_DIR}/${name}" << 'EOF'
import sys
import re

src_file = sys.argv[1]
dst_file = sys.argv[2]

with open(src_file, 'r') as f:
    content = f.read()

# Replace hardcoded namespace with Helm template
content = re.sub(
    r'cert-manager\.io/inject-ca-from:\s*[a-zA-Z0-9_-]+/garm-operator-serving-cert',
    'cert-manager.io/inject-ca-from: {{ .Release.Namespace }}/garm-operator-serving-cert',
    content
)
content = re.sub(
    r'service:\n(\s+)name:\s*garm-operator-webhook-service\n\1namespace:\s*[a-zA-Z0-9_-]+',
    r'service:\n\1name: garm-operator-webhook-service\n\1namespace: {{ .Release.Namespace }}',
    content
)

# Insert crd.keep annotation under metadata.annotations if annotations exists
if 'annotations:' in content:
    content = content.replace(
        '  annotations:\n',
        '  annotations:\n{{- if .Values.crd.keep }}\n    helm.sh/resource-policy: keep\n{{- end }}\n'
    )
else:
    content = content.replace(
        'metadata:\n',
        'metadata:\n{{- if .Values.crd.keep }}\n  annotations:\n    helm.sh/resource-policy: keep\n{{- end }}\n'
    )

output = f"{{{{- if .Values.crd.enable }}}}\n{content.strip()}\n{{{{- end }}}}\n"

with open(dst_file, 'w') as f:
    f.write(output)
EOF
done

echo "2. Synchronizing RBAC rules..."
python3 - "${REPO_ROOT}/config/rbac/role.yaml" "${CHART_RBAC_DIR}/manager-role.yaml" << 'EOF'
import sys

src_file = sys.argv[1]
dst_file = sys.argv[2]

with open(src_file, 'r') as f:
    content = f.read()

docs = content.split('---')
cluster_rules = []
role_rules = []

for doc in docs:
    lines = doc.strip().splitlines()
    if not lines:
        continue
    is_cluster = any('kind: ClusterRole' in l for l in lines)
    is_role = any('kind: Role' in l for l in lines)
    
    rules_lines = []
    recording = False
    for line in lines:
        if line.strip().startswith('rules:'):
            recording = True
            continue
        if recording:
            rules_lines.append(line)
            
    if is_cluster:
        cluster_rules = rules_lines
    elif is_role:
        role_rules = rules_lines

# Ensure events.k8s.io is included for controller-runtime event broadcaster
updated_cluster_rules = []
for line in cluster_rules:
    updated_cluster_rules.append(line)
    if line.strip() == '- ""':
        updated_cluster_rules.append('  - "events.k8s.io"')
if updated_cluster_rules:
    cluster_rules = updated_cluster_rules

output = f"""{{{{- if .Values.rbac.create }}}}
{{{{- if empty .Values.operator.watchNamespace }}}}
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: {{{{ include "garm-operator.fullname" . }}}}-manager-role
  labels:
    {{{{- include "garm-operator.labels" . | nindent 4 }}}}
    app.kubernetes.io/component: rbac
rules:
""" + '\n'.join(cluster_rules + role_rules) + f"""
{{{{- else }}}}
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: {{{{ include "garm-operator.fullname" . }}}}-manager-role
  labels:
    {{{{- include "garm-operator.labels" . | nindent 4 }}}}
    app.kubernetes.io/component: rbac
rules:
""" + '\n'.join(cluster_rules) + f"""
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: {{{{ include "garm-operator.fullname" . }}}}-manager-role
  namespace: {{{{ .Values.operator.watchNamespace | default .Release.Namespace }}}}
  labels:
    {{{{- include "garm-operator.labels" . | nindent 4 }}}}
    app.kubernetes.io/component: rbac
rules:
""" + '\n'.join(role_rules) + f"""
{{{{- end }}}}
{{{{- end }}}}
"""

with open(dst_file, 'w') as f:
    f.write(output)
EOF

echo "3. Synchronizing Webhook configurations..."
python3 - "${REPO_ROOT}/config/webhook/manifests.yaml" "${CHART_WEBHOOK_DIR}/validating-webhook-configuration.yaml" << 'EOF'
import sys
import re

src_file = sys.argv[1]
dst_file = sys.argv[2]

with open(src_file, 'r') as f:
    content = f.read()

# Extract webhooks: section
webhooks_idx = content.find('webhooks:')
if webhooks_idx != -1:
    webhooks_content = content[webhooks_idx:]
else:
    webhooks_content = "webhooks: []"

webhooks_content = re.sub(
    r'name:\s*webhook-service',
    r'name: {{ include "garm-operator.webhookServiceName" . }}',
    webhooks_content
)
webhooks_content = re.sub(
    r'namespace:\s*system',
    r'namespace: {{ .Release.Namespace }}',
    webhooks_content
)

output = f"""{{{{- if .Values.webhook.enable }}}}
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  name: {{{{ include "garm-operator.fullname" . }}}}-validating-webhook-configuration
  labels:
    {{{{- include "garm-operator.labels" . | nindent 4 }}}}
    app.kubernetes.io/component: webhook
  annotations:
    {{{{- if .Values.certManager.enable }}}}
    cert-manager.io/inject-ca-from: {{{{ .Release.Namespace }}}}/{{{{ include "garm-operator.fullname" . }}}}-serving-cert
    {{{{- end }}}}
{webhooks_content.strip()}
{{{{- end }}}}
"""

with open(dst_file, 'w') as f:
    f.write(output)
EOF

echo "4. Synchronizing kube-state-metrics configuration..."
python3 - "${REPO_ROOT}/config/kube-state-metrics/configmap.yaml" "${CHART_MONITORING_DIR}/kube-state-metrics-configmap.yaml" << 'EOF'
import sys

src_file = sys.argv[1]
dst_file = sys.argv[2]

with open(src_file, 'r') as f:
    content = f.read()

lines = content.splitlines()
data_lines = []
capturing_data = False

for line in lines:
    if line.startswith('data:'):
        capturing_data = True
    if capturing_data:
        data_lines.append(line)

output = f"""{{{{- if .Values.kubeStateMetrics.createConfigMap }}}}
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{{{ include "garm-operator.fullname" . }}}}-kube-state-metrics-config
  namespace: {{{{ .Release.Namespace }}}}
  labels:
    {{{{- include "garm-operator.labels" . | nindent 4 }}}}
    app.kubernetes.io/component: metrics
    app.kubernetes.io/name: kube-state-metrics
""" + '\n'.join(data_lines) + f"""
{{{{- end }}}}
"""

with open(dst_file, 'w') as f:
    f.write(output)
EOF

echo "Helm chart successfully synchronized with Kubebuilder manifests."
