# Event-Driven Messaging API Specification

## Core Interfaces

### MessageBroker Interface

The central interface for broker operations, providing connection management and factory methods for publishers and subscribers.

```go
package messaging

import (
    "context"
    "time"
)

// MessageBroker defines the core interface for message broker operations
type MessageBroker interface {
    // Connect establishes connection to the message broker
    Connect(ctx context.Context) error
    
    // Disconnect gracefully closes the broker connection
    Disconnect(ctx context.Context) error
    
    // IsConnected returns the current connection status
    IsConnected() bool
    
    // CreatePublisher creates a new publisher instance
    CreatePublisher(config PublisherConfig) (EventPublisher, error)
    
    // CreateSubscriber creates a new subscriber instance
    CreateSubscriber(config SubscriberConfig) (EventSubscriber, error)
    
    // HealthCheck performs a health check on the broker connection
    HealthCheck(ctx context.Context) (*HealthStatus, error)
    
    // GetMetrics returns current broker metrics
    GetMetrics() *BrokerMetrics
    
    // GetCapabilities returns broker-specific capabilities
    GetCapabilities() *BrokerCapabilities
}

// BrokerCapabilities describes what features a broker supports
type BrokerCapabilities struct {
    SupportsTransactions     bool
    SupportsOrdering        bool
    SupportsPartitioning    bool
    SupportsDeadLetter      bool
    SupportsDelayedDelivery bool
    SupportsPriority        bool
    SupportsStreaming       bool
    MaxMessageSize          int64
    MaxBatchSize           int
}

// Example usage:
func ExampleMessageBroker() {
    // Create broker with configuration
    config := &BrokerConfig{
        Type: BrokerTypeKafka,
        Endpoints: []string{"localhost:9092"},
        Authentication: &AuthConfig{
            Type: AuthTypeSASL,
            Username: "user",
            Password: "pass",
        },
    }
    
    broker, err := NewMessageBroker(config)
    if err != nil {
        log.Fatal(err)
    }
    
    // Connect to broker
    ctx := context.Background()
    if err := broker.Connect(ctx); err != nil {
        log.Fatal(err)
    }
    defer broker.Disconnect(ctx)
    
    // Create publisher
    publisher, err := broker.CreatePublisher(PublisherConfig{
        Topic: "user.events",
        Async: true,
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Publish message
    message := &Message{
        ID:      "msg-123",
        Topic:   "user.events",
        Payload: []byte(`{"user_id": "123", "action": "login"}`),
    }
    
    if err := publisher.Publish(ctx, message); err != nil {
        log.Error("Failed to publish:", err)
    }
}
```

### EventPublisher Interface

Defines the contract for publishing messages to the broker.

```go
// EventPublisher defines the interface for publishing messages
type EventPublisher interface {
    // Publish sends a single message
    Publish(ctx context.Context, message *Message) error
    
    // PublishBatch sends multiple messages in a batch
    PublishBatch(ctx context.Context, messages []*Message) error
    
    // PublishWithConfirm sends a message and waits for broker confirmation
    PublishWithConfirm(ctx context.Context, message *Message) (*PublishConfirmation, error)
    
    // PublishAsync sends a message asynchronously with callback
    PublishAsync(ctx context.Context, message *Message, callback PublishCallback) error
    
    // BeginTransaction starts a new transaction (if supported)
    BeginTransaction(ctx context.Context) (Transaction, error)
    
    // Flush ensures all pending messages are sent
    Flush(ctx context.Context) error
    
    // Close closes the publisher and releases resources
    Close() error
    
    // GetMetrics returns publisher metrics
    GetMetrics() *PublisherMetrics
}

// PublishCallback defines the async publish callback
type PublishCallback func(confirmation *PublishConfirmation, err error)

// PublishConfirmation contains publish confirmation details
type PublishConfirmation struct {
    MessageID   string
    Partition   int32
    Offset      int64
    Timestamp   time.Time
    Metadata    map[string]string
}

// Transaction defines the transaction interface
type Transaction interface {
    // Publish publishes a message within the transaction
    Publish(ctx context.Context, message *Message) error
    
    // Commit commits the transaction
    Commit(ctx context.Context) error
    
    // Rollback rolls back the transaction
    Rollback(ctx context.Context) error
}

// Example: Batch publishing with confirmation
func ExampleBatchPublishing() {
    publisher, _ := broker.CreatePublisher(PublisherConfig{
        Topic:     "order.events",
        BatchSize: 100,
        Compression: CompressionSnappy,
    })
    
    // Prepare batch of messages
    messages := make([]*Message, 0, 100)
    for i := 0; i < 100; i++ {
        messages = append(messages, &Message{
            ID:    fmt.Sprintf("order-%d", i),
            Topic: "order.events",
            Key:   []byte(fmt.Sprintf("customer-%d", i%10)),
            Payload: []byte(fmt.Sprintf(`{"order_id": %d, "amount": %.2f}`, i, float64(i)*10.5)),
            Headers: map[string]string{
                "source": "order-service",
                "version": "1.0",
            },
        })
    }
    
    // Publish batch
    if err := publisher.PublishBatch(ctx, messages); err != nil {
        log.Error("Batch publish failed:", err)
    }
}

// Example: Transactional publishing
func ExampleTransactionalPublishing() {
    publisher, _ := broker.CreatePublisher(PublisherConfig{
        Topic: "payment.events",
        Transactional: true,
    })
    
    // Begin transaction
    tx, err := publisher.BeginTransaction(ctx)
    if err != nil {
        log.Fatal(err)
    }
    
    // Publish messages in transaction
    if err := tx.Publish(ctx, &Message{
        ID:      "payment-1",
        Topic:   "payment.events",
        Payload: []byte(`{"type": "debit", "amount": 100}`),
    }); err != nil {
        tx.Rollback(ctx)
        return
    }
    
    if err := tx.Publish(ctx, &Message{
        ID:      "payment-2",
        Topic:   "payment.events",
        Payload: []byte(`{"type": "credit", "amount": 100}`),
    }); err != nil {
        tx.Rollback(ctx)
        return
    }
    
    // Commit transaction
    if err := tx.Commit(ctx); err != nil {
        log.Error("Transaction commit failed:", err)
    }
}
```

