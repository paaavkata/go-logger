package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
	"gopkg.in/natefinch/lumberjack.v2"
)

type logLevel string

const (
	LevelInfo  logLevel = "INFO"
	LevelWarn  logLevel = "WARNING"
	LevelError logLevel = "ERROR"
	LevelDebug logLevel = "DEBUG"
	LevelFatal logLevel = "FATAL"
)

var (
	infoLogger       *log.Logger
	warningLogger    *log.Logger
	errorLogger      *log.Logger
	debugLogger      *log.Logger
	structuredWriter io.Writer

	currentLevel string
	serviceName  string
	environment  string
)

type jsonLogger struct {
	logType string
	writer  io.Writer
}

func SetMetadata(service, env string) {
	serviceName = service
	environment = env
}

func (j *jsonLogger) Write(p []byte) (n int, err error) {
	msg := strings.TrimSuffix(string(p), "\n")
	msg = strings.ReplaceAll(msg, "\n", " ")

	logEntry := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"level":     j.logType,
		"message":   msg,
	}

	jsonData, err := json.Marshal(logEntry)
	if err != nil {
		return 0, err
	}
	jsonData = append(jsonData, '\n')
	return j.writer.Write(jsonData)
}

func Init(
	logLevel string, // "debug", "info", "warn", "error"
	logFormat string,
	serviceName string,
	environment string,
	writeToAFile bool,
	writeToStdout bool,
	sendToAKafkaQueue bool,
	kafkaBrokers *[]string,
	kafkaTopic *string) {

	currentLevel = logLevel

	var writers []io.Writer

	if writeToAFile {
		// ✅ Log rotation with lumberjack
		rotatingFile := &lumberjack.Logger{
			Filename:   "app.log",
			MaxSize:    10, // MB
			MaxBackups: 5,
			MaxAge:     28,   // days
			Compress:   true, // gzip
		}
		writers = append(writers, rotatingFile)
	}

	if writeToStdout {
		writers = append(writers, os.Stdout)
	}

	if sendToAKafkaQueue {
		writers = append(writers, newKafkaWriter(*kafkaBrokers, *kafkaTopic))
	}

	multiWriter := io.MultiWriter(writers...)
	structuredWriter = multiWriter

	if logFormat == "json" {
		infoLogger = log.New(&jsonLogger{"INFO", multiWriter}, "", 0)
		warningLogger = log.New(&jsonLogger{"WARNING", multiWriter}, "", 0)
		errorLogger = log.New(&jsonLogger{"ERROR", multiWriter}, "", 0)
		debugLogger = log.New(&jsonLogger{"DEBUG", multiWriter}, "", 0)
	} else {
		flags := log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile
		infoLogger = log.New(multiWriter, "INFO: ", flags)
		warningLogger = log.New(multiWriter, "WARNING: ", flags)
		errorLogger = log.New(multiWriter, "ERROR: ", flags)
		debugLogger = log.New(multiWriter, "DEBUG: ", flags)
	}
}

func shouldLog(level logLevel) bool {
	switch strings.ToLower(currentLevel) {
	case "debug":
		return true
	case "info":
		return level != LevelDebug
	case "warn":
		return level == LevelWarn || level == LevelError || level == LevelFatal
	case "error":
		return level == LevelError || level == LevelFatal
	default:
		return true
	}
}

func Info(msg string) {
	if shouldLog(LevelInfo) {
		infoLogger.Output(2, msg)
	}
}
func Warning(msg string) {
	if shouldLog(LevelWarn) {
		warningLogger.Output(2, msg)
	}
}
func Error(msg string) {
	if shouldLog(LevelError) {
		errorLogger.Output(2, msg)
	}
}
func Debug(msg string) {
	if shouldLog(LevelDebug) {
		debugLogger.Output(2, msg)
	}
}

func Fatal(msg string) {
	if shouldLog(LevelFatal) {
		errorLogger.Output(2, msg)
		os.Exit(1)
	}
}

func Infof(msg string, args ...interface{}) {
	if shouldLog(LevelInfo) {
		infoLogger.Output(2, fmt.Sprintf(msg, args...))
	}
}
func Warningf(msg string, args ...interface{}) {
	if shouldLog(LevelWarn) {
		warningLogger.Output(2, fmt.Sprintf(msg, args...))
	}
}
func Errorf(msg string, args ...interface{}) {
	if shouldLog(LevelError) {
		errorLogger.Output(2, fmt.Sprintf(msg, args...))
	}
}
func Debugf(msg string, args ...interface{}) {
	if shouldLog(LevelDebug) {
		debugLogger.Output(2, fmt.Sprintf(msg, args...))
	}
}
func Fatalf(msg string, args ...interface{}) {
	if shouldLog(LevelFatal) {
		errorLogger.Output(2, fmt.Sprintf("FATAL: "+msg, args...))
		os.Exit(1)
	}
}

func logWithMap(level logLevel, ctx context.Context, fields map[string]interface{}) {
	if !shouldLog(level) {
		return
	}

	fields["service"] = serviceName
	fields["environment"] = environment
	fields["timestamp"] = time.Now().Format(time.RFC3339)
	fields["level"] = level

	if ctx != nil {
		if traceID := ctx.Value("trace_id"); traceID != nil {
			fields["trace_id"] = traceID
		}
	}

	jsonData, err := json.Marshal(fields)
	if err != nil {
		errorLogger.Output(2, fmt.Sprintf("Failed to marshal structured log: %v", err))
		return
	}
	jsonData = append(jsonData, '\n')

	if structuredWriter != nil {
		_, _ = structuredWriter.Write(jsonData)
	}

	if level == LevelFatal {
		os.Exit(1)
	}
}

func InfofMap(ctx context.Context, fields map[string]interface{}) { logWithMap(LevelInfo, ctx, fields) }
func WarningfMap(ctx context.Context, fields map[string]interface{}) {
	logWithMap(LevelWarn, ctx, fields)
}
func ErrorfMap(ctx context.Context, fields map[string]interface{}) {
	logWithMap(LevelError, ctx, fields)
}
func DebugfMap(ctx context.Context, fields map[string]interface{}) {
	logWithMap(LevelDebug, ctx, fields)
}

func (k *kafkaLogWriter) Write(p []byte) (int, error) {
	msg := kafka.Message{
		Topic: k.topic,
		Value: p,
	}
	err := k.writer.WriteMessages(context.Background(), msg)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func newKafkaWriter(kafkaBrokers []string, kafkaTopic string) io.Writer {
	return &kafkaLogWriter{
		topic: kafkaTopic,
		writer: &kafka.Writer{
			Addr:         kafka.TCP(kafkaBrokers...), // Replace with your broker(s)
			Balancer:     &kafka.LeastBytes{},
			RequiredAcks: kafka.RequireNone,
		},
	}
}

type kafkaLogWriter struct {
	writer *kafka.Writer
	topic  string
}
