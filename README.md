# appcat-service-3

[![Build](https://img.shields.io/github/workflow/status/vshn/appcat-service-s3/Test)][build]
![Go version](https://img.shields.io/github/go-mod/go-version/vshn/appcat-service-s3)
[![Version](https://img.shields.io/github/v/release/vshn/appcat-service-s3)][releases]
[![Maintainability](https://img.shields.io/codeclimate/maintainability/vshn/appcat-service-s3)][codeclimate]
[![Coverage](https://img.shields.io/codeclimate/coverage/vshn/appcat-service-s3)][codeclimate]
[![GitHub downloads](https://img.shields.io/github/downloads/vshn/appcat-service-s3/total)][releases]

[build]: https://github.com/vshn/appcat-service-s3/actions?query=workflow%3ATest
[releases]: https://github.com/vshn/appcat-service-s3/releases
[codeclimate]: https://codeclimate.com/github/vshn/appcat-service-s3

VSHN opinionated operator to deploy S3 resources on supported cloud providers.

https://vshn.github.io/appcat-service-s3/

## Local Development

### Requirements

* `docker`
* `go`
* `helm`
* `kubectl`
* `yq`
* `sed` (or `gsed` for Mac)

Some other requirements (e.g. `kind`) will be compiled on-the-fly and put in the local cache dir `.kind` as needed.

### Common make targets

* `make build` to build the binary and docker image
* `make generate` to (re)generate additional code artifacts
* `make test` run test suite
* `make local-install` to install the operator in local cluster
* `make install-samples` to run the provider in local cluster and apply a sample instance
* `make run-operator` to run the code in operator mode against your current kubecontext

See all targets with `make help`

### QuickStart Demonstration

1. Get an API token cloudscale.ch
1. `export CLOUDSCALE_API_TOKEN=<the-token>`
1. `make local-install install-samples`