### EventSubscriber Interface

Defines the contract for consuming messages from the broker.

```go
// EventSubscriber defines the interface for consuming messages
type EventSubscriber interface {
    // Subscribe starts consuming messages with the given handler
    Subscribe(ctx context.Context, handler MessageHandler) error
    
    // SubscribeWithMiddleware subscribes with middleware chain
    SubscribeWithMiddleware(ctx context.Context, handler MessageHandler, middleware ...Middleware) error
    
    // Unsubscribe stops consuming messages
    Unsubscribe(ctx context.Context) error
    
    // Pause temporarily stops message consumption
    Pause(ctx context.Context) error
    
    // Resume resumes message consumption
    Resume(ctx context.Context) error
    
    // Seek seeks to a specific position (if supported)
    Seek(ctx context.Context, position SeekPosition) error
    
    // GetLag returns the current consumer lag
    GetLag(ctx context.Context) (int64, error)
    
    // Close closes the subscriber and releases resources
    Close() error
    
    // GetMetrics returns subscriber metrics
    GetMetrics() *SubscriberMetrics
}

// MessageHandler defines the interface for handling messages
type MessageHandler interface {
    // Handle processes a message
    Handle(ctx context.Context, message *Message) error
    
    // OnError handles processing errors
    OnError(ctx context.Context, message *Message, err error) ErrorAction
}

// ErrorAction defines what to do when message processing fails
type ErrorAction int

const (
    ErrorActionRetry ErrorAction = iota
    ErrorActionDeadLetter
    ErrorActionDiscard
    ErrorActionPause
)

// SeekPosition defines where to seek in the message stream
type SeekPosition struct {
    Type      SeekType
    Offset    int64
    Timestamp time.Time
    Position  string // broker-specific position
}

type SeekType int

const (
    SeekTypeBeginning SeekType = iota
    SeekTypeEnd
    SeekTypeOffset
    SeekTypeTimestamp
    SeekTypePosition
)

// Example: Message subscription with handler
func ExampleMessageSubscription() {
    // Create subscriber
    subscriber, _ := broker.CreateSubscriber(SubscriberConfig{
        Topics:        []string{"user.events", "order.events"},
        ConsumerGroup: "analytics-service",
        Concurrency:   10,
        PrefetchCount: 100,
    })
    
    // Define message handler
    handler := &AnalyticsHandler{
        processor: NewEventProcessor(),
        storage:   NewEventStorage(),
    }
    
    // Subscribe with middleware
    err := subscriber.SubscribeWithMiddleware(
        ctx,
        handler,
        LoggingMiddleware(),
        MetricsMiddleware(),
        RetryMiddleware(3, time.Second),
        DeadLetterMiddleware("dlq.analytics"),
    )
    if err != nil {
        log.Fatal(err)
    }
    
    // Handle graceful shutdown
    <-shutdownSignal
    subscriber.Unsubscribe(ctx)
}

// AnalyticsHandler implements MessageHandler
type AnalyticsHandler struct {
    processor EventProcessor
    storage   EventStorage
}

func (h *AnalyticsHandler) Handle(ctx context.Context, message *Message) error {
    // Process the event
    event, err := h.processor.Parse(message.Payload)
    if err != nil {
        return fmt.Errorf("failed to parse event: %w", err)
    }
    
    // Store for analytics
    if err := h.storage.Store(ctx, event); err != nil {
        return fmt.Errorf("failed to store event: %w", err)
    }
    
    return nil
}

func (h *AnalyticsHandler) OnError(ctx context.Context, message *Message, err error) ErrorAction {
    // Determine error handling strategy
    if errors.Is(err, ErrInvalidFormat) {
        // Can't retry invalid messages
        return ErrorActionDeadLetter
    }
    
    if errors.Is(err, ErrStorageUnavailable) {
        // Retry transient errors
        return ErrorActionRetry
    }
    
    // Default to dead letter
    return ErrorActionDeadLetter
}
```

## Message Types and Formats

### Core Message Structure

