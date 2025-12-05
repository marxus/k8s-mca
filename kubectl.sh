#!/bin/sh
cd "$(dirname "$0")"
export KUBECONFIG=tmp/.kubeconfig
cat <<EOF >"$KUBECONFIG"
apiVersion: v1
kind: Config
current-context: _
preferences: {}
contexts:
  - name: _
    context:
      cluster: _
      user: _
clusters:
  - name: _
    cluster:
      certificate-authority: var/run/secrets/kubernetes.io/mca-serviceaccount/ca.crt
      server: https://127.0.0.1:6443
users:
  - name: _
    user:
      username: _
EOF
exec kubectl "$@"