#!/bin/bash

set -eo pipefail
#set -x

endpoint=${1}
bucket_name=${2}
file_path=${3}
secret_name=${4}
namespace=${5:-default}

export PATH=$PATH:$(go env GOPATH)/bin
access_key=$(kubectl -n ${namespace} get secret ${secret_name} -o jsonpath='{.data.AWS_ACCESS_KEY_ID}' | base64 -d)
secret_key=$(kubectl -n ${namespace} get secret ${secret_name} -o jsonpath='{.data.AWS_SECRET_ACCESS_KEY}' | base64 -d)
export MC_HOST_cloudscale=https://${access_key}:${secret_key}@${endpoint}

mc cp --quiet "${file_path}" "cloudscale/${bucket_name}"
