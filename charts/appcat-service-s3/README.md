# appcat-service-s3

![Version: 0.1.0](https://img.shields.io/badge/Version-0.1.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square)

VSHN-opinionated S3 operator for AppCat

## Installation

```bash
helm repo add appcat-service-s3 https://vshn.github.io/appcat-service-s3
helm install appcat-service-s3 appcat-service-s3/appcat-service-s3
```

<!---
Common/Useful Link references from values.yaml
-->
[resource-units]: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#resource-units-in-kubernetes
[prometheus-operator]: https://github.com/coreos/prometheus-operator
# appcat-service-s3

![Version: 0.1.0](https://img.shields.io/badge/Version-0.1.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square)

VSHN-opinionated S3 operator for AppCat

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` |  |
| fullnameOverride | string | `""` |  |
| image.pullPolicy | string | `"IfNotPresent"` | Operator image pull policy If set to empty, then Kubernetes default behaviour applies. |
| image.registry | string | `"ghcr.io"` | Operator image registry |
| image.repository | string | `"vshn/appcat-service-s3"` | Operator image repository |
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

