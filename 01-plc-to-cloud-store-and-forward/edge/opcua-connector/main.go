// opcua-connector reads tags from an OPC UA server (here: the CODESYS soft
// PLC running the Factory I/O Level Control scene) and publishes every value
// change to JetStream on the local edge NATS server.
//
// Which tags to read, and the subject each one goes to, come from a YAML
// config file (CONFIG_FILE, default /etc/opcua-connector/config.yaml).
// ${VAR} references in that file are filled from the environment, so the
// same file works on a laptop or on a Raspberry Pi with a different .env.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"gopkg.in/yaml.v3"
)

type config struct {
	OPCUA struct {
		Endpoint           string `yaml:"endpoint"`
		SecurityPolicy     string `yaml:"security_policy"` // None | Basic256Sha256 | ...
		SecurityMode       string `yaml:"security_mode"`   // None | Sign | SignAndEncrypt
		CertFile           string `yaml:"cert_file"`       // only for secure modes
		KeyFile            string `yaml:"key_file"`
		SamplingInterval   string `yaml:"sampling_interval"`
		PublishingInterval string `yaml:"publishing_interval"`
		NodePrefix         string `yaml:"node_prefix"`
	} `yaml:"opcua"`
	NATS struct {
		URL           string `yaml:"url"`
		SubjectPrefix string `yaml:"subject_prefix"`
	} `yaml:"nats"`
	Tags []tag `yaml:"tags"`
}

type tag struct {
	Name string `yaml:"name"` // last subject token, e.g. level_meter
	Node string `yaml:"node"` // appended to node_prefix, e.g. iLevelMeter
}

// reading is the message payload. The field names match the hub's
// InfluxDB writer.
type reading struct {
	Item      string  `json:"item"`
	Value     float64 `json:"value"`
	Timestamp string  `json:"timestamp"` // OPC UA source timestamp
	Source    string  `json:"source"`
	Node      string  `json:"node"`
	Quality   string  `json:"quality"`
}

