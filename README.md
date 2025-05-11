# go-logger

A lightweight, efficient, and production-grade structured logging module for Go.

Supports:

- ✅ JSON and plain text output
- ✅ Log level filtering (`debug`, `info`, `warn`, `error`, `fatal`)
- ✅ Context-aware structured logs (e.g. trace IDs)
- ✅ Output to stdout, file (with rotation), or Kafka
- ✅ Newline sanitization for structured logs
- ✅ Reusable across microservices

---

## 📦 Installation

```bash
go get github.com/paaavkata/go-logger@latest
```

---

## 🚀 Usage

### Logger Initialization

```go
import "github.com/paaavkata/go-logger/logger"

func main() {
	brokers := []string{"localhost:9092"}
	topic := "logs"

	logger.Init(
		logLevel = "debug",
		logFormat = "json",
		serviceName = "file-service",
		environment = "dev",
		writeToAFile = true,
		writeToStdout = true,
		sendToAKafkaQueue = true,
		kafkaBrokers = &brokers,
		kafkaTopic = &topic,
	)

	logger.Infof("Server started on port %d", 8080)
}
```

---

## 🧾 Examples

### Plain Logging

```go
logger.Info("service started")
logger.Warning("disk usage warning")
logger.Debugf("request payload: %v", payload)
logger.Errorf("failed to reach DB: %v", err)
```

### Fatal (exits app)

```go
logger.Fatal("could not connect to redis")
```

---

### Structured Logging

```go
ctx := context.WithValue(context.Background(), "trace_id", "abc-123")

logger.InfofMap(ctx, map[string]interface{}{
	"event": "user_signup",
	"user":  "johndoe",
	"ip":    "192.168.0.1",
})
```

#### Output (JSON):
```json
{
  "timestamp": "2025-05-11T19:30:12Z",
  "level": "INFO",
  "serviceName": "file-service",
  "environment": "dev",
  "event": "user_signup",
  "user": "johndoe",
  "ip": "192.168.0.1",
  "trace_id": "abc-123"
}
```

---

## 🧪 Running Tests

```bash
go test ./logger -v
```

Covers:

- JSON formatting
- Timestamp formatting
- Trace context
- Newline sanitization

---

## 🔧 Advanced Features

- [x] Log file rotation (via `lumberjack`)
- [x] Kafka integration (via `segmentio/kafka-go`)
- [x] Context injection for traceability
- [x] Structured map-based logging
- [ ] Buffered Kafka writer (coming soon)
- [ ] gRPC metadata integration (coming soon)

---

## 📌 License

MIT © Pavel Damyanov
