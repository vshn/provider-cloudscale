```bash
kubectl apply -f https://github.com/vshn/provider-cloudscale/releases/download/provider-cloudscale-{{ template "chart.version" . }}/crds.yaml
```

{{ template "chart.sourcesSection" . }}

{{ template "chart.requirementsSection" . }}
<!---
The values below are generated with helm-docs!

Document your changes in values.yaml and let `make chart-docs` generate this section.
-->
{{ template "chart.valuesSection" . }}