```go
// Message represents a message in the messaging system
type Message struct {
    // Unique message identifier
    ID string `json:"id"`
    
    // Topic/queue name
    Topic string `json:"topic"`
    
    // Optional routing/partition key
    Key []byte `json:"key,omitempty"`
    
    // Message payload
    Payload []byte `json:"payload"`
    
    // Message headers/metadata
    Headers map[string]string `json:"headers,omitempty"`
    
    // Timestamp when message was created
    Timestamp time.Time `json:"timestamp"`
    
    // Optional correlation ID for request-response patterns
    CorrelationID string `json:"correlation_id,omitempty"`
    
    // Optional reply-to topic for request-response patterns
    ReplyTo string `json:"reply_to,omitempty"`
    
    // Message priority (if supported by broker)
    Priority int `json:"priority,omitempty"`
    
    // TTL in seconds (if supported by broker)
    TTL int `json:"ttl,omitempty"`
    
    // Delivery attempt count
    DeliveryAttempt int `json:"delivery_attempt,omitempty"`
    
    // Broker-specific metadata
    BrokerMetadata interface{} `json:"-"`
}

// CloudEvent represents a CloudEvents formatted message
type CloudEvent struct {
    SpecVersion     string                 `json:"specversion"`
    Type           string                 `json:"type"`
    Source         string                 `json:"source"`
    Subject        string                 `json:"subject,omitempty"`
    ID             string                 `json:"id"`
    Time           time.Time              `json:"time"`
    DataContentType string                 `json:"datacontenttype,omitempty"`
    DataSchema     string                 `json:"dataschema,omitempty"`
    Data           interface{}            `json:"data"`
    Extensions     map[string]interface{} `json:"-"`
}

// Example: Creating and sending a CloudEvent
func ExampleCloudEvent() {
    event := &CloudEvent{
        SpecVersion: "1.0",
        Type:        "com.example.user.created",
        Source:      "user-service",
        Subject:     "user-123",
        ID:          uuid.New().String(),
        Time:        time.Now(),
        DataContentType: "application/json",
        Data: map[string]interface{}{
            "user_id": "123",
            "email":   "user@example.com",
            "name":    "John Doe",
        },
    }
    
    // Convert to Message
    payload, _ := json.Marshal(event)
    message := &Message{
        ID:      event.ID,
        Topic:   "user.events",
        Payload: payload,
        Headers: map[string]string{
            "ce-specversion": event.SpecVersion,
            "ce-type":        event.Type,
            "ce-source":      event.Source,
        },
    }
    
    publisher.Publish(ctx, message)
}
```

### Message Serialization

```go
// Serializer defines the interface for message serialization
type Serializer interface {
    Serialize(data interface{}) ([]byte, error)
    Deserialize(data []byte, target interface{}) error
    ContentType() string
}

// JSONSerializer implements JSON serialization
type JSONSerializer struct{}

func (s *JSONSerializer) Serialize(data interface{}) ([]byte, error) {
    return json.Marshal(data)
}

func (s *JSONSerializer) Deserialize(data []byte, target interface{}) error {
    return json.Unmarshal(data, target)
}

func (s *JSONSerializer) ContentType() string {
    return "application/json"
}

// ProtobufSerializer implements Protocol Buffer serialization
type ProtobufSerializer struct{}

func (s *ProtobufSerializer) Serialize(data interface{}) ([]byte, error) {
    msg, ok := data.(proto.Message)
    if !ok {
        return nil, errors.New("data must implement proto.Message")
    }
    return proto.Marshal(msg)
}

func (s *ProtobufSerializer) Deserialize(data []byte, target interface{}) error {
    msg, ok := target.(proto.Message)
    if !ok {
        return errors.New("target must implement proto.Message")
    }
    return proto.Unmarshal(data, msg)
}

func (s *ProtobufSerializer) ContentType() string {
    return "application/x-protobuf"
}

// AvroSerializer implements Apache Avro serialization
type AvroSerializer struct {
    schemaRegistry SchemaRegistry
}

func (s *AvroSerializer) Serialize(data interface{}) ([]byte, error) {
    schema, err := s.schemaRegistry.GetSchema(data)
    if err != nil {
        return nil, err
    }
    return avro.Marshal(schema, data)
}

func (s *AvroSerializer) Deserialize(data []byte, target interface{}) error {
    schema, err := s.schemaRegistry.GetSchema(target)
    if err != nil {
        return err
    }
    return avro.Unmarshal(schema, data, target)
}

func (s *AvroSerializer) ContentType() string {
    return "application/avro"
}
```

## Configuration Structures

### Broker Configuration

