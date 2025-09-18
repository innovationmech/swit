package bench

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/innovationmech/swit/pkg/messaging/benchmark"
)

func TestLoadConfigAndConvert(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bench.yaml")

	yaml := `timeout: 90s
brokers:
  - name: inmemory
    description: In-memory broker for tests
    topic_prefix: demo
    config:
      type: inmemory
      endpoints: ["localhost"]
workloads:
  - name: sample
    topic: demo.sample
    messages: 42
    message_size: 256
    publishers: 2
    batch_size: 1
`

	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o644))

	cfg, err := loadConfig(path)
	require.NoError(t, err)

	targets, workloads, timeout, err := cfg.toSuiteInputs("default")
	require.NoError(t, err)
	require.Len(t, targets, 1)
	require.Len(t, workloads, 1)
	require.Equal(t, time.Second*90, timeout)

	require.Equal(t, "inmemory", targets[0].Name)
	require.Equal(t, "demo.sample", workloads[0].Topic)
	require.Equal(t, 42, workloads[0].Messages)
}

func TestDefaultProfileFallback(t *testing.T) {
	cfg := &fileConfig{}

	// Without brokers we expect no targets but also no error (checked upstream).
	targets, workloads, _, err := cfg.toSuiteInputs("default")
	require.NoError(t, err)
	require.Empty(t, targets)
	require.Equal(t, benchmark.DefaultWorkloads(), workloads)

	cfg.Brokers = []brokerEntry{{
		Name:   "fake",
		Config: dummyBrokerConfig(),
	}}

	targets, workloads, _, err = cfg.toSuiteInputs("balanced")
	require.NoError(t, err)
	require.Len(t, targets, 1)
	require.Len(t, workloads, 1)
	require.Equal(t, "balanced", workloads[0].Name)
}

func dummyBrokerConfig() *messaging.BrokerConfig {
	cfg := &messaging.BrokerConfig{
		Type:      messaging.BrokerTypeInMemory,
		Endpoints: []string{"localhost"},
	}
	cfg.SetDefaults()
	return cfg
}
