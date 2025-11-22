#!/bin/sh
cd "$(dirname "$0")"
exec docker run --rm -it \
  -v "$PWD/tmp/var/run/secrets/kubernetes.io/mca-serviceaccount:/var/run/secrets/kubernetes.io/serviceaccount" \
  -e KUBERNETES_SERVICE_HOST=192.168.5.2 \
  -e KUBERNETES_SERVICE_PORT=6443 \
  alpine/kubectl:1.34.2 "$@"