```go
// BrokerConfig defines the broker configuration
type BrokerConfig struct {
    // Broker type (kafka, rabbitmq, nats)
    Type BrokerType `yaml:"type" validate:"required,oneof=kafka rabbitmq nats"`
    
    // Broker endpoints
    Endpoints []string `yaml:"endpoints" validate:"required,min=1"`
    
    // Connection configuration
    Connection ConnectionConfig `yaml:"connection"`
    
    // Authentication configuration
    Authentication *AuthConfig `yaml:"authentication,omitempty"`
    
    // TLS configuration
    TLS *TLSConfig `yaml:"tls,omitempty"`
    
    // Retry configuration
    Retry RetryConfig `yaml:"retry"`
    
    // Monitoring configuration
    Monitoring MonitoringConfig `yaml:"monitoring"`
    
    // Broker-specific configuration
    Extra map[string]interface{} `yaml:"extra,omitempty"`
}

// ConnectionConfig defines connection settings
type ConnectionConfig struct {
    // Connection timeout
    Timeout time.Duration `yaml:"timeout" default:"10s"`
    
    // Keep-alive interval
    KeepAlive time.Duration `yaml:"keep_alive" default:"30s"`
    
    // Maximum connection attempts
    MaxAttempts int `yaml:"max_attempts" default:"3"`
    
    // Connection pool size
    PoolSize int `yaml:"pool_size" default:"10"`
    
    // Idle connection timeout
    IdleTimeout time.Duration `yaml:"idle_timeout" default:"5m"`
}

// AuthConfig defines authentication settings
type AuthConfig struct {
    Type     AuthType `yaml:"type" validate:"required,oneof=none sasl oauth2 apikey"`
    Username string   `yaml:"username,omitempty"`
    Password string   `yaml:"password,omitempty"`
    Token    string   `yaml:"token,omitempty"`
    APIKey   string   `yaml:"api_key,omitempty"`
    
    // SASL specific
    Mechanism string `yaml:"mechanism,omitempty"` // PLAIN, SCRAM-SHA-256, SCRAM-SHA-512
    
    // OAuth2 specific
    ClientID     string `yaml:"client_id,omitempty"`
    ClientSecret string `yaml:"client_secret,omitempty"`
    TokenURL     string `yaml:"token_url,omitempty"`
    Scopes       []string `yaml:"scopes,omitempty"`
}

// TLSConfig defines TLS settings
type TLSConfig struct {
    Enabled    bool   `yaml:"enabled"`
    CertFile   string `yaml:"cert_file,omitempty"`
    KeyFile    string `yaml:"key_file,omitempty"`
    CAFile     string `yaml:"ca_file,omitempty"`
    ServerName string `yaml:"server_name,omitempty"`
    SkipVerify bool   `yaml:"skip_verify"`
}

// RetryConfig defines retry settings
type RetryConfig struct {
    MaxAttempts  int           `yaml:"max_attempts" default:"3"`
    InitialDelay time.Duration `yaml:"initial_delay" default:"1s"`
    MaxDelay     time.Duration `yaml:"max_delay" default:"30s"`
    Multiplier   float64       `yaml:"multiplier" default:"2.0"`
    Jitter       float64       `yaml:"jitter" default:"0.1"`
}

// Example: Complete broker configuration
func ExampleBrokerConfiguration() {
    config := &BrokerConfig{
        Type: BrokerTypeKafka,
        Endpoints: []string{
            "kafka-1.example.com:9092",
            "kafka-2.example.com:9092",
            "kafka-3.example.com:9092",
        },
        Connection: ConnectionConfig{
            Timeout:     10 * time.Second,
            KeepAlive:   30 * time.Second,
            MaxAttempts: 5,
            PoolSize:    20,
        },
        Authentication: &AuthConfig{
            Type:      AuthTypeSASL,
            Mechanism: "SCRAM-SHA-512",
            Username:  os.Getenv("KAFKA_USERNAME"),
            Password:  os.Getenv("KAFKA_PASSWORD"),
        },
        TLS: &TLSConfig{
            Enabled:  true,
            CertFile: "/certs/client.crt",
            KeyFile:  "/certs/client.key",
            CAFile:   "/certs/ca.crt",
        },
        Retry: RetryConfig{
            MaxAttempts:  5,
            InitialDelay: 500 * time.Millisecond,
            MaxDelay:     30 * time.Second,
            Multiplier:   2.0,
            Jitter:       0.1,
        },
        Extra: map[string]interface{}{
            "client.id":                    "user-service",
            "metadata.max.age.ms":          300000,
            "socket.keepalive.enable":      true,
            "api.version.request":          true,
        },
    }
}
```

### Publisher Configuration

```go
// PublisherConfig defines publisher configuration
type PublisherConfig struct {
    // Topic/queue to publish to
    Topic string `yaml:"topic" validate:"required"`
    
    // Routing configuration
    Routing RoutingConfig `yaml:"routing"`
    
    // Batching configuration
    Batching BatchingConfig `yaml:"batching"`
    
    // Compression settings
    Compression CompressionType `yaml:"compression" default:"none"`
    
    // Async publishing
    Async bool `yaml:"async" default:"false"`
    
    // Transactional publishing
    Transactional bool `yaml:"transactional" default:"false"`
    
    // Confirmation settings
    Confirmation ConfirmationConfig `yaml:"confirmation"`
    
    // Retry configuration
    Retry RetryConfig `yaml:"retry"`
    
    // Timeout settings
    Timeout TimeoutConfig `yaml:"timeout"`
}

// BatchingConfig defines batching settings
type BatchingConfig struct {
    Enabled      bool          `yaml:"enabled" default:"true"`
    MaxMessages  int           `yaml:"max_messages" default:"100"`
    MaxBytes     int           `yaml:"max_bytes" default:"1048576"` // 1MB
    FlushInterval time.Duration `yaml:"flush_interval" default:"100ms"`
}

// ConfirmationConfig defines confirmation settings
type ConfirmationConfig struct {
    Required bool          `yaml:"required" default:"false"`
    Timeout  time.Duration `yaml:"timeout" default:"5s"`
    Retries  int           `yaml:"retries" default:"3"`
}

// Example: Publisher configuration for high-throughput
func ExampleHighThroughputPublisher() {
    config := PublisherConfig{
        Topic: "events.stream",
        Batching: BatchingConfig{
            Enabled:       true,
            MaxMessages:   1000,
            MaxBytes:      5 * 1024 * 1024, // 5MB
            FlushInterval: 50 * time.Millisecond,
        },
        Compression: CompressionSnappy,
        Async:       true,
        Confirmation: ConfirmationConfig{
            Required: false, // Fire and forget for max throughput
        },
    }
    
    publisher, _ := broker.CreatePublisher(config)
}
```

### Subscriber Configuration

