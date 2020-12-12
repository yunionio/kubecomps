package service

import (
	"bytes"
	"text/template"

	"yunion.io/x/pkg/errors"
)

const (
	tmpl = `
{{ $var := .ExternalURL}}{{ range $k,$v:=.Alerts }}
{{if eq $v.Status "resolved"}}
## [Prometheus恢复信息]({{$v.GeneratorURL}})
#### [{{$v.labels.alertname}}]({{$var}})
###### 告警级别：{{$v.labels.level}}
###### 开始时间：{{$v.startsAt}}
###### 结束时间：{{$v.endsAt}}
###### 故障主机IP：{{$v.labels.instance}}
##### {{$v.Annotations.description}}
![Prometheus](https://raw.githubusercontent.com/feiyu563/PrometheusAlert/master/doc/alert-center.png)
{{else}}
# [Prometheus告警信息]({{$v.GeneratorURL}})
## [{{$v.Labels.alertname}}]({{$var}})
### 告警级别：{{$v.Labels.severity}}
### 开始时间：{{$v.StartsAt}}
### pod：{{$v.Labels.namespace}}/{{$v.Labels.pod}}
### node：{{$v.Labels.hostname}}
### 描述: {{$v.Annotations.description}}
![Prometheus](https://raw.githubusercontent.com/feiyu563/PrometheusAlert/master/doc/alert-center.png)
{{end}}
---
{{ end }}
	`
)

func genMarkdownContent(input *PrometheusRequest) (string, error) {
	tmp, err := template.New("").Parse(tmpl)
	if err != nil {
		return "", errors.Wrap(err, "parse template")
	}

	var out bytes.Buffer
	if err := tmp.Execute(&out, input); err != nil {
		return "", errors.Wrap(err, "execute template")
	}

	return out.String(), nil
}
