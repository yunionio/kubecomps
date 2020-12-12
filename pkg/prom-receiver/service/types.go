package service

type PrometheusRequest struct {
	Status      string            `json:"status"`
	ExternalURL string            `json:"externalURL"`
	Alerts      []PrometheusAlert `json:"alert"`
}

type PrometheusAlert struct {
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     string            `json:"startsAt"`
	EndsAt       string            `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
}