```go
// SubscriberConfig defines subscriber configuration
type SubscriberConfig struct {
    // Topics/queues to subscribe to
    Topics []string `yaml:"topics" validate:"required,min=1"`
    
    // Consumer group name
    ConsumerGroup string `yaml:"consumer_group" validate:"required"`
    
    // Subscription type
    Type SubscriptionType `yaml:"type" default:"shared"`
    
    // Concurrency settings
    Concurrency int `yaml:"concurrency" default:"1"`
    
    // Prefetch settings
    PrefetchCount int `yaml:"prefetch_count" default:"10"`
    
    // Processing configuration
    Processing ProcessingConfig `yaml:"processing"`
    
    // Dead letter configuration
    DeadLetter DeadLetterConfig `yaml:"dead_letter"`
    
    // Offset management
    Offset OffsetConfig `yaml:"offset"`
    
    // Retry configuration
    Retry RetryConfig `yaml:"retry"`
}

// ProcessingConfig defines message processing settings
type ProcessingConfig struct {
    // Maximum processing time per message
    MaxProcessingTime time.Duration `yaml:"max_processing_time" default:"30s"`
    
    // Acknowledgment mode
    AckMode AckMode `yaml:"ack_mode" default:"auto"`
    
    // Maximum in-flight messages
    MaxInFlight int `yaml:"max_in_flight" default:"100"`
    
    // Ordered processing
    Ordered bool `yaml:"ordered" default:"false"`
}

// DeadLetterConfig defines dead letter settings
type DeadLetterConfig struct {
    Enabled    bool   `yaml:"enabled" default:"true"`
    Topic      string `yaml:"topic"`
    MaxRetries int    `yaml:"max_retries" default:"3"`
    TTL        time.Duration `yaml:"ttl" default:"7d"`
}

// OffsetConfig defines offset management settings
type OffsetConfig struct {
    Initial    OffsetPosition `yaml:"initial" default:"latest"`
    AutoCommit bool          `yaml:"auto_commit" default:"true"`
    Interval   time.Duration `yaml:"interval" default:"5s"`
}

// Example: Subscriber configuration for reliable processing
func ExampleReliableSubscriber() {
    config := SubscriberConfig{
        Topics:        []string{"order.events"},
        ConsumerGroup: "order-processor",
        Type:          SubscriptionShared,
        Concurrency:   5,
        PrefetchCount: 20,
        Processing: ProcessingConfig{
            MaxProcessingTime: 1 * time.Minute,
            AckMode:          AckModeManual,
            MaxInFlight:      50,
            Ordered:          true,
        },
        DeadLetter: DeadLetterConfig{
            Enabled:    true,
            Topic:      "order.events.dlq",
            MaxRetries: 5,
            TTL:        7 * 24 * time.Hour,
        },
        Offset: OffsetConfig{
            Initial:    OffsetEarliest,
            AutoCommit: false,
        },
    }
    
    subscriber, _ := broker.CreateSubscriber(config)
}
```

## Error Types and Handling

### Error Types

```go
// MessagingError is the base error type for messaging errors
type MessagingError struct {
    Code      ErrorCode
    Message   string
    Cause     error
    Retryable bool
    Details   map[string]interface{}
}

func (e *MessagingError) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("%s: %s (cause: %v)", e.Code, e.Message, e.Cause)
    }
    return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *MessagingError) Unwrap() error {
    return e.Cause
}

// ErrorCode defines error codes
type ErrorCode string

const (
    // Connection errors
    ErrConnectionFailed   ErrorCode = "CONNECTION_FAILED"
    ErrConnectionTimeout  ErrorCode = "CONNECTION_TIMEOUT"
    ErrConnectionLost     ErrorCode = "CONNECTION_LOST"
    
    // Authentication errors
    ErrAuthenticationFailed ErrorCode = "AUTH_FAILED"
    ErrAuthorizationDenied  ErrorCode = "AUTH_DENIED"
    
    // Publishing errors
    ErrPublishFailed      ErrorCode = "PUBLISH_FAILED"
    ErrPublishTimeout     ErrorCode = "PUBLISH_TIMEOUT"
    ErrMessageTooLarge    ErrorCode = "MESSAGE_TOO_LARGE"
    ErrTopicNotFound      ErrorCode = "TOPIC_NOT_FOUND"
    
    // Subscription errors
    ErrSubscriptionFailed ErrorCode = "SUBSCRIPTION_FAILED"
    ErrConsumerGroupError ErrorCode = "CONSUMER_GROUP_ERROR"
    ErrRebalancing       ErrorCode = "REBALANCING"
    
    // Processing errors
    ErrProcessingFailed  ErrorCode = "PROCESSING_FAILED"
    ErrProcessingTimeout ErrorCode = "PROCESSING_TIMEOUT"
    ErrInvalidMessage    ErrorCode = "INVALID_MESSAGE"
    
    // Transaction errors
    ErrTransactionFailed    ErrorCode = "TRANSACTION_FAILED"
    ErrTransactionAborted   ErrorCode = "TRANSACTION_ABORTED"
    ErrTransactionTimeout   ErrorCode = "TRANSACTION_TIMEOUT"
)

// Common error constructors
func NewConnectionError(message string, cause error) *MessagingError {
    return &MessagingError{
        Code:      ErrConnectionFailed,
        Message:   message,
        Cause:     cause,
        Retryable: true,
    }
}

func NewPublishError(message string, cause error) *MessagingError {
    return &MessagingError{
        Code:      ErrPublishFailed,
        Message:   message,
        Cause:     cause,
        Retryable: true,
    }
}

func NewProcessingError(message string, cause error) *MessagingError {
    return &MessagingError{
        Code:      ErrProcessingFailed,
        Message:   message,
        Cause:     cause,
        Retryable: false,
    }
}

// Example: Error handling in message processing
func ExampleErrorHandling() {
    handler := func(ctx context.Context, msg *Message) error {
        // Process message
        if err := processMessage(msg); err != nil {
            // Check error type
            var msgErr *MessagingError
            if errors.As(err, &msgErr) {
                if msgErr.Retryable {
                    // Retryable error - let middleware handle retry
                    return err
                }
                
                // Non-retryable - send to DLQ
                if err := sendToDeadLetter(msg, err); err != nil {
                    log.Error("Failed to send to DLQ:", err)
                }
                
                // Don't retry
                return nil
            }
            
            // Unknown error - retry
            return err
        }
        
        return nil
    }
}
```

