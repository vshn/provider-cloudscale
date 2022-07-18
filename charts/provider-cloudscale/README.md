# provider-cloudscale

![Version: 0.1.1](https://img.shields.io/badge/Version-0.1.1-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square)

VSHN-opinionated S3 operator for AppCat

## Installation

```bash
helm repo add provider-cloudscale https://vshn.github.io/provider-cloudscale
helm install provider-cloudscale provider-cloudscale/provider-cloudscale
```
```bash
kubectl apply -f https://github.com/vshn/provider-cloudscale/releases/download/provider-cloudscale-0.1.1/crds.yaml
```

<!---
The values below are generated with helm-docs!

Document your changes in values.yaml and let `make chart-docs` generate this section.
-->
## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` |  |
| fullnameOverride | string | `""` |  |
| image.pullPolicy | string | `"IfNotPresent"` | Operator image pull policy If set to empty, then Kubernetes default behaviour applies. |
| image.registry | string | `"ghcr.io"` | Operator image registry |
| image.repository | string | `"vshn/provider-cloudscale"` | Operator image repository |
| image.tag | string | `"latest"` | Operator image tag |
| imagePullSecrets | list | `[]` | List of image pull secrets if custom image is behind authentication. |
| nameOverride | string | `""` |  |
| nodeSelector | object | `{}` |  |
| operator.args | list | `[]` | Overrides arguments passed to the entrypoint |
| podAnnotations | object | `{}` | Annotations to add to the Pod spec. |
| podSecurityContext | object | `{}` | Security context to add to the Pod spec. |
| replicaCount | int | `1` |  |
| resources | object | `{}` |  |
| securityContext | object | `{}` | Container security context |
| service.annotations | object | `{}` | Annotations to add to the service |
| service.port | int | `80` | Service port number |
| service.type | string | `"ClusterIP"` | Service type |
| serviceAccount.annotations | object | `{}` | Annotations to add to the service account |
| serviceAccount.create | bool | `true` | Specifies whether a service account should be created |
| serviceAccount.name | string | `""` | The name of the service account to use. If not set and `.create` is `true`, a name is generated using the fullname template |
| tokens.cloudscale | string | `""` | The cloudscale.ch API token |
| tokens.externalSecretName | string | `""` | Name of the external secret if tokens are not managed by this chart.   See `templates/secret.yaml` to figure out how to setup the expected environment variables. |
| tolerations | list | `[]` |  |

<!---
Common/Useful Link references from values.yaml
-->
[resource-units]: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#resource-units-in-kubernetes
[prometheus-operator]: https://github.com/coreos/prometheus-operator
