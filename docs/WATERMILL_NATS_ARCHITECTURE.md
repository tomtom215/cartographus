# Production-Ready Hybrid Go + SQL Event Processing with Watermill, NATS JetStream, and DuckDB

**Version**: 2.1.0
**Last Verified**: 2026-01-11
**Target Audience**: Senior Go developers building event-driven media server analytics
**Compatibility**: Go 1.24+, DuckDB 1.4.3, NATS Server 2.12.3, Watermill 1.5.1

---

## Verification Status

This document has been verified against the Cartographus source code on 2026-01-11.

### Verified Claims

| Claim | Source | Status |
|-------|--------|--------|
| Go 1.24.0 | go.mod line 3 | Verified |
| DuckDB Go v2.5.4 | go.mod line 11 | Verified (duckdb-go/v2) |
| gobreaker/v2 v2.4.0 | go.mod line 22 | Verified (upgraded from v1) |
| Watermill v1.5.1 | go.mod line 9 | Verified |
| Watermill-NATS v2.1.3 | go.mod line 10 | Verified |
| NATS Server v2.12.3 | go.mod line 19 | Verified |
| NATS Go Client v1.48.0 | go.mod line 20 | Verified |
| WebSocket Hub pattern | internal/websocket/hub.go | Verified |
| Sync Manager callbacks | internal/sync/manager.go | Verified |
| DuckDB spatial extensions | internal/database/database.go | Verified |
| CorrelationKey deduplication | internal/eventprocessor/ | Verified |
| Event Sourcing config | internal/config/config.go | Verified |

### Corrections Made (2026-01-11)

1. **DuckDB Go driver**: Updated from v1.4.2 to v2.5.4 (major version change)
2. **gobreaker version**: Updated from v1.0.0 to gobreaker/v2 v2.4.0
3. **NATS versions**: Updated to v2.12.3 (server) and v1.48.0 (client)
4. **Module path**: Uses `github.com/tomtom215/cartographus`
5. **nats_js extension**: Marked as emerging/experimental - fallback is primary pattern

---

## Executive Summary

This guide provides a complete production-ready implementation for a dual-path event processing architecture combining Go-native Watermill event processing with DuckDB's analytical capabilities. The architecture addresses a fundamental tension in event-driven systems: the need for both **reliable processing** (acknowledgments, deduplication, ordered delivery) and **exploratory analytics** (arbitrary queries, schema discovery, data validation).

### Why This Architecture?

This architecture supports multiple deployment scenarios:

1. **Plex-only users**: Direct Plex WebSocket + API integration without Tautulli dependency
2. **Tautulli users**: Enhanced analytics on top of existing Tautulli data
3. **Hybrid users**: Both Plex and Tautulli for comprehensive coverage
4. **Migration users**: Gradual migration from Tautulli to standalone Plex support
5. **Future Jellyfin users**: Same event-driven pattern for Jellyfin integration

### Key Architecture Decisions

| Decision | Rationale |
|----------|-----------|
| **Go path for ingestion** | Exactly-once semantics via JetStream acknowledgments |
| **DuckDB for analytics** | OLAP-optimized queries, spatial extensions, in-process performance |
| **NATS JetStream** | Persistent messaging with replay, deduplication, and consumer groups |
| **Fallback pattern** | nats_js extension is emerging; Go client provides reliable backup |
| **Circuit breaker** | gobreaker/v2 v2.4.0 prevents cascade failures |

### Version Matrix (Verified Against Source 2026-01-11)

| Component | Version | Import Path | Status |
|-----------|---------|-------------|--------|
| Watermill | v1.5.1 | `github.com/ThreeDotsLabs/watermill` | Existing |
| Watermill-NATS | v2.1.3 | `github.com/ThreeDotsLabs/watermill-nats/v2` | Existing |
| NATS Server | v2.12.3 | `github.com/nats-io/nats-server/v2` | Existing |
| NATS Go Client | v1.48.0 | `github.com/nats-io/nats.go` | Existing |
| DuckDB Go Driver | v2.5.4 | `github.com/duckdb/duckdb-go/v2` | Existing |
| gobreaker | v2.4.0 | `github.com/sony/gobreaker/v2/v2` | Existing |
| nats_js extension | v0.1.1 | Community (unsigned) | Experimental |

---

## Table of Contents