## Middleware Interfaces

### Core Middleware Interface

```go
// Middleware defines the interface for message processing middleware
type Middleware interface {
    // Name returns the middleware name
    Name() string
    
    // Wrap wraps a handler with middleware logic
    Wrap(next MessageHandler) MessageHandler
}

// MiddlewareChain manages a chain of middleware
type MiddlewareChain struct {
    middlewares []Middleware
}

func NewMiddlewareChain(middlewares ...Middleware) *MiddlewareChain {
    return &MiddlewareChain{
        middlewares: middlewares,
    }
}

func (c *MiddlewareChain) Then(handler MessageHandler) MessageHandler {
    // Build middleware chain in reverse order
    for i := len(c.middlewares) - 1; i >= 0; i-- {
        handler = c.middlewares[i].Wrap(handler)
    }
    return handler
}

// Example: Custom middleware implementation
type LoggingMiddleware struct {
    logger *zap.Logger
}

func (m *LoggingMiddleware) Name() string {
    return "logging"
}

func (m *LoggingMiddleware) Wrap(next MessageHandler) MessageHandler {
    return MessageHandlerFunc(func(ctx context.Context, msg *Message) error {
        start := time.Now()
        
        m.logger.Info("Processing message",
            zap.String("id", msg.ID),
            zap.String("topic", msg.Topic),
            zap.Int("size", len(msg.Payload)),
        )
        
        err := next.Handle(ctx, msg)
        
        m.logger.Info("Message processed",
            zap.String("id", msg.ID),
            zap.Duration("duration", time.Since(start)),
            zap.Error(err),
        )
        
        return err
    })
}
```

### Built-in Middleware

```go
// RetryMiddleware implements retry logic
func RetryMiddleware(maxAttempts int, delay time.Duration) Middleware {
    return &retryMiddleware{
        maxAttempts: maxAttempts,
        delay:       delay,
    }
}

// MetricsMiddleware collects processing metrics
func MetricsMiddleware(collector MetricsCollector) Middleware {
    return &metricsMiddleware{
        collector: collector,
    }
}

// TracingMiddleware adds distributed tracing
func TracingMiddleware(tracer opentracing.Tracer) Middleware {
    return &tracingMiddleware{
        tracer: tracer,
    }
}

// ValidationMiddleware validates message format
func ValidationMiddleware(validator MessageValidator) Middleware {
    return &validationMiddleware{
        validator: validator,
    }
}

// DeadLetterMiddleware handles failed messages
func DeadLetterMiddleware(dlqTopic string, publisher EventPublisher) Middleware {
    return &deadLetterMiddleware{
        dlqTopic:  dlqTopic,
        publisher: publisher,
    }
}

// RateLimitMiddleware implements rate limiting
func RateLimitMiddleware(limiter RateLimiter) Middleware {
    return &rateLimitMiddleware{
        limiter: limiter,
    }
}

// Example: Building a middleware chain
func ExampleMiddlewareChain() {
    // Create middleware chain
    chain := NewMiddlewareChain(
        LoggingMiddleware(logger),
        MetricsMiddleware(metricsCollector),
        TracingMiddleware(tracer),
        ValidationMiddleware(validator),
        RetryMiddleware(3, time.Second),
        DeadLetterMiddleware("dlq.topic", dlqPublisher),
        RateLimitMiddleware(rateLimiter),
    )
    
    // Create base handler
    handler := &MyMessageHandler{}
    
    // Wrap handler with middleware chain
    wrappedHandler := chain.Then(handler)
    
    // Subscribe with wrapped handler
    subscriber.Subscribe(ctx, wrappedHandler)
}
```

### Context Propagation

```go
// ContextPropagator handles context propagation across message boundaries
type ContextPropagator interface {
    // Inject injects context into message headers
    Inject(ctx context.Context, headers map[string]string) error
    
    // Extract extracts context from message headers
    Extract(headers map[string]string) (context.Context, error)
}

// TracingContextPropagator propagates tracing context
type TracingContextPropagator struct {
    tracer opentracing.Tracer
}

func (p *TracingContextPropagator) Inject(ctx context.Context, headers map[string]string) error {
    span := opentracing.SpanFromContext(ctx)
    if span == nil {
        return nil
    }
    
    carrier := opentracing.TextMapCarrier(headers)
    return p.tracer.Inject(span.Context(), opentracing.TextMap, carrier)
}

func (p *TracingContextPropagator) Extract(headers map[string]string) (context.Context, error) {
    carrier := opentracing.TextMapCarrier(headers)
    spanCtx, err := p.tracer.Extract(opentracing.TextMap, carrier)
    if err != nil {
        // No span context in headers
        return context.Background(), nil
    }
    
    span := p.tracer.StartSpan("message.process", opentracing.ChildOf(spanCtx))
    return opentracing.ContextWithSpan(context.Background(), span), nil
}

// Example: Using context propagation
func ExampleContextPropagation() {
    propagator := &TracingContextPropagator{tracer: tracer}
    
    // Publishing with context
    publisher := func(ctx context.Context, msg *Message) error {
        // Inject tracing context
        if err := propagator.Inject(ctx, msg.Headers); err != nil {
            log.Warn("Failed to inject context:", err)
        }
        
        return eventPublisher.Publish(ctx, msg)
    }
    
    // Consuming with context
    handler := func(ctx context.Context, msg *Message) error {
        // Extract tracing context
        ctx, err := propagator.Extract(msg.Headers)
        if err != nil {
            log.Warn("Failed to extract context:", err)
        }
        
        // Process with traced context
        span := opentracing.SpanFromContext(ctx)
        defer span.Finish()
        
        return processMessage(ctx, msg)
    }
}
```

