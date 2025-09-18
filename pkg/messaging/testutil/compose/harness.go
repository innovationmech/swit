package compose

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	// DefaultProjectName is used to isolate docker compose state between test runs.
	DefaultProjectName = "swit-messaging"
	// defaultStartTimeout bounds how long the harness waits for all services to become healthy.
	defaultStartTimeout = 2 * time.Minute
	// DefaultKafkaHostPort is the external address exposed by the docker compose stack.
	DefaultKafkaHostPort = "localhost:19092"
	// DefaultRabbitHostPort is the external AMQP endpoint exposed by the stack.
	DefaultRabbitHostPort = "localhost:5672"
	// DefaultRabbitUsername sets the broker default credentials.
	DefaultRabbitUsername = "test"
	// DefaultRabbitPassword sets the broker default credentials.
	DefaultRabbitPassword = "test"
	// DefaultNATSHostPort is the NATS endpoint exposed by the stack.
	DefaultNATSHostPort = "localhost:4222"
)

var (
	// DefaultComposeFile is the absolute path to the docker compose manifest shipped with the repository.
	DefaultComposeFile string
)

func init() {
	// Determine the directory at runtime to keep the harness resilient to working directory changes.
	if _, path, _, ok := runtime.Caller(0); ok {
		DefaultComposeFile = filepath.Join(filepath.Dir(path), "docker-compose.messaging.yml")
	}
}

// Endpoints captures externally accessible addresses for the compose stack services.
type Endpoints struct {
	Kafka  string
	Rabbit string
	NATS   string
}

func (e Endpoints) clone() Endpoints {
	return Endpoints{Kafka: e.Kafka, Rabbit: e.Rabbit, NATS: e.NATS}
}

var defaultEndpoints = Endpoints{
	Kafka:  DefaultKafkaHostPort,
	Rabbit: fmt.Sprintf("amqp://%s:%s@%s/", DefaultRabbitUsername, DefaultRabbitPassword, DefaultRabbitHostPort),
	NATS:   fmt.Sprintf("nats://%s", DefaultNATSHostPort),
}

// Harness manages the lifecycle for the docker compose integration stack.
type Harness struct {
	composeFile string
	projectName string
	services    []string
	env         map[string]string
	endpoints   Endpoints

	mu      sync.Mutex
	started bool
	cmd     *composeCommand
}

// Option mutates harness configuration during construction.
type Option func(*Harness)

// WithComposeFile overrides the docker compose manifest path used by the harness.
func WithComposeFile(path string) Option {
	return func(h *Harness) {
		if path != "" {
			h.composeFile = path
		}
	}
}

// WithProjectName sets a custom docker compose project name.
func WithProjectName(name string) Option {
	return func(h *Harness) {
		if name != "" {
			h.projectName = name
		}
	}
}

// WithServices restricts the set of services managed by the harness.
func WithServices(services ...string) Option {
	return func(h *Harness) {
		if len(services) > 0 {
			h.services = append([]string(nil), services...)
		}
	}
}

// WithEnv injects docker compose environment variables.
func WithEnv(env map[string]string) Option {
	return func(h *Harness) {
		if len(env) == 0 {
			return
		}
		if h.env == nil {
			h.env = make(map[string]string, len(env))
		}
		for k, v := range env {
			h.env[k] = v
		}
	}
}

// WithEndpoints customises service endpoints.
func WithEndpoints(endpoints Endpoints) Option {
	return func(h *Harness) {
		if endpoints.Kafka != "" {
			h.endpoints.Kafka = endpoints.Kafka
		}
		if endpoints.Rabbit != "" {
			h.endpoints.Rabbit = endpoints.Rabbit
		}
		if endpoints.NATS != "" {
			h.endpoints.NATS = endpoints.NATS
		}
	}
}