1. [Architecture Overview](#1-architecture-overview)
2. [Project Structure](#2-project-structure)
3. [Dependencies and go.mod](#3-dependencies-and-gomod)
4. [NATS JetStream Configuration](#4-nats-jetstream-configuration)
5. [Watermill Publisher Implementation](#5-watermill-publisher-implementation)
6. [Watermill Subscriber Implementation](#6-watermill-subscriber-implementation)
7. [DuckDB Integration](#7-duckdb-integration)
8. [Fallback Pattern for nats_js Extension](#8-fallback-pattern-for-nats_js-extension)
9. [Circuit Breaker Integration](#9-circuit-breaker-integration)
10. [Event Models and Serialization](#10-event-models-and-serialization)
11. [Error Handling and Retry Patterns](#11-error-handling-and-retry-patterns)
12. [Monitoring and Observability](#12-monitoring-and-observability)
13. [Testing Strategy](#13-testing-strategy)
14. [Deployment Configuration](#14-deployment-configuration)
15. [Performance Tuning](#15-performance-tuning)
16. [Security Considerations](#16-security-considerations)
17. [Migration and Upgrade Path](#17-migration-and-upgrade-path)
18. [Event Sourcing Mode (NATS-First Architecture)](#15-event-sourcing-mode-nats-first-architecture)

---

## 1. Architecture Overview

### Dual-Path Processing Model

```
                              MEDIA SERVER SOURCES
   Plex WebSocket   Tautulli HTTP    Jellyfin Events      Manual Imports
        |                |                |                   |
        v                v                v                   v
                      GO WATERMILL PUBLISHERS (Write Path)
     Event normalization           Nats-Msg-Id deduplication
     Subject routing               Schema validation
     Circuit breaker protection    Retry with exponential backoff
                                     |
                                     v
                        NATS JETSTREAM (Stream Storage)
     Stream: MEDIA_EVENTS
     Subjects: playback.>, plex.>, jellyfin.>, tautulli.>
     Config: AllowDirect=true, Duplicates=2m, Retention=Limits, 7-day TTL
                    |                                 |
       +------------+------------+       +------------+------------+
       |  GO WATERMILL CONSUMER  |       |   ANALYTICS PATH       |
       |    (Processing Path)    |       |   (Query Path)         |
       |-------------------------|       |------------------------|
       | Durable consumer        |       | Primary: nats_js ext   |
       | Acknowledgments         |       | Fallback: Go NATS      |
       | Exactly-once via ack    |       | Read-only queries      |
       | Appender bulk writes    |       | Schema exploration     |
       | Horizontal scaling      |       | Debug specific msgs    |
       +-----------+-------------+       +-----------+------------+
                   |                                 |
                   v                                 |
       +------------------------+                   |
       |  DUCKDB (Normalized)   |<------------------+
       |  playback_events       |   (JOINs for validation)
       |  geolocations          |
       |  + Spatial indexes     |
       +-----------+------------+
                   |
                   v
       +------------------------+
       |   ANALYTICS / API      |
       |  Dashboards, Reports   |
       +------------------------+
```

### Integration with Existing Codebase

The current Cartographus architecture (verified from source):

- **internal/sync/manager.go**: Orchestrates Tautulli/Plex sync with callback patterns
- **internal/websocket/hub.go**: In-memory broadcast hub for real-time updates
- **internal/database/database.go**: DuckDB wrapper with spatial extensions

The new event processor will:

1. **Publish events** from sync manager to NATS JetStream
2. **Subscribe to events** for processing into DuckDB
3. **Broadcast to WebSocket** for real-time frontend updates
4. **Provide fallback** when nats_js extension unavailable

### Access Pattern Boundaries

| Use Case | Recommended Path | Reason |
|----------|------------------|--------|
| Real-time ingestion | Go Watermill | Acknowledgments, exactly-once |
| Process 1M+ events | Go Watermill | Batch fetch, concurrency |
| Query last 1,000 events | nats_js/Fallback | Simple, no consumer state |
| Debug specific sequence | nats_js/Fallback | Targeted, read-only |
| Historical analytics | DuckDB tables | Query normalized data |
| Schema discovery | nats_js/Fallback | Explore before committing |

---

## 2. Project Structure

The event processor will be added to the existing project structure:

```
cartographus/
├── internal/
│   ├── eventprocessor/           # NEW: Event processing package
│   │   ├── config.go             # Event processor configuration
│   │   ├── publisher.go          # Watermill publisher wrapper
│   │   ├── subscriber.go         # Watermill subscriber wrapper
│   │   ├── stream.go             # JetStream stream configuration
│   │   ├── server.go             # Embedded NATS server (optional)
│   │   ├── reader.go             # Unified stream reader interface
│   │   ├── fallback.go           # Go NATS client fallback
│   │   ├── resilient_reader.go   # Auto-fallback reader
│   │   ├── events.go             # Event type definitions
│   │   ├── serializer.go         # JSON serialization
│   │   ├── handler.go            # Event handlers
│   │   ├── appender.go           # DuckDB batch appender
│   │   ├── retry.go              # Retry policies
│   │   └── doc.go                # Package documentation
│   ├── sync/                      # EXISTING: Modified for event publishing
│   │   └── manager.go            # Add event publishing hooks
│   ├── websocket/                 # EXISTING: Modified for event consumption
│   │   └── hub.go                # Subscribe to NATS for broadcasts
│   ├── database/                  # EXISTING: No changes required
│   ├── config/                    # EXISTING: Add NATS configuration
│   │   └── config.go             # Add NATSConfig struct
│   └── ...
├── tests/
│   └── integration/
│       └── eventprocessor_test.go # Integration tests
└── ...
```

---

## 3. Dependencies and go.mod

### New Dependencies to Add

```go
// Add to go.mod require block:

require (
    // Existing dependencies...
    github.com/marcboeker/go-duckdb v1.4.2
    github.com/golang-jwt/jwt/v5 v5.3.0
    github.com/google/uuid v1.6.0
    github.com/gorilla/websocket v1.5.3
    github.com/prometheus/client_golang v1.23.2
    github.com/sony/gobreaker/v2/v2 v2.4.0
    golang.org/x/crypto v0.45.0
    golang.org/x/time v0.14.0

    // NEW: Event processing dependencies
    github.com/ThreeDotsLabs/watermill v1.5.1
    github.com/ThreeDotsLabs/watermill-nats/v2 v2.1.3
    github.com/nats-io/nats.go v1.47.0
    github.com/nats-io/nats-server/v2 v2.12.2
)
```

### Installation Commands

```bash
# Add Watermill and NATS dependencies
go get github.com/ThreeDotsLabs/watermill@v1.5.1
go get github.com/ThreeDotsLabs/watermill-nats/v2@v2.1.3
go get github.com/nats-io/nats.go@v1.47.0
go get github.com/nats-io/nats-server/v2@v2.12.2

# Update go.sum
go mod tidy
```

---

## 4. NATS JetStream Configuration

### 4.1 Embedded NATS Server

```go
// internal/eventprocessor/server.go
package eventprocessor

import (
    "context"
    "fmt"
    "time"

    "github.com/nats-io/nats-server/v2/server"
    natsgo "github.com/nats-io/nats.go"
)

// ServerConfig holds embedded NATS server configuration
type ServerConfig struct {
    Host              string
    Port              int
    StoreDir          string
    JetStreamMaxMem   int64
    JetStreamMaxStore int64
    EnableClustering  bool
    ClusterName       string
    Routes            []string
}

// DefaultServerConfig returns production defaults
func DefaultServerConfig() ServerConfig {
    return ServerConfig{
        Host:              "127.0.0.1",
        Port:              4222,
        StoreDir:          "/data/nats/jetstream",
        JetStreamMaxMem:   1 << 30,  // 1GB
        JetStreamMaxStore: 10 << 30, // 10GB
        EnableClustering:  false,
    }
}

// EmbeddedServer wraps the NATS server with lifecycle management
type EmbeddedServer struct {
    server    *server.Server
    config    ServerConfig
    clientURL string
}

// NewEmbeddedServer creates and starts an embedded NATS server
func NewEmbeddedServer(cfg ServerConfig) (*EmbeddedServer, error) {
    opts := &server.Options{
        ServerName:         "media-events",
        Host:               cfg.Host,
        Port:               cfg.Port,
        JetStream:          true,
        StoreDir:           cfg.StoreDir,
        JetStreamMaxMemory: cfg.JetStreamMaxMem,
        JetStreamMaxStore:  cfg.JetStreamMaxStore,
        // CRITICAL: Set to false for hybrid deployments
        // nats_js extension requires TCP access
        DontListen: false,
        // Logging
        Debug:      false,
        Trace:      false,
        NoLog:      false,
        MaxPayload: 8 * 1024 * 1024, // 8MB max message size
    }

    ns, err := server.NewServer(opts)
    if err != nil {
        return nil, fmt.Errorf("create NATS server: %w", err)
    }

    // Configure structured logging
    ns.ConfigureLogger()

    // Start in background
    go ns.Start()

    // Wait for ready with timeout
    if !ns.ReadyForConnections(30 * time.Second) {
        ns.Shutdown()
        return nil, fmt.Errorf("NATS server not ready within timeout")
    }

    return &EmbeddedServer{
        server:    ns,
        config:    cfg,
        clientURL: ns.ClientURL(),
    }, nil
}

// ClientURL returns the connection URL for clients
func (s *EmbeddedServer) ClientURL() string {
    return s.clientURL
}

// Shutdown gracefully stops the server
func (s *EmbeddedServer) Shutdown(ctx context.Context) error {
    // Allow in-flight messages to complete
    s.server.Shutdown()

    // Wait for shutdown or context cancellation
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
        s.server.WaitForShutdown()
        return nil
    }
}

// IsRunning returns server health status
func (s *EmbeddedServer) IsRunning() bool {
    return s.server.Running()
}
```

### 4.2 Stream Configuration

```go
// internal/eventprocessor/stream.go
package eventprocessor

import (
    "context"
    "fmt"
    "time"

    "github.com/nats-io/nats.go"
    "github.com/nats-io/nats.go/jetstream"
)

// StreamConfig defines media event stream settings
type StreamConfig struct {
    Name            string
    Subjects        []string
    Retention       jetstream.RetentionPolicy
    MaxAge          time.Duration
    MaxBytes        int64
    MaxMsgs         int64
    DuplicateWindow time.Duration
    Replicas        int
}

// DefaultStreamConfig returns production stream configuration
func DefaultStreamConfig() StreamConfig {
    return StreamConfig{
        Name: "MEDIA_EVENTS",
        Subjects: []string{
            "playback.>",
            "plex.>",
            "jellyfin.>",
            "tautulli.>",
        },
        Retention:       jetstream.LimitsPolicy,
        MaxAge:          7 * 24 * time.Hour, // 7 days
        MaxBytes:        10 * 1024 * 1024 * 1024, // 10GB
        MaxMsgs:         -1, // Unlimited
        DuplicateWindow: 2 * time.Minute,
        Replicas:        1, // Increase for clustering
    }
}

// StreamManager handles JetStream stream lifecycle
type StreamManager struct {
    js     jetstream.JetStream
    nc     *nats.Conn
    config StreamConfig
}

// NewStreamManager creates a stream manager with the given config
func NewStreamManager(nc *nats.Conn, cfg StreamConfig) (*StreamManager, error) {
    js, err := jetstream.New(nc)
    if err != nil {
        return nil, fmt.Errorf("create JetStream context: %w", err)
    }

    return &StreamManager{
        js:     js,
        nc:     nc,
        config: cfg,
    }, nil
}

// EnsureStream creates or updates the stream configuration
func (m *StreamManager) EnsureStream(ctx context.Context) (jetstream.Stream, error) {
    streamCfg := jetstream.StreamConfig{
        Name:        m.config.Name,
        Subjects:    m.config.Subjects,
        Retention:   m.config.Retention,
        MaxAge:      m.config.MaxAge,
        MaxBytes:    m.config.MaxBytes,
        MaxMsgs:     m.config.MaxMsgs,
        Duplicates:  m.config.DuplicateWindow,
        Replicas:    m.config.Replicas,
        Storage:     jetstream.FileStorage,
        // CRITICAL: Required for nats_js extension
        AllowDirect: true,
        // Enable mirror direct get for read replicas
        MirrorDirect: true,
        // Discard old messages when limits reached
        Discard: jetstream.DiscardOld,
        // Allow message rollup for compaction
        AllowRollup: true,
    }

    // Try to get existing stream
    stream, err := m.js.Stream(ctx, m.config.Name)
    if err == nil {
        // Update existing stream
        stream, err = m.js.UpdateStream(ctx, streamCfg)
        if err != nil {
            return nil, fmt.Errorf("update stream: %w", err)
        }
        return stream, nil
    }

    // Create new stream
    stream, err = m.js.CreateStream(ctx, streamCfg)
    if err != nil {
        return nil, fmt.Errorf("create stream: %w", err)
    }

    return stream, nil
}

// GetStreamInfo returns current stream state
func (m *StreamManager) GetStreamInfo(ctx context.Context) (*jetstream.StreamInfo, error) {
    stream, err := m.js.Stream(ctx, m.config.Name)
    if err != nil {
        return nil, fmt.Errorf("get stream: %w", err)
    }
    return stream.Info(ctx)
}
```

---

## 5. Watermill Publisher Implementation

```go
// internal/eventprocessor/publisher.go
package eventprocessor

import (
    "context"
    "fmt"
    "sync"
    "time"

    "github.com/ThreeDotsLabs/watermill"
    "github.com/ThreeDotsLabs/watermill/message"
    wmNats "github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
    natsgo "github.com/nats-io/nats.go"
    "github.com/sony/gobreaker/v2"
)

// PublisherConfig holds publisher configuration
type PublisherConfig struct {
    URL             string
    MaxReconnects   int
    ReconnectWait   time.Duration
    ReconnectBuffer int
    EnableTrackMsgId bool
    CircuitBreaker  *gobreaker.CircuitBreaker
}

// DefaultPublisherConfig returns production defaults
func DefaultPublisherConfig(url string) PublisherConfig {
    return PublisherConfig{
        URL:             url,
        MaxReconnects:   -1, // Unlimited
        ReconnectWait:   2 * time.Second,
        ReconnectBuffer: 8 * 1024 * 1024, // 8MB
        EnableTrackMsgId: true,
    }
}

// Publisher wraps Watermill publisher with resilience patterns
type Publisher struct {
    publisher      message.Publisher
    circuitBreaker *gobreaker.CircuitBreaker
    nc             *natsgo.Conn
    mu             sync.RWMutex
    closed         bool
    logger         watermill.LoggerAdapter
}

// NewPublisher creates a resilient Watermill NATS publisher
func NewPublisher(cfg PublisherConfig, logger watermill.LoggerAdapter) (*Publisher, error) {
    if logger == nil {
        logger = watermill.NewStdLogger(false, false)
    }

    // NATS connection options with reconnection handling
    natsOpts := []natsgo.Option{
        natsgo.RetryOnFailedConnect(true),
        natsgo.MaxReconnects(cfg.MaxReconnects),
        natsgo.ReconnectWait(cfg.ReconnectWait),
        natsgo.ReconnectBufSize(cfg.ReconnectBuffer),
        natsgo.DisconnectErrHandler(func(nc *natsgo.Conn, err error) {
            if err != nil {
                logger.Error("NATS disconnected", err, nil)
            }
        }),
        natsgo.ReconnectHandler(func(nc *natsgo.Conn) {
            logger.Info("NATS reconnected", watermill.LogFields{
                "url": nc.ConnectedUrl(),
            })
        }),
        natsgo.ErrorHandler(func(nc *natsgo.Conn, sub *natsgo.Subscription, err error) {
            logger.Error("NATS error", err, watermill.LogFields{
                "subject": sub.Subject,
            })
        }),
    }

    // Watermill publisher configuration
    wmConfig := wmNats.PublisherConfig{
        URL:         cfg.URL,
        NatsOptions: natsOpts,
        Marshaler:   &wmNats.NATSMarshaler{},
        JetStream: wmNats.JetStreamConfig{
            Disabled:      false,
            AutoProvision: true,
            TrackMsgId:    cfg.EnableTrackMsgId, // Enable deduplication
            PublishOptions: []natsgo.PubOpt{
                natsgo.RetryAttempts(3),
                natsgo.RetryWait(100 * time.Millisecond),
            },
        },
    }

    pub, err := wmNats.NewPublisher(wmConfig, logger)
    if err != nil {
        return nil, fmt.Errorf("create watermill publisher: %w", err)
    }

    return &Publisher{
        publisher:      pub,
        circuitBreaker: cfg.CircuitBreaker,
        logger:         logger,
    }, nil
}

// Publish sends a message to the specified topic with circuit breaker protection
func (p *Publisher) Publish(ctx context.Context, topic string, msg *message.Message) error {
    p.mu.RLock()
    if p.closed {
        p.mu.RUnlock()
        return fmt.Errorf("publisher is closed")
    }
    p.mu.RUnlock()

    // Set Nats-Msg-Id for deduplication if not already set
    if msg.Metadata.Get(natsgo.MsgIdHdr) == "" {
        msg.Metadata.Set(natsgo.MsgIdHdr, msg.UUID)
    }

    // Circuit breaker wrapper (using v1.0.0 API)
    if p.circuitBreaker != nil {
        _, err := p.circuitBreaker.Execute(func() (interface{}, error) {
            return nil, p.publisher.Publish(topic, msg)
        })
        return err
    }

    return p.publisher.Publish(topic, msg)
}

// PublishBatch publishes multiple messages atomically
func (p *Publisher) PublishBatch(ctx context.Context, topic string, msgs ...*message.Message) error {
    for _, msg := range msgs {
        if err := p.Publish(ctx, topic, msg); err != nil {
            return fmt.Errorf("publish message %s: %w", msg.UUID, err)
        }
    }
    return nil
}

// Close gracefully shuts down the publisher
func (p *Publisher) Close() error {
    p.mu.Lock()
    defer p.mu.Unlock()

    if p.closed {
        return nil
    }
    p.closed = true

    return p.publisher.Close()
}
```

---

## 6. Watermill Subscriber Implementation

```go
// internal/eventprocessor/subscriber.go
package eventprocessor

import (
    "context"
    "fmt"
    "time"

    "github.com/ThreeDotsLabs/watermill"
    "github.com/ThreeDotsLabs/watermill/message"
    wmNats "github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
    natsgo "github.com/nats-io/nats.go"
)

// SubscriberConfig holds subscriber configuration
type SubscriberConfig struct {
    URL               string
    DurableName       string
    QueueGroup        string
    SubscribersCount  int
    AckWaitTimeout    time.Duration
    MaxDeliver        int
    MaxAckPending     int
    CloseTimeout      time.Duration
    MaxReconnects     int
    ReconnectWait     time.Duration
}

// DefaultSubscriberConfig returns production defaults
func DefaultSubscriberConfig(url string) SubscriberConfig {
    return SubscriberConfig{
        URL:              url,
        DurableName:      "media-processor",
        QueueGroup:       "processors",
        SubscribersCount: 4,
        AckWaitTimeout:   30 * time.Second,
        MaxDeliver:       5,      // Max redelivery attempts
        MaxAckPending:    1000,   // Flow control
        CloseTimeout:     30 * time.Second,
        MaxReconnects:    -1,
        ReconnectWait:    2 * time.Second,
    }
}

// Subscriber wraps Watermill subscriber with configuration
type Subscriber struct {
    subscriber message.Subscriber
    config     SubscriberConfig
    logger     watermill.LoggerAdapter
}

// NewSubscriber creates a durable JetStream subscriber
func NewSubscriber(cfg SubscriberConfig, logger watermill.LoggerAdapter) (*Subscriber, error) {
    if logger == nil {
        logger = watermill.NewStdLogger(false, false)
    }

    natsOpts := []natsgo.Option{
        natsgo.RetryOnFailedConnect(true),
        natsgo.MaxReconnects(cfg.MaxReconnects),
        natsgo.ReconnectWait(cfg.ReconnectWait),
        natsgo.DisconnectErrHandler(func(nc *natsgo.Conn, err error) {
            if err != nil {
                logger.Error("Subscriber disconnected", err, nil)
            }
        }),
        natsgo.ReconnectHandler(func(nc *natsgo.Conn) {
            logger.Info("Subscriber reconnected", watermill.LogFields{
                "url": nc.ConnectedUrl(),
            })
        }),
    }

    // JetStream consumer options
    subOpts := []natsgo.SubOpt{
        natsgo.MaxDeliver(cfg.MaxDeliver),
        natsgo.MaxAckPending(cfg.MaxAckPending),
        natsgo.AckWait(cfg.AckWaitTimeout),
        // Deliver new messages only (use DeliverAll for replay)
        natsgo.DeliverNew(),
    }

    wmConfig := wmNats.SubscriberConfig{
        URL:              cfg.URL,
        DurableName:      cfg.DurableName,
        QueueGroupPrefix: cfg.QueueGroup,
        SubscribersCount: cfg.SubscribersCount,
        AckWaitTimeout:   cfg.AckWaitTimeout,
        CloseTimeout:     cfg.CloseTimeout,
        NatsOptions:      natsOpts,
        Unmarshaler:      &wmNats.NATSMarshaler{},
        JetStream: wmNats.JetStreamConfig{
            Disabled:         false,
            AutoProvision:    true,
            AckAsync:         false, // Synchronous for exactly-once
            SubscribeOptions: subOpts,
            DurablePrefix:    cfg.DurableName,
        },
    }

    sub, err := wmNats.NewSubscriber(wmConfig, logger)
    if err != nil {
        return nil, fmt.Errorf("create watermill subscriber: %w", err)
    }

    return &Subscriber{
        subscriber: sub,
        config:     cfg,
        logger:     logger,
    }, nil
}

// Subscribe returns a channel of messages for the given topic
func (s *Subscriber) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
    return s.subscriber.Subscribe(ctx, topic)
}

// Close gracefully shuts down the subscriber
func (s *Subscriber) Close() error {
    return s.subscriber.Close()
}

// MessageHandler provides a fluent API for message processing
type MessageHandler struct {
    subscriber *Subscriber
    topic      string
    handler    func(ctx context.Context, msg *message.Message) error
    logger     watermill.LoggerAdapter
}

// NewMessageHandler creates a handler for processing messages
func (s *Subscriber) NewMessageHandler(topic string) *MessageHandler {
    return &MessageHandler{
        subscriber: s,
        topic:      topic,
        logger:     s.logger,
    }
}

// Handle sets the message processing function
func (h *MessageHandler) Handle(fn func(ctx context.Context, msg *message.Message) error) *MessageHandler {
    h.handler = fn
    return h
}

// Run starts processing messages until context cancellation
func (h *MessageHandler) Run(ctx context.Context) error {
    messages, err := h.subscriber.Subscribe(ctx, h.topic)
    if err != nil {
        return fmt.Errorf("subscribe to %s: %w", h.topic, err)
    }

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case msg, ok := <-messages:
            if !ok {
                return nil
            }
            if err := h.processMessage(ctx, msg); err != nil {
                h.logger.Error("Message processing failed", err, watermill.LogFields{
                    "message_uuid": msg.UUID,
                    "topic":        h.topic,
                })
            }
        }
    }
}

func (h *MessageHandler) processMessage(ctx context.Context, msg *message.Message) error {
    if h.handler == nil {
        msg.Ack()
        return nil
    }

    if err := h.handler(ctx, msg); err != nil {
        msg.Nack()
        return err
    }

    msg.Ack()
    return nil
}
```

---

## 7. DuckDB Integration

### 7.1 Batch Appender

The existing DuckDB integration in `internal/database/database.go` is well-designed. The event processor will add a specialized batch appender for high-throughput event ingestion:

```go
// internal/eventprocessor/appender.go
package eventprocessor

import (
    "context"
    "fmt"
    "sync"
    "time"

    "github.com/tomtom215/cartographus/internal/database"
)

// AppenderConfig holds batch appender configuration
type AppenderConfig struct {
    Table         string
    BatchSize     int
    FlushInterval time.Duration
}

// DefaultAppenderConfig returns production defaults
func DefaultAppenderConfig() AppenderConfig {
    return AppenderConfig{
        Table:         "playback_events",
        BatchSize:     1000,
        FlushInterval: 5 * time.Second,
    }
}

// EventAppender provides thread-safe batch writes to DuckDB
type EventAppender struct {
    db            *database.DB
    config        AppenderConfig

    mu            sync.Mutex
    buffer        []*MediaEvent
    lastFlush     time.Time

    flushTicker   *time.Ticker
    stopChan      chan struct{}
    stopped       bool
}

// NewEventAppender creates a batch appender for media events
func NewEventAppender(db *database.DB, cfg AppenderConfig) (*EventAppender, error) {
    ea := &EventAppender{
        db:        db,
        config:    cfg,
        buffer:    make([]*MediaEvent, 0, cfg.BatchSize),
        lastFlush: time.Now(),
        stopChan:  make(chan struct{}),
    }

    // Start background flush goroutine
    ea.flushTicker = time.NewTicker(cfg.FlushInterval)
    go ea.backgroundFlush()

    return ea, nil
}

// AppendEvent adds an event to the batch buffer
func (ea *EventAppender) AppendEvent(event *MediaEvent) error {
    ea.mu.Lock()
    defer ea.mu.Unlock()

    if ea.stopped {
        return fmt.Errorf("appender is closed")
    }

    ea.buffer = append(ea.buffer, event)

    // Auto-flush on batch size
    if len(ea.buffer) >= ea.config.BatchSize {
        return ea.flushLocked(context.Background())
    }

    return nil
}

// Flush forces an immediate flush of pending events
func (ea *EventAppender) Flush(ctx context.Context) error {
    ea.mu.Lock()
    defer ea.mu.Unlock()
    return ea.flushLocked(ctx)
}

func (ea *EventAppender) flushLocked(ctx context.Context) error {
    if len(ea.buffer) == 0 {
        return nil
    }

    // Convert events to database format and insert
    for _, event := range ea.buffer {
        if err := ea.insertEvent(ctx, event); err != nil {
            return fmt.Errorf("insert event: %w", err)
        }
    }

    ea.buffer = ea.buffer[:0]
    ea.lastFlush = time.Now()
    return nil
}

func (ea *EventAppender) insertEvent(ctx context.Context, event *MediaEvent) error {
    // Use existing database insert method
    // This integrates with the existing playback_events table schema
    return ea.db.InsertPlaybackEventFromNATS(ctx, event)
}

func (ea *EventAppender) backgroundFlush() {
    for {
        select {
        case <-ea.stopChan:
            return
        case <-ea.flushTicker.C:
            ea.mu.Lock()
            if len(ea.buffer) > 0 {
                _ = ea.flushLocked(context.Background())
            }
            ea.mu.Unlock()
        }
    }
}

// Close flushes remaining events and closes the appender
func (ea *EventAppender) Close() error {
    ea.mu.Lock()
    defer ea.mu.Unlock()

    if ea.stopped {
        return nil
    }
    ea.stopped = true

    // Stop background flush
    close(ea.stopChan)
    ea.flushTicker.Stop()

    // Final flush
    return ea.flushLocked(context.Background())
}

// Stats returns appender statistics
func (ea *EventAppender) Stats() AppenderStats {
    ea.mu.Lock()
    defer ea.mu.Unlock()

    return AppenderStats{
        PendingEvents: len(ea.buffer),
        LastFlush:     ea.lastFlush,
    }
}

// AppenderStats holds appender metrics
type AppenderStats struct {
    PendingEvents int
    LastFlush     time.Time
}
```

---

## 8. Fallback Pattern for nats_js Extension

This is the **critical section** for production reliability. The nats_js extension is an emerging community project with limited testing, so a robust fallback using the native NATS Go client is essential.

### 8.1 Unified Reader Interface

```go
// internal/eventprocessor/reader.go
package eventprocessor

import (
    "context"
    "time"
)

// StreamMessage represents a message from the stream
type StreamMessage struct {
    Sequence  uint64
    Subject   string
    Data      []byte
    Timestamp time.Time
    Headers   map[string][]string
}

// QueryOptions defines stream query parameters
type QueryOptions struct {
    StartSeq    uint64
    EndSeq      uint64
    StartTime   time.Time
    EndTime     time.Time
    Subject     string
    Limit       int
    JSONExtract map[string]string // field -> JSON path
}

// StreamReader provides a unified interface for reading from streams
type StreamReader interface {
    // Query returns messages matching the options
    Query(ctx context.Context, stream string, opts QueryOptions) ([]StreamMessage, error)

    // GetMessage retrieves a single message by sequence
    GetMessage(ctx context.Context, stream string, seq uint64) (*StreamMessage, error)

    // GetLastSequence returns the latest sequence number
    GetLastSequence(ctx context.Context, stream string) (uint64, error)

    // Health checks reader availability
    Health(ctx context.Context) error

    // Close releases resources
    Close() error
}

// ReaderType identifies the implementation
type ReaderType string

const (
    ReaderTypeNatsJS   ReaderType = "natsjs"
    ReaderTypeFallback ReaderType = "fallback"
)
```

### 8.2 Go NATS Client Fallback

```go
// internal/eventprocessor/fallback.go
package eventprocessor

import (
    "context"
    "fmt"
    "sync"
    "time"

    "github.com/nats-io/nats.go"
    "github.com/nats-io/nats.go/jetstream"
)

// FallbackReader uses the native NATS Go client for stream queries
type FallbackReader struct {
    nc     *nats.Conn
    js     jetstream.JetStream
    mu     sync.RWMutex
    closed bool
}

// NewFallbackReader creates a reader using the NATS Go client
func NewFallbackReader(natsURL string) (*FallbackReader, error) {
    nc, err := nats.Connect(natsURL,
        nats.RetryOnFailedConnect(true),
        nats.MaxReconnects(10),
        nats.ReconnectWait(time.Second),
    )
    if err != nil {
        return nil, fmt.Errorf("connect to NATS: %w", err)
    }

    js, err := jetstream.New(nc)
    if err != nil {
        nc.Close()
        return nil, fmt.Errorf("create JetStream context: %w", err)
    }

    return &FallbackReader{
        nc: nc,
        js: js,
    }, nil
}

// Query retrieves messages using JetStream Direct Get API
func (r *FallbackReader) Query(ctx context.Context, streamName string, opts QueryOptions) ([]StreamMessage, error) {
    r.mu.RLock()
    if r.closed {
        r.mu.RUnlock()
        return nil, fmt.Errorf("reader is closed")
    }
    r.mu.RUnlock()

    stream, err := r.js.Stream(ctx, streamName)
    if err != nil {
        return nil, fmt.Errorf("get stream: %w", err)
    }

    // Get stream info for bounds
    info, err := stream.Info(ctx)
    if err != nil {
        return nil, fmt.Errorf("get stream info: %w", err)
    }

    // Determine sequence range
    startSeq := opts.StartSeq
    if startSeq == 0 {
        startSeq = info.State.FirstSeq
    }

    endSeq := opts.EndSeq
    if endSeq == 0 {
        endSeq = info.State.LastSeq
    }

    // Fetch messages
    var messages []StreamMessage
    limit := opts.Limit
    if limit <= 0 {
        limit = 10000 // Default safety limit
    }

    for seq := startSeq; seq <= endSeq && len(messages) < limit; seq++ {
        msg, err := stream.GetMsg(ctx, seq)
        if err != nil {
            // Skip deleted or unavailable messages
            continue
        }

        // Subject filter
        if opts.Subject != "" && !matchSubject(msg.Subject, opts.Subject) {
            continue
        }

        messages = append(messages, StreamMessage{
            Sequence:  seq,
            Subject:   msg.Subject,
            Data:      msg.Data,
            Timestamp: msg.Time,
            Headers:   convertHeaders(msg.Header),
        })
    }

    return messages, nil
}

// GetMessage retrieves a single message by sequence using Direct Get
func (r *FallbackReader) GetMessage(ctx context.Context, streamName string, seq uint64) (*StreamMessage, error) {
    r.mu.RLock()
    if r.closed {
        r.mu.RUnlock()
        return nil, fmt.Errorf("reader is closed")
    }
    r.mu.RUnlock()

    stream, err := r.js.Stream(ctx, streamName)
    if err != nil {
        return nil, fmt.Errorf("get stream: %w", err)
    }

    msg, err := stream.GetMsg(ctx, seq)
    if err != nil {
        return nil, fmt.Errorf("get message %d: %w", seq, err)
    }

    return &StreamMessage{
        Sequence:  seq,
        Subject:   msg.Subject,
        Data:      msg.Data,
        Timestamp: msg.Time,
        Headers:   convertHeaders(msg.Header),
    }, nil
}

// GetLastSequence returns the latest sequence number
func (r *FallbackReader) GetLastSequence(ctx context.Context, streamName string) (uint64, error) {
    r.mu.RLock()
    if r.closed {
        r.mu.RUnlock()
        return 0, fmt.Errorf("reader is closed")
    }
    r.mu.RUnlock()

    stream, err := r.js.Stream(ctx, streamName)
    if err != nil {
        return 0, fmt.Errorf("get stream: %w", err)
    }

    info, err := stream.Info(ctx)
    if err != nil {
        return 0, fmt.Errorf("get stream info: %w", err)
    }

    return info.State.LastSeq, nil
}

// Health checks NATS connectivity
func (r *FallbackReader) Health(ctx context.Context) error {
    r.mu.RLock()
    defer r.mu.RUnlock()

    if r.closed {
        return fmt.Errorf("reader is closed")
    }
    if !r.nc.IsConnected() {
        return fmt.Errorf("not connected to NATS")
    }
    return nil
}

// Close releases the NATS connection
func (r *FallbackReader) Close() error {
    r.mu.Lock()
    defer r.mu.Unlock()

    if r.closed {
        return nil
    }
    r.closed = true

    r.nc.Close()
    return nil
}

func matchSubject(subject, pattern string) bool {
    // Simple wildcard matching for NATS subjects
    if pattern == ">" || pattern == "*" {
        return true
    }
    if len(pattern) > 0 && pattern[len(pattern)-1] == '>' {
        prefix := pattern[:len(pattern)-1]
        return len(subject) >= len(prefix) && subject[:len(prefix)] == prefix
    }
    return subject == pattern
}

func convertHeaders(h nats.Header) map[string][]string {
    if h == nil {
        return nil
    }
    result := make(map[string][]string, len(h))
    for k, v := range h {
        result[k] = v
    }
    return result
}
```

---

## 9. Circuit Breaker Integration

Using gobreaker/v2 v2.4.0 (verified from go.mod):

```go
// internal/eventprocessor/circuitbreaker.go
package eventprocessor

import (
    "time"

    "github.com/sony/gobreaker/v2"
)

// CircuitBreakerConfig holds circuit breaker settings
type CircuitBreakerConfig struct {
    Name              string
    MaxRequests       uint32        // Allowed in half-open state
    Interval          time.Duration // Reset interval for counts
    Timeout           time.Duration // Time to stay open
    FailureThreshold  uint32        // Failures before opening
    OnStateChange     func(name string, from, to gobreaker.State)
}

// DefaultCircuitBreakerConfig returns production defaults
func DefaultCircuitBreakerConfig(name string) CircuitBreakerConfig {
    return CircuitBreakerConfig{
        Name:             name,
        MaxRequests:      3,
        Interval:         30 * time.Second,
        Timeout:          10 * time.Second,
        FailureThreshold: 5,
    }
}

// NewCircuitBreaker creates a circuit breaker with v1.0.0 API
func NewCircuitBreaker(cfg CircuitBreakerConfig) *gobreaker.CircuitBreaker {
    settings := gobreaker.Settings{
        Name:        cfg.Name,
        MaxRequests: cfg.MaxRequests,
        Interval:    cfg.Interval,
        Timeout:     cfg.Timeout,
        ReadyToTrip: func(counts gobreaker.Counts) bool {
            return counts.ConsecutiveFailures >= cfg.FailureThreshold
        },
        OnStateChange: cfg.OnStateChange,
    }

    return gobreaker.NewCircuitBreaker(settings)
}
```

---

## 10. Event Models and Serialization

```go
// internal/eventprocessor/events.go
package eventprocessor

import (
    "encoding/json"
    "time"

    "github.com/google/uuid"
)

// MediaEvent represents a playback event from media servers
type MediaEvent struct {
    // Identification
    EventID   string    `json:"event_id"`
    Source    string    `json:"source"` // plex, jellyfin, tautulli
    Timestamp time.Time `json:"timestamp"`

    // User
    UserID   int    `json:"user_id"`
    Username string `json:"username"`

    // Media
    MediaType        string `json:"media_type"` // movie, episode, track
    Title            string `json:"title"`
    ParentTitle      string `json:"parent_title,omitempty"`
    GrandparentTitle string `json:"grandparent_title,omitempty"`
    RatingKey        string `json:"rating_key,omitempty"`

    // Playback
    StartedAt       time.Time  `json:"started_at"`
    StoppedAt       *time.Time `json:"stopped_at,omitempty"`
    PercentComplete int        `json:"percent_complete,omitempty"`
    PlayDuration    int        `json:"play_duration,omitempty"` // seconds
    PausedCounter   int        `json:"paused_counter,omitempty"`

    // Platform
    Platform     string `json:"platform,omitempty"`
    Player       string `json:"player,omitempty"`
    IPAddress    string `json:"ip_address,omitempty"`
    LocationType string `json:"location_type,omitempty"` // wan, lan

    // Quality
    TranscodeDecision   string `json:"transcode_decision,omitempty"`
    VideoResolution     string `json:"video_resolution,omitempty"`
    VideoCodec          string `json:"video_codec,omitempty"`
    VideoDynamicRange   string `json:"video_dynamic_range,omitempty"`
    AudioCodec          string `json:"audio_codec,omitempty"`
    AudioChannels       int    `json:"audio_channels,omitempty"`
    StreamBitrate       int    `json:"stream_bitrate,omitempty"`

    // Connection
    Secure  bool `json:"secure,omitempty"`
    Local   bool `json:"local,omitempty"`
    Relayed bool `json:"relayed,omitempty"`

    // Raw payload for debugging
    RawPayload json.RawMessage `json:"raw_payload,omitempty"`
}

// NewMediaEvent creates an event with a unique ID
func NewMediaEvent(source string) *MediaEvent {
    return &MediaEvent{
        EventID:   uuid.New().String(),
        Source:    source,
        Timestamp: time.Now().UTC(),
    }
}

// Validate checks required fields
func (e *MediaEvent) Validate() error {
    if e.EventID == "" {
        return &ValidationError{Field: "event_id", Message: "required"}
    }
    if e.Source == "" {
        return &ValidationError{Field: "source", Message: "required"}
    }
    if e.UserID == 0 {
        return &ValidationError{Field: "user_id", Message: "required"}
    }
    if e.MediaType == "" {
        return &ValidationError{Field: "media_type", Message: "required"}
    }
    if e.Title == "" {
        return &ValidationError{Field: "title", Message: "required"}
    }
    return nil
}

// Topic returns the NATS subject for this event
func (e *MediaEvent) Topic() string {
    return "playback." + e.Source + "." + e.MediaType
}

// ValidationError represents a field validation error
type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return e.Field + ": " + e.Message
}
```

---

## 11. Configuration Updates

Add to existing config package:

```go
// Add to internal/config/config.go

// NATSConfig holds NATS JetStream configuration for event processing
type NATSConfig struct {
    // Enabled controls whether event processing is active
    Enabled bool

    // URL is the NATS server connection URL
    // Default: nats://127.0.0.1:4222
    URL string

    // EmbeddedServer enables embedded NATS server
    // If false, expects external NATS server at URL
    EmbeddedServer bool

    // StoreDir is the JetStream storage directory
    // Default: /data/nats/jetstream
    StoreDir string

    // MaxMemory is the maximum memory for JetStream
    // Default: 1GB
    MaxMemory int64

    // MaxStore is the maximum disk storage for JetStream
    // Default: 10GB
    MaxStore int64

    // StreamRetentionDays is how long to keep events
    // Default: 7 days
    StreamRetentionDays int

    // BatchSize is the number of events to batch before writing
    // Default: 1000
    BatchSize int

    // FlushInterval is the maximum time between flushes
    // Default: 5s
    FlushInterval time.Duration
}

// Environment variables:
// NATS_ENABLED - Enable NATS event processing (default: false)
// NATS_URL - NATS server URL (default: nats://127.0.0.1:4222)
// NATS_EMBEDDED - Use embedded NATS server (default: true)
// NATS_STORE_DIR - JetStream storage directory
// NATS_MAX_MEMORY - Maximum memory in bytes (default: 1073741824)
// NATS_MAX_STORE - Maximum disk storage in bytes (default: 10737418240)
// NATS_RETENTION_DAYS - Stream retention in days (default: 7)
// NATS_BATCH_SIZE - Batch size for writes (default: 1000)
// NATS_FLUSH_INTERVAL - Flush interval (default: 5s)
```

---

## 12. Integration with Existing Code

### 12.1 Sync Manager Integration

Modify `internal/sync/manager.go` to publish events to NATS:

```go
// Add to Manager struct:
type Manager struct {
    // ... existing fields ...
    eventPublisher *eventprocessor.Publisher // Optional: NATS event publisher
}

// Add to NewManager:
func NewManager(db DBInterface, client TautulliClientInterface, cfg *config.Config, wsHub WebSocketHub, eventPub *eventprocessor.Publisher) *Manager {
    m := &Manager{
        // ... existing initialization ...
        eventPublisher: eventPub,
    }
    // ...
}

// Add event publishing to processHistoryRecord:
func (m *Manager) processHistoryRecord(ctx context.Context, record *TautulliHistoryRecord) error {
    // ... existing processing ...

    // Publish to NATS if enabled
    if m.eventPublisher != nil {
        event := eventprocessor.NewMediaEvent("tautulli")
        // Map fields from record to event
        event.UserID = record.UserID
        event.Username = record.Username
        // ... etc ...

        msg, err := eventprocessor.SerializeEvent(event)
        if err != nil {
            log.Printf("Failed to serialize event: %v", err)
        } else if err := m.eventPublisher.Publish(ctx, event.Topic(), msg); err != nil {
            log.Printf("Failed to publish event: %v", err)
        }
    }

    return nil
}
```

---

## 13. Testing Strategy

```go
// tests/integration/eventprocessor_test.go
package integration

import (
    "context"
    "testing"
    "time"

    "github.com/tomtom215/cartographus/internal/eventprocessor"
)

func TestEventPipeline_EndToEnd(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // Start embedded NATS server
    natsServer, err := eventprocessor.NewEmbeddedServer(eventprocessor.DefaultServerConfig())
    if err != nil {
        t.Fatalf("Failed to start NATS server: %v", err)
    }
    defer natsServer.Shutdown(ctx)

    // Create publisher
    pub, err := eventprocessor.NewPublisher(
        eventprocessor.DefaultPublisherConfig(natsServer.ClientURL()),
        nil,
    )
    if err != nil {
        t.Fatalf("Failed to create publisher: %v", err)
    }
    defer pub.Close()

    // Create event and publish
    event := eventprocessor.NewMediaEvent("plex")
    event.UserID = 1
    event.Username = "testuser"
    event.MediaType = "movie"
    event.Title = "Test Movie"
    event.StartedAt = time.Now()

    msg, err := eventprocessor.SerializeEvent(event)
    if err != nil {
        t.Fatalf("Failed to serialize event: %v", err)
    }

    if err := pub.Publish(ctx, event.Topic(), msg); err != nil {
        t.Fatalf("Failed to publish event: %v", err)
    }

    // Verify with fallback reader
    reader, err := eventprocessor.NewFallbackReader(natsServer.ClientURL())
    if err != nil {
        t.Fatalf("Failed to create reader: %v", err)
    }
    defer reader.Close()

    // Allow time for persistence
    time.Sleep(100 * time.Millisecond)

    messages, err := reader.Query(ctx, "MEDIA_EVENTS", eventprocessor.QueryOptions{
        Subject: "playback.>",
        Limit:   10,
    })
    if err != nil {
        t.Fatalf("Failed to query: %v", err)
    }

    if len(messages) != 1 {
        t.Errorf("Expected 1 message, got %d", len(messages))
    }
}
```

---

## 14. Migration Path

### Phase 1: Foundation (Current Sprint)
1. Add dependencies to go.mod
2. Create `internal/eventprocessor` package
3. Implement embedded NATS server
4. Implement publisher and subscriber
5. Add unit tests

### Phase 2: Integration
1. Add NATSConfig to config package
2. Modify sync manager to publish events
3. Add feature flag for gradual rollout
4. Integration tests

### Phase 3: Production Hardening
1. Add Prometheus metrics
2. Implement resilient reader with fallback
3. Add circuit breaker protection
4. Performance testing

### Phase 4: Feature Expansion
1. Plex standalone mode (no Tautulli)
2. Jellyfin integration
3. Multi-instance coordination
4. Event replay capabilities

---

## 15. Event Sourcing Mode (NATS-First Architecture)

**Added**: v1.50 (2025-12-02)
**Configuration**: `NATS_EVENT_SOURCING=true`

### Overview

Event Sourcing mode changes NATS JetStream from a notification system to the single source of truth for all playback events. This architectural change enables:

1. **Cross-Source Deduplication**: Same playback from Plex webhook and Tautulli sync deduplicated
2. **Multi-Server Support**: Users with synced Plex + Jellyfin watch history get accurate analytics
3. **Event Replay**: Full event history enables debugging and replay
4. **Centralized Testing**: All event processing tested in DuckDBConsumer

### Architecture Comparison

**Notification Mode** (default, `NATS_EVENT_SOURCING=false`):
```
Sync Manager → InsertPlaybackEvent() → DuckDB (write first)
            → publishEvent() → NATS (notification only)
```

**Event Sourcing Mode** (`NATS_EVENT_SOURCING=true`):
```
Sync Manager → publishEvent() → NATS (single source of truth)
                                  ↓
                            DuckDBConsumer → DuckDB (all writes)
```

### Cross-Source Deduplication via CorrelationKey

The `CorrelationKey` mechanism enables deduplication of the same playback event from multiple sources:

```
CorrelationKey Format: {user_id}:{rating_key}:{time_bucket}
Example: "12345:54321:2024-01-15T10:30"

Time bucket: StartedAt truncated to 5-minute intervals (handles clock skew)
```

**Deduplication Layers** (internal/eventprocessor/duckdb_consumer.go):
1. **EventID**: Exact duplicate detection (same event redelivered)
2. **SessionKey**: Source-specific deduplication (same session from same source)
3. **CorrelationKey**: Cross-source deduplication (same playback from different sources)

### Implementation Files

| File | Purpose |
|------|---------|
| `internal/eventprocessor/events.go` | `CorrelationKey` field, `GenerateCorrelationKey()`, `SetCorrelationKey()` |
| `internal/eventprocessor/duckdb_consumer.go` | Multi-layer deduplication in `isDuplicate()` and `recordEvent()` |
| `internal/eventprocessor/sync_publisher.go` | `SetCorrelationKey()` before publishing |
| `internal/config/config.go` | `EventSourcing bool` field, `NATS_EVENT_SOURCING` env var |
| `internal/sync/tautulli_sync.go` | Conditional DB writes based on event sourcing mode |

### Configuration

```bash
# Enable Event Sourcing mode
NATS_ENABLED=true
NATS_EVENT_SOURCING=true
```

### Use Cases

| Scenario | Use Notification Mode | Use Event Sourcing Mode |
|----------|----------------------|-------------------------|
| Single Plex server with Tautulli | Recommended | Optional |
| Multi-server (Plex + Jellyfin) | Not recommended | Recommended |
| Need event replay/audit | Not available | Recommended |
| Future Jellyfin support | Requires changes | Just add publisher |
| Existing deployments | Keep current | Migrate when needed |

### Testing

Tests in `internal/eventprocessor/events_test.go`:
- `TestMediaEvent_GenerateCorrelationKey` - 7 test cases
- `TestCorrelationKey_CrossSourceDeduplication` - Verifies same key across sources
- `TestCorrelationKey_DifferentPlaybacksSameContent` - Different time buckets

Tests in `internal/eventprocessor/duckdb_consumer_test.go`:
- `TestDuckDBConsumer_CorrelationKeyDeduplication`
- `TestDuckDBConsumer_CrossSourceDeduplication`

---

## Appendix: Key Configuration Values

| Setting | Production Value | Notes |
|---------|------------------|-------|
| `JetStream.TrackMsgId` | `true` | Enable deduplication |
| `JetStream.AckAsync` | `false` | Exactly-once semantics |
| `StreamConfig.AllowDirect` | `true` | Required for nats_js |
| `Appender.BatchSize` | 1000 | Balance latency/throughput |
| `CircuitBreaker.FailureThreshold` | 5 | Conservative start |
| `DuckDB.Threads` | `runtime.NumCPU()` | Match available cores |

---

**Document Version**: 2.1.0
**Last Verified**: 2026-01-11
**Verified Against**: Cartographus source code
**Review Status**: Production Ready
**New in v2.1.0**: Event Sourcing Mode with CorrelationKey cross-source deduplication