## Complete Code Examples

### Example 1: Building a Complete Event-Driven Service

```go
package main

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "syscall"
    "time"
    
    "github.com/swit/pkg/messaging"
    "github.com/swit/pkg/server"
    "go.uber.org/zap"
)

// OrderService demonstrates a complete event-driven service
type OrderService struct {
    broker     messaging.MessageBroker
    publisher  messaging.EventPublisher
    logger     *zap.Logger
    repository OrderRepository
}

// RegisterServices implements BusinessServiceRegistrar
func (s *OrderService) RegisterServices(registry server.BusinessServiceRegistry) error {
    // Register HTTP handlers
    if err := registry.RegisterBusinessHTTPHandler(s); err != nil {
        return err
    }
    
    // Register messaging
    coordinator := registry.GetMessagingCoordinator()
    if err := s.RegisterMessaging(coordinator); err != nil {
        return err
    }
    
    return nil
}

// RegisterMessaging registers message handlers
func (s *OrderService) RegisterMessaging(coordinator *messaging.MessagingCoordinator) error {
    // Create subscriber for payment events
    paymentSubscriber, err := s.broker.CreateSubscriber(messaging.SubscriberConfig{
        Topics:        []string{"payment.events"},
        ConsumerGroup: "order-service",
        Concurrency:   5,
    })
    if err != nil {
        return err
    }
    
    // Subscribe with middleware
    chain := messaging.NewMiddlewareChain(
        messaging.LoggingMiddleware(s.logger),
        messaging.MetricsMiddleware(metrics),
        messaging.RetryMiddleware(3, time.Second),
        messaging.DeadLetterMiddleware("payment.events.dlq", s.publisher),
    )
    
    handler := &PaymentEventHandler{
        service: s,
    }
    
    return paymentSubscriber.SubscribeWithMiddleware(
        context.Background(),
        handler,
        chain.middlewares...,
    )
}

// PaymentEventHandler handles payment events
type PaymentEventHandler struct {
    service *OrderService
}

func (h *PaymentEventHandler) Handle(ctx context.Context, msg *messaging.Message) error {
    // Parse payment event
    var event PaymentEvent
    if err := json.Unmarshal(msg.Payload, &event); err != nil {
        return messaging.NewProcessingError("invalid payment event", err)
    }
    
    // Update order status
    order, err := h.service.repository.GetOrder(ctx, event.OrderID)
    if err != nil {
        return fmt.Errorf("failed to get order: %w", err)
    }
    
    switch event.Status {
    case "completed":
        order.Status = OrderStatusPaid
        
        // Publish order completed event
        orderEvent := OrderCompletedEvent{
            OrderID:   order.ID,
            UserID:    order.UserID,
            Amount:    order.Amount,
            Timestamp: time.Now(),
        }
        
        if err := h.service.PublishOrderEvent(ctx, orderEvent); err != nil {
            return fmt.Errorf("failed to publish order event: %w", err)
        }
        
    case "failed":
        order.Status = OrderStatusPaymentFailed
        
    default:
        return messaging.NewProcessingError(
            fmt.Sprintf("unknown payment status: %s", event.Status),
            nil,
        )
    }
    
    // Save order
    if err := h.service.repository.SaveOrder(ctx, order); err != nil {
        return fmt.Errorf("failed to save order: %w", err)
    }
    
    h.service.logger.Info("Payment event processed",
        zap.String("order_id", event.OrderID),
        zap.String("status", event.Status),
    )
    
    return nil
}

func (h *PaymentEventHandler) OnError(ctx context.Context, msg *messaging.Message, err error) messaging.ErrorAction {
    var msgErr *messaging.MessagingError
    if errors.As(err, &msgErr) && !msgErr.Retryable {
        // Non-retryable error - send to DLQ
        return messaging.ErrorActionDeadLetter
    }
    
    // Retry transient errors
    return messaging.ErrorActionRetry
}

// PublishOrderEvent publishes order events
func (s *OrderService) PublishOrderEvent(ctx context.Context, event interface{}) error {
    // Serialize event
    payload, err := json.Marshal(event)
    if err != nil {
        return err
    }
    
    // Create message
    msg := &messaging.Message{
        ID:      uuid.New().String(),
        Topic:   "order.events",
        Payload: payload,
        Headers: map[string]string{
            "source":  "order-service",
            "version": "1.0",
            "type":    fmt.Sprintf("%T", event),
        },
        Timestamp: time.Now(),
    }
    
    // Publish with confirmation
    confirmation, err := s.publisher.PublishWithConfirm(ctx, msg)
    if err != nil {
        return err
    }
    
    s.logger.Info("Order event published",
        zap.String("event_id", msg.ID),
        zap.String("confirmation_id", confirmation.MessageID),
    )
    
    return nil
}

func main() {
    // Load configuration
    config := loadConfig()
    
    // Create broker
    broker, err := messaging.NewMessageBroker(config.Messaging.Broker)
    if err != nil {
        log.Fatal(err)
    }
    
    // Connect to broker
    ctx := context.Background()
    if err := broker.Connect(ctx); err != nil {
        log.Fatal(err)
    }
    defer broker.Disconnect(ctx)
    
    // Create publisher
    publisher, err := broker.CreatePublisher(config.Messaging.Publisher)
    if err != nil {
        log.Fatal(err)
    }
    
    // Create service
    service := &OrderService{
        broker:     broker,
        publisher:  publisher,
        logger:     logger,
        repository: repository,
    }
    
    // Create and start server
    server, err := server.NewBusinessServerCore(config.Server, service, nil)
    if err != nil {
        log.Fatal(err)
    }
    
    // Start server
    if err := server.Start(ctx); err != nil {
        log.Fatal(err)
    }
    
    // Wait for shutdown signal
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    <-sigChan
    
    // Graceful shutdown
    shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    
    if err := server.Stop(shutdownCtx); err != nil {
        log.Error("Server shutdown error:", err)
    }
}
```