// NewHarness constructs a Harness with sensible defaults and applied options.
func NewHarness(opts ...Option) *Harness {
	h := &Harness{
		composeFile: DefaultComposeFile,
		projectName: DefaultProjectName,
		services:    []string{"kafka", "rabbitmq", "nats"},
		env:         make(map[string]string),
		endpoints:   defaultEndpoints.clone(),
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// Start launches the compose stack and waits for messaging services to be reachable.
func (h *Harness) Start(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.started {
		return nil
	}

	if err := h.ensureComposeCommand(); err != nil {
		return err
	}

	args := []string{"up", "-d", "--remove-orphans"}
	if len(h.services) > 0 {
		args = append(args, h.services...)
	}

	if err := h.cmd.Run(ctx, h.composeFile, h.projectName, h.env, args...); err != nil {
		return fmt.Errorf("compose up failed: %w", err)
	}

	waitCtx := ctx
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		waitCtx, cancel = context.WithTimeout(ctx, defaultStartTimeout)
		defer cancel()
	}

	if err := WaitForKafka(waitCtx, h.endpoints.Kafka); err != nil {
		_ = h.stopLocked(ctx)
		return fmt.Errorf("kafka readiness failed: %w", err)
	}
	if err := WaitForRabbitMQ(waitCtx, h.endpoints.Rabbit); err != nil {
		_ = h.stopLocked(ctx)
		return fmt.Errorf("rabbitmq readiness failed: %w", err)
	}
	if err := WaitForNATS(waitCtx, h.endpoints.NATS); err != nil {
		_ = h.stopLocked(ctx)
		return fmt.Errorf("nats readiness failed: %w", err)
	}

	h.started = true
	return nil
}

// Stop tears down the compose stack and releases resources.
func (h *Harness) Stop(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.stopLocked(ctx)
}

func (h *Harness) stopLocked(ctx context.Context) error {
	if !h.started {
		return nil
	}
	if err := h.ensureComposeCommand(); err != nil {
		return err
	}
	args := []string{"down", "-v"}
	if err := h.cmd.Run(ctx, h.composeFile, h.projectName, h.env, args...); err != nil {
		return fmt.Errorf("compose down failed: %w", err)
	}
	h.started = false
	return nil
}

func (h *Harness) ensureComposeCommand() error {
	if h.cmd != nil {
		return nil
	}
	cmd, err := detectComposeCommand()
	if err != nil {
		return err
	}
	h.cmd = cmd
	return nil
}

// IsStarted reports whether the harness already launched the stack.
func (h *Harness) IsStarted() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.started
}

// ComposeFile exposes the docker compose manifest path.
func (h *Harness) ComposeFile() string {
	return h.composeFile
}

// ProjectName exposes the docker compose project name.
func (h *Harness) ProjectName() string {
	return h.projectName
}

// Services returns a copy of the managed service identifiers.
func (h *Harness) Services() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]string, len(h.services))
	copy(out, h.services)
	return out
}

// Endpoints returns current service endpoints.
func (h *Harness) Endpoints() Endpoints {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.endpoints.clone()
}

// SetEnv applies or overrides an environment variable used by docker compose.
func (h *Harness) SetEnv(key, value string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.env == nil {
		h.env = make(map[string]string)
	}
	h.env[key] = value
}

// Env returns a copy of the configured environment variables.
func (h *Harness) Env() map[string]string {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make(map[string]string, len(h.env))
	for k, v := range h.env {
		out[k] = v
	}
	return out
}

// ComposeVersion returns the detected docker compose version.
func ComposeVersion(ctx context.Context) (string, error) {
	cmd, err := detectComposeCommand()
	if err != nil {
		return "", err
	}
	args := append([]string(nil), cmd.subcommand...)
	args = append(args, "version", "--short")
	c := exec.CommandContext(ctx, cmd.executable, args...)
	output, err := c.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// composeCommand wraps docker compose invocation details.
type composeCommand struct {
	executable string
	subcommand []string
}

func (c *composeCommand) Run(ctx context.Context, composeFile, project string, env map[string]string, args ...string) error {
	if composeFile == "" {
		return errors.New("compose file is required")
	}
	if project == "" {
		return errors.New("project name is required")
	}

	fullArgs := append([]string(nil), c.subcommand...)
	fullArgs = append(fullArgs, "-f", composeFile, "-p", project)
	fullArgs = append(fullArgs, args...)

	command := exec.CommandContext(ctx, c.executable, fullArgs...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	if len(env) > 0 {
		combined := os.Environ()
		for k, v := range env {
			combined = append(combined, fmt.Sprintf("%s=%s", k, v))
		}
		command.Env = combined
	}

	return command.Run()
}

func detectComposeCommand() (*composeCommand, error) {
	if _, err := exec.LookPath("docker"); err == nil {
		if err := exec.Command("docker", "compose", "version").Run(); err == nil {
			return &composeCommand{executable: "docker", subcommand: []string{"compose"}}, nil
		}
	}

	if _, err := exec.LookPath("docker-compose"); err == nil {
		return &composeCommand{executable: "docker-compose"}, nil
	}

	return nil, ErrDockerComposeMissing
}

// ErrDockerComposeMissing indicates that docker compose could not be located in PATH.
var ErrDockerComposeMissing = errors.New("docker compose is not available in PATH")
