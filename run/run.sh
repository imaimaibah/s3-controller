#!/bin/bash

if [ ! -d $(pwd)/project ];then
  mkdir -p $(pwd)/project
fi
CERT_AUTH_DATA=$(kubectl config view --raw -o json | jq -r '.clusters[] | select(.name=="kind-kind").cluster."certificate-authority-data"')
USER_CERT_DATA=$(kubectl config view --raw -o json | jq -r '.users[] | select(.name=="kind-kind").user."client-certificate-data"')
USER_KEY_DATA=$(kubectl config view --raw -o json | jq -r '.users[] | select(.name=="kind-kind").user."client-key-data"')
cat <<__EOF__ > .kubeconfig
apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: ${CERT_AUTH_DATA}
    server: https://kind-control-plane:6443
  name: kind-kind
contexts:
- context:
    cluster: kind-kind
    user: kind-kind
  name: kind-kind
current-context: kind-kind
kind: Config
preferences: {}
users:
- name: kind-kind
  user:
    client-certificate-data: ${USER_CERT_DATA}
    client-key-data: ${USER_KEY_DATA}
__EOF__

docker run --rm -it \
  --network kind \
  -e AWS_ROLE_ARN=arn:aws:iam::158484697723:role/connect-dev-external-dns,AWS_WEB_IDENTITY_TOKEN_FILE=token,AWS_REGION=eu-west-2 \
  --workdir /go/src/github.com/imaimaibah/s3-controller \
  -v $(pwd)/.kubeconfig:/root/.kube/config \
  -v $(pwd)/project:/go/src/github.com/imaimaibah/s3-controller \
  kubebuilder:s3-controller