### Example 2: Request-Response Pattern

```go
// RequestResponseExample demonstrates request-response messaging pattern
func RequestResponseExample() {
    // Create request publisher
    requestPublisher, _ := broker.CreatePublisher(messaging.PublisherConfig{
        Topic: "requests",
    })
    
    // Create response subscriber
    responseSubscriber, _ := broker.CreateSubscriber(messaging.SubscriberConfig{
        Topics:        []string{"responses"},
        ConsumerGroup: "requester",
    })
    
    // Send request
    correlationID := uuid.New().String()
    request := &messaging.Message{
        ID:            uuid.New().String(),
        Topic:         "requests",
        Payload:       []byte(`{"action": "get_user", "user_id": "123"}`),
        CorrelationID: correlationID,
        ReplyTo:       "responses",
    }
    
    // Set up response handler
    responseChan := make(chan *messaging.Message)
    responseHandler := messaging.MessageHandlerFunc(func(ctx context.Context, msg *messaging.Message) error {
        if msg.CorrelationID == correlationID {
            responseChan <- msg
        }
        return nil
    })
    
    // Subscribe for responses
    responseSubscriber.Subscribe(context.Background(), responseHandler)
    
    // Send request
    requestPublisher.Publish(context.Background(), request)
    
    // Wait for response
    select {
    case response := <-responseChan:
        fmt.Printf("Received response: %s\n", response.Payload)
    case <-time.After(5 * time.Second):
        fmt.Println("Request timeout")
    }
}
```

### Example 3: Event Sourcing Implementation

```go
// EventStore implements event sourcing pattern
type EventStore struct {
    publisher  messaging.EventPublisher
    subscriber messaging.EventSubscriber
    storage    EventStorage
    snapshots  SnapshotStore
}

// AppendEvent appends an event to the event stream
func (es *EventStore) AppendEvent(ctx context.Context, aggregateID string, event Event) error {
    // Store event
    if err := es.storage.Store(ctx, event); err != nil {
        return err
    }
    
    // Publish event
    msg := &messaging.Message{
        ID:    event.ID,
        Topic: fmt.Sprintf("events.%s", event.AggregateType),
        Key:   []byte(aggregateID),
        Payload: event.Data,
        Headers: map[string]string{
            "aggregate_id":   aggregateID,
            "aggregate_type": event.AggregateType,
            "event_type":    event.Type,
            "event_version": fmt.Sprintf("%d", event.Version),
        },
    }
    
    return es.publisher.Publish(ctx, msg)
}

// LoadAggregate loads an aggregate from events
func (es *EventStore) LoadAggregate(ctx context.Context, aggregateID string, aggregate Aggregate) error {
    // Try to load from snapshot
    snapshot, err := es.snapshots.Load(ctx, aggregateID)
    if err == nil {
        aggregate.ApplySnapshot(snapshot)
    }
    
    // Load events after snapshot
    events, err := es.storage.LoadEvents(ctx, aggregateID, snapshot.Version)
    if err != nil {
        return err
    }
    
    // Apply events
    for _, event := range events {
        if err := aggregate.ApplyEvent(event); err != nil {
            return err
        }
    }
    
    // Create snapshot if needed
    if len(events) > 100 {
        snapshot := aggregate.CreateSnapshot()
        es.snapshots.Store(ctx, aggregateID, snapshot)
    }
    
    return nil
}
```

## Summary

This API specification provides a comprehensive interface design for the SWIT messaging system. The interfaces follow Go best practices and integrate seamlessly with the existing SWIT framework patterns. Key features include:

1. **Clean Abstractions**: Well-defined interfaces for brokers, publishers, and subscribers
2. **Flexible Configuration**: Comprehensive configuration options with sensible defaults
3. **Robust Error Handling**: Typed errors with retry logic and error classification
4. **Middleware Support**: Extensible middleware chain for cross-cutting concerns
5. **Production Patterns**: Support for common patterns like request-response, event sourcing, and saga
6. **Framework Integration**: Seamless integration with SWIT's service registration and lifecycle management

The API design prioritizes developer experience while maintaining the flexibility needed for production deployments across different message broker implementations.
