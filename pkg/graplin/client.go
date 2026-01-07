package graplin

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

var (
	ErrRequestFailed = errors.New("request failed")
)

const httpClientTimeout = 5 * time.Second

// Measurement represents a single InfluxDB Line Protocol measurement
type Measurement struct {
	// Measurement (Required) The measurement name. InfluxDB accepts one measurement per point. Measurement names are case-sensitive and subject to naming restrictions.
	Measurement string
	// Tags Optional – All tag key-value pairs for the point. Key-value relationships are denoted with the = operand. Multiple tag key-value pairs are comma-delimited. Tag keys and tag values are case-sensitive. Tag keys are subject to naming restrictions. Tag values cannot be empty; instead, omit the tag from the tag set.
	Tags map[string]string
	// Fields (Required) All field key-value pairs for the point. Points must have at least one field. Field keys and string values are case-sensitive. Field keys are subject to naming restrictions.
	Fields map[string]interface{}
	// Optional – The unix timestamp for the data point. InfluxDB accepts one timestamp per point. If no timestamp is provided, InfluxDB uses the system time (UTC) of its host machine
	Timestamp time.Time
}

// String converts the Measurement to InfluxDB Line Protocol format
func (m *Measurement) String() string {
	line := m.Measurement

	line += m.formatTags()
	line += " " + m.formatFields()

	if !m.Timestamp.IsZero() {
		line += fmt.Sprintf(" %d", m.Timestamp.UnixNano())
	}

	return line
}

func (m *Measurement) formatTags() string {
	if len(m.Tags) == 0 {
		return ""
	}

	tags := make([]string, 0, len(m.Tags))
	for key, value := range m.Tags {
		tags = append(tags, fmt.Sprintf("%s=%s", key, value))
	}

	return "," + strings.Join(tags, ",")
}

func (m *Measurement) formatFields() string {
	fields := make([]string, 0, len(m.Fields))
	for key, value := range m.Fields {
		fields = append(fields, m.formatFieldValue(key, value))
	}

	return strings.Join(fields, ",")
}

func (m *Measurement) formatFieldValue(key string, value interface{}) string {
	switch v := value.(type) {
	case string:
		return fmt.Sprintf("%s=\"%s\"", key, v)
	case int, int32, int64:
		return fmt.Sprintf("%s=%di", key, v)
	case uint, uint32, uint64:
		return fmt.Sprintf("%s=%du", key, v)
	case float32, float64:
		return fmt.Sprintf("%s=%f", key, v)
	case bool:
		return fmt.Sprintf("%s=%t", key, v)
	default:
		return fmt.Sprintf("%s=\"%v\"", key, v)
	}
}

// Client represents the Grafana Cloud InfluxDB Line Protocol Pusher client backed by HTTP Client
type Client struct {
	endpoint   string
	user       string
	token      string
	debug      bool
	httpClient *http.Client
}

// NewClient returns a new client instance that can be configured
// with various options(Functional Options Pattern)
func NewClient(options ...func(client *Client)) *Client {
	client := &Client{}
	client.httpClient = &http.Client{
		Timeout: httpClientTimeout,
	}
	for _, o := range options {
		o(client)
	}
	return client
}

// WithHost sets the Grafana Cloud InfluxDB Line Protocol endpoint host (without path)
func WithHost(host string) func(*Client) {
	return func(c *Client) {
		if host != "" {
			c.endpoint = fmt.Sprintf("%s/api/v1/push/influx/write", host)
			log.Printf("📈 Grafana Push Metrics configured %s", c.endpoint)
		}
	}
}

// WithAuth sets the authentication credentials for Grafana Cloud (user:token format)
func WithAuth(auth string) func(*Client) {
	return func(c *Client) {
		if strings.Contains(auth, ":") {
			c.user = strings.Split(auth, ":")[0]
			c.token = strings.Split(auth, ":")[1]
		} else {
			c.user = auth
		}
	}
}

// WithDebug enables or disables debug mode for the client (more logging)
func WithDebug(debugMode bool) func(*Client) {
	return func(c *Client) {
		c.debug = debugMode
	}
}

// Push sends the given payload formatted  as InfluxDB line protocol to the configured Grafana Cloud Endpoint
func (c *Client) Push(ctx context.Context, m Measurement) error {
	if c.debug {
		log.Printf("📈 Pushing %s metrics %s: %s", m.Measurement, c.endpoint, m.String())
	}
	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewBuffer([]byte(m.String())))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "text/plain")
	req.SetBasicAuth(c.user, c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func(Body io.ReadCloser) { _ = Body.Close() }(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%w with status code: %d", ErrRequestFailed, resp.StatusCode)
	}
	if c.debug {
		log.Printf("📈 %s metrics successfully pushed with status %d", m.Measurement, resp.StatusCode)
	}
	return nil
}
