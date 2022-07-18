{{ template "chart.header" . }}
{{ template "chart.deprecationWarning" . }}

{{ template "chart.badgesSection" . }}

{{ template "chart.description" . }}

{{ template "chart.homepageLine" . }}

## Installation

```bash
helm repo add provider-cloudscale https://vshn.github.io/provider-cloudscale
helm install {{ template "chart.name" . }} provider-cloudscale/{{ template "chart.name" . }}
```
