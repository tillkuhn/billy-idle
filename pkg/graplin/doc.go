// Package graplin (🍇) provides a client for pushing metrics to Grafana Cloud using
// InfluxDB Line Protocol (https://docs.influxdata.com/influxdb/cloud/reference/syntax/line-protocol/)
//
// Grafana Docs: https://grafana.com/docs/grafana-cloud/send-data/metrics/metrics-influxdb/push-from-telegraf/#pushing-from-applications-directly%5BPush
// Example Endpoint: https://prometheus-prod-xx-prod-eu-west-2.grafana.net/api/v1/push/influx/write
//
// Payload anatomy:
//
//	<measurement>[,<tag_key>=<tag_value>[,<tag_key>=<tag_value>]] <field_key>=<field_value>[,<field_key>=<field_value>] [<timestamp>]
//
// Example Line:
//
//	myMeasurement,tag1=value1,tag2=value2 fieldKey="fieldValue" 1556813561098000000
package graplin
