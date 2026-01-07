// Package graplin provides a client for pushing metrics to Grafana Cloud using InfluxDB Line Protocol.
//
// Protocol: https://docs.influxdata.com/influxdb/cloud/reference/syntax/line-protocol/
//
// Example Endpoint:
//
//	https://prometheus-prod-xx-prod-eu-west-2.grafana.net/api/prom/push/influx/write
//
// Example Payload:
//
//	<measurement>[,<tag_key>=<tag_value>[,<tag_key>=<tag_value>]] <field_key>=<field_value>[,<field_key>=<field_value>] [<timestamp>]
//	myMeasurement,tag1=value1,tag2=value2 fieldKey="fieldValue" 1556813561098000000
package graplin
