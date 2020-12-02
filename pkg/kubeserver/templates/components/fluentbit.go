package components

type FluentBitImage struct {
	FluentBit Image `json:"fluent_bit"`
}

type FluentBitBackendTLS struct {
	TLS       string `json:"tls"`
	TLSVerify string `json:"tls_verify"`
	TLSCA     string `json:"tls_ca"`
}

type FluentBitBackendCommon struct {
	Enabled bool `json:"enabled"`
}

type FluentBitBackendES struct {
	FluentBitBackendCommon
	Host string `json:"host"`
	Port int    `json:"port"`
	// Elastic index name, default: fluentbit
	Index string `json:"index"`
	// Type name, default: flb_type
	Type           string `json:"type"`
	LogstashPrefix string `json:"logstash_prefix"`
	ReplaceDots    string `json:"replace_dots"`
	LogstashFormat string `json:"logstash_format"`
	// Optional username credential for Elastic X-Pack access
	HTTPUser string `json:"httpUser"`
	// Password for user defined in HTTPUser
	HTTPPassword string `json:"httpPassword"`
	FluentBitBackendTLS
}

type FluentBitBackendKafka struct {
	FluentBitBackendCommon
	// specify data format, options available: json, msgpack, default: json
	Format string `json:"format"`
	// Optional key to store the message
	MessageKey string `json:"message_key"`
	// Set the key to store the record timestamp
	TimestampKey string `json:"timestamp_key"`
	// Single of multiple list of kafka brokers
	Brokers string `json:"brokers"`
	// Single entry or list of topics separated by comma(,)
	Topics string `json:"topics"`
}

type FluentBitBackend struct {
	ES    *FluentBitBackendES    `json:"es"`
	Kafka *FluentBitBackendKafka `json:"kafka"`
}

type FluentBit struct {
	Image   FluentBitImage   `json:"image"`
	Backend FluentBitBackend `json:"backend"`
}
