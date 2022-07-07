{{ template "chart.header" . }}
{{ template "chart.deprecationWarning" . }}

{{ template "chart.badgesSection" . }}

{{ template "chart.description" . }}

{{ template "chart.homepageLine" . }}

## Installation

```bash
helm repo add appcat-service-s3 https://vshn.github.io/appcat-service-s3
helm install {{ template "chart.name" . }} appcat-service-s3/{{ template "chart.name" . }}
```
