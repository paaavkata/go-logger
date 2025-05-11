package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"strings"
	"testing"
	"time"
)

func TestPlainInfoLog_JSONFormat(t *testing.T) {
	var buf bytes.Buffer
	initTestLogger(&buf, "json", "debug")

	Info("plain info message")
	checkLogJSON(t, buf.String(), "INFO", "plain info message")
}

func TestFormattedDebugLog_JSONFormat(t *testing.T) {
	var buf bytes.Buffer
	initTestLogger(&buf, "json", "debug")

	Debugf("this is a %s message", "debug")
	checkLogJSON(t, buf.String(), "DEBUG", "this is a debug message")
}

func TestStructuredMapLog_JSONFormat(t *testing.T) {
	var buf bytes.Buffer
	initTestLogger(&buf, "json", "debug")

	ctx := context.WithValue(context.Background(), "trace_id", "xyz-123")
	InfofMap(ctx, map[string]interface{}{
		"event": "deploy",
		"app":   "logger-service",
	})

	var result map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &result)
	if err != nil {
		t.Fatalf("invalid JSON in structured log: %v", err)
	}

	if result["event"] != "deploy" {
		t.Errorf("expected event=deploy, got %v", result["event"])
	}
	if result["trace_id"] != "xyz-123" {
		t.Errorf("expected trace_id to be xyz-123")
	}
}

func TestNewlineSanitization(t *testing.T) {
	var buf bytes.Buffer
	initTestLogger(&buf, "json", "debug")

	Error("this message has a newline\nsecond line")

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("invalid JSON log: %v", err)
	}

	msg, ok := logEntry["message"].(string)
	if !ok {
		t.Fatalf("log message is missing or not a string")
	}

	if strings.Contains(msg, "\n") {
		t.Errorf("log message contains newline character: %q", msg)
	}
}

func TestTimestampFormatRFC3339(t *testing.T) {
	var buf bytes.Buffer
	initTestLogger(&buf, "json", "debug")

	Warning("timestamp test")
	var out map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &out)
	if err != nil {
		t.Fatalf("error unmarshaling log: %v", err)
	}

	timestamp := out["timestamp"].(string)
	if _, err := time.Parse(time.RFC3339, timestamp); err != nil {
		t.Errorf("timestamp not in RFC3339: %s", timestamp)
	}
}

func initTestLogger(buf *bytes.Buffer, format, level string) {
	currentLevel = level
	structuredWriter = buf

	infoLogger = log.New(&jsonLogger{"INFO", buf}, "", 0)
	warningLogger = log.New(&jsonLogger{"WARNING", buf}, "", 0)
	errorLogger = log.New(&jsonLogger{"ERROR", buf}, "", 0)
	debugLogger = log.New(&jsonLogger{"DEBUG", buf}, "", 0)
}

func checkLogJSON(t *testing.T, logLine string, expectedLevel, expectedMessage string) {
	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(logLine), &logEntry); err != nil {
		t.Fatalf("invalid JSON log: %v\nlog: %s", err, logLine)
	}
	if logEntry["level"] != expectedLevel {
		t.Errorf("expected level %s, got %v", expectedLevel, logEntry["level"])
	}
	if !strings.Contains(logEntry["message"].(string), expectedMessage) {
		t.Errorf("log message does not contain expected text: %s", expectedMessage)
	}
}
