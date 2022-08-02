# provider-cloudscale

[![Build](https://img.shields.io/github/workflow/status/vshn/provider-cloudscale/Test)][build]
![Go version](https://img.shields.io/github/go-mod/go-version/vshn/provider-cloudscale)
[![Version](https://img.shields.io/github/v/release/vshn/provider-cloudscale)][releases]
[![Maintainability](https://img.shields.io/codeclimate/maintainability/vshn/provider-cloudscale)][codeclimate]
[![Coverage](https://img.shields.io/codeclimate/coverage/vshn/provider-cloudscale)][codeclimate]
[![GitHub downloads](https://img.shields.io/github/downloads/vshn/provider-cloudscale/total)][releases]

[build]: https://github.com/vshn/provider-cloudscale/actions?query=workflow%3ATest
[releases]: https://github.com/vshn/provider-cloudscale/releases
[codeclimate]: https://codeclimate.com/github/vshn/provider-cloudscale

Crossplane provider for managing resources on cloudscale.ch.

Documentation: https://vshn.github.io/provider-cloudscale/

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
* `make install-samples` to run the provider in local cluster and apply sample manifests
* `make run-operator` to run the code in operator mode against your current kubecontext

See all targets with `make help`

### QuickStart Demonstration

1. Get an API token cloudscale.ch
1. `export CLOUDSCALE_API_TOKEN=<the-token>`
1. `make local-install install-samples`

### Kubernetes Webhook Troubleshooting

The provider comes with mutating and validation admission webhook server.

To test and troubleshoot the webhooks on the cluster, simply apply your changes with `kubectl`.

1.  To debug the webhook in an IDE, we need to generate certificates:
    ```bash
    make webhook-cert
    ```
2.  Start the operator in your IDE with `WEBHOOK_TLS_CERT_DIR` environment set to `.kind`.

3.  Send an admission request sample of the spec:
    ```bash
    # send an admission request
    curl -k -v -H "Content-Type: application/json" --data @samples/admission.k8s.io_admissionreview.json https://localhost:9443/validate-cloudscale-crossplane-io-v1-bucket
    ```