func main() {
	cfgPath := getenv("CONFIG_FILE", "/etc/opcua-connector/config.yaml")
	cfg, err := loadConfig(cfgPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	sampling := mustDuration(cfg.OPCUA.SamplingInterval, 100*time.Millisecond)
	publishing := mustDuration(cfg.OPCUA.PublishingInterval, 500*time.Millisecond)

	nc := connectNATS(cfg.NATS.URL)
	defer nc.Close()
	js, err := jetstream.New(nc)
	if err != nil {
		log.Fatalf("jetstream init: %v", err)
	}

	log.Printf("%d tags from %s -> %s.<tag>", len(cfg.Tags), cfg.OPCUA.Endpoint, cfg.NATS.SubjectPrefix)
	for {
		if err := run(cfg, sampling, publishing, js); err != nil {
			log.Printf("opcua session ended: %v - reconnecting in 5s", err)
			time.Sleep(5 * time.Second)
		}
	}
}

func loadConfig(path string) (*config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg config
	if err := yaml.Unmarshal([]byte(os.ExpandEnv(string(raw))), &cfg); err != nil {
		return nil, err
	}
	if cfg.OPCUA.Endpoint == "" || cfg.NATS.URL == "" || cfg.NATS.SubjectPrefix == "" {
		return nil, fmt.Errorf("opcua.endpoint, nats.url and nats.subject_prefix are required")
	}
	if len(cfg.Tags) == 0 {
		return nil, fmt.Errorf("no tags configured")
	}
	if cfg.OPCUA.SecurityPolicy == "" {
		cfg.OPCUA.SecurityPolicy = "None"
	}
	if cfg.OPCUA.SecurityMode == "" {
		cfg.OPCUA.SecurityMode = "None"
	}
	return &cfg, nil
}

func connectNATS(url string) *nats.Conn {
	for {
		nc, err := nats.Connect(url, nats.Name("opcua-connector"), nats.MaxReconnects(-1))
		if err == nil {
			log.Printf("connected to NATS at %s", url)
			return nc
		}
		log.Printf("waiting for NATS at %s: %v", url, err)
		time.Sleep(3 * time.Second)
	}
}

func run(cfg *config, sampling, publishing time.Duration, js jetstream.JetStream) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	endpoints, err := opcua.GetEndpoints(ctx, cfg.OPCUA.Endpoint)
	if err != nil {
		return fmt.Errorf("get endpoints: %w", err)
	}
	mode := ua.MessageSecurityModeFromString(cfg.OPCUA.SecurityMode)
	ep, err := opcua.SelectEndpoint(endpoints, cfg.OPCUA.SecurityPolicy, mode)
	if err != nil {
		return fmt.Errorf("select endpoint (%s/%s): %w", cfg.OPCUA.SecurityPolicy, cfg.OPCUA.SecurityMode, err)
	}
	// CODESYS advertises its Windows computer name (e.g. DESKTOP-XXXX),
	// which other machines usually can't resolve. Always dial the address
	// we were given instead.
	ep.EndpointURL = cfg.OPCUA.Endpoint

	opts := []opcua.Option{
		opcua.SecurityPolicy(cfg.OPCUA.SecurityPolicy),
		opcua.SecurityMode(mode),
		opcua.SecurityFromEndpoint(ep, ua.UserTokenTypeAnonymous),
		opcua.AuthAnonymous(),
	}
	if cfg.OPCUA.CertFile != "" && cfg.OPCUA.KeyFile != "" {
		opts = append(opts, opcua.CertificateFile(cfg.OPCUA.CertFile), opcua.PrivateKeyFile(cfg.OPCUA.KeyFile))
	}

	c, err := opcua.NewClient(ep.EndpointURL, opts...)
	if err != nil {
		return err
	}
	if err := c.Connect(ctx); err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer c.Close(ctx)
	log.Printf("connected to OPC UA server at %s (%s/%s)", cfg.OPCUA.Endpoint, cfg.OPCUA.SecurityPolicy, cfg.OPCUA.SecurityMode)

	notifyCh := make(chan *opcua.PublishNotificationData)
	sub, err := c.Subscribe(ctx, &opcua.SubscriptionParameters{Interval: publishing}, notifyCh)
	if err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}
	defer sub.Cancel(ctx)

	reqs := make([]*ua.MonitoredItemCreateRequest, 0, len(cfg.Tags))
	handles := make(map[uint32]tag, len(cfg.Tags))
	for i, t := range cfg.Tags {
		id, err := ua.ParseNodeID(cfg.OPCUA.NodePrefix + t.Node)
		if err != nil {
			return fmt.Errorf("tag %s: %w", t.Name, err)
		}
		h := uint32(i + 1)
		req := opcua.NewMonitoredItemCreateRequestWithDefaults(id, ua.AttributeIDValue, h)
		req.RequestedParameters.SamplingInterval = float64(sampling.Milliseconds())
		reqs = append(reqs, req)
		handles[h] = t
	}
	res, err := sub.Monitor(ctx, ua.TimestampsToReturnBoth, reqs...)
	if err != nil {
		return fmt.Errorf("monitor: %w", err)
	}
	ok := 0
	for i, r := range res.Results {
		if r.StatusCode != ua.StatusOK {
			log.Printf("warning: tag %s (%s) not monitored: %v", cfg.Tags[i].Name, cfg.Tags[i].Node, r.StatusCode)
			continue
		}
		ok++
	}
	log.Printf("monitoring %d/%d tags (sampling %s, publishing %s)", ok, len(cfg.Tags), sampling, publishing)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case n, open := <-notifyCh:
			if !open {
				return fmt.Errorf("notification channel closed")
			}
			if n.Error != nil {
				return fmt.Errorf("subscription: %w", n.Error)
			}
			dcn, isData := n.Value.(*ua.DataChangeNotification)
			if !isData {
				continue
			}
			for _, mi := range dcn.MonitoredItems {
				t, known := handles[mi.ClientHandle]
				if !known || mi.Value == nil || mi.Value.Value == nil {
					continue
				}
				v, isNum := toFloat(mi.Value.Value.Value())
				if !isNum {
					log.Printf("tag %s: unsupported type %T", t.Name, mi.Value.Value.Value())
					continue
				}
				publish(js, cfg, t, v, mi.Value)
			}
		}
	}
}

func publish(js jetstream.JetStream, cfg *config, t tag, v float64, dv *ua.DataValue) {
	ts := dv.SourceTimestamp
	if ts.IsZero() {
		ts = dv.ServerTimestamp
	}
	if ts.IsZero() {
		ts = time.Now()
	}
	quality := "good"
	if dv.Status != ua.StatusOK {
		quality = dv.Status.Error()
	}
	data, err := json.Marshal(reading{
		Item:      t.Name,
		Value:     v,
		Timestamp: ts.UTC().Format(time.RFC3339Nano),
		Source:    "opcua-connector",
		Node:      t.Node,
		Quality:   quality,
	})
	if err != nil {
		log.Printf("marshal: %v", err)
		return
	}
	subject := cfg.NATS.SubjectPrefix + "." + t.Name
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ack, err := js.Publish(ctx, subject, data)
	if err != nil {
		log.Printf("publish %s: %v", subject, err)
		return
	}
	log.Printf("-> %s [%s #%d] %v", subject, ack.Stream, ack.Sequence, v)
}

// toFloat turns any numeric or boolean OPC UA value into a float64, so
// lights and buttons arrive as 1/0 and INT displays as numbers.
func toFloat(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case bool:
		if n {
			return 1, true
		}
		return 0, true
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int8:
		return float64(n), true
	case int16:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint8:
		return float64(n), true
	case uint16:
		return float64(n), true
	case uint32:
		return float64(n), true
	case uint64:
		return float64(n), true
	default:
		return 0, false
	}
}

func mustDuration(s string, def time.Duration) time.Duration {
	if strings.TrimSpace(s) == "" {
		return def
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		log.Fatalf("bad duration %q: %v", s, err)
	}
	return d
}

func getenv(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}
