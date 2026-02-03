// Package logger provides structured logging capabilities for the application.
// It wraps uber-go/zap for high-performance, leveled logging with JSON output support.
package logger

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/verustcode/verustcode/internal/model"
)

// bufferpool is a pool of buffers for efficient memory allocation
var bufferpool = buffer.NewPool()

var (
	// Global logger instance
	globalLogger *zap.Logger
	once         sync.Once
	// taskLogHook is the global task log hook for capturing task-related logs
	taskLogHook *TaskLogHook
)

// Config holds the logger configuration
type Config struct {
	// Level is the minimum log level (debug, info, warn, error)
	Level string `yaml:"level"`
	// Format is the output format (json, text)
	Format string `yaml:"format"`
	// File is the log file path (empty for stdout only)
	// When set, logs are written to both console and file
	File string `yaml:"file"`
	// MaxSize is the maximum size in megabytes of the log file before it gets rotated
	MaxSize int `yaml:"max_size"`
	// MaxAge is the maximum number of days to retain old log files
	MaxAge int `yaml:"max_age"`
	// MaxBackups is the maximum number of old log files to retain
	MaxBackups int `yaml:"max_backups"`
	// Compress determines if the rotated log files should be compressed using gzip
	Compress bool `yaml:"compress"`
	// AccessLog determines if HTTP request logs should be printed at info level
	// When true, successful requests (status < 400) are logged; when false, they are not
	// Default: false
	AccessLog bool `yaml:"access_log"`
}

// Init initializes the global logger with the given configuration.
// This function is safe to call multiple times; only the first call will take effect.
func Init(cfg Config) error {
	var initErr error
	once.Do(func() {
		initErr = initLogger(cfg)
	})
	return initErr
}

// initLogger creates and sets the global logger
func initLogger(cfg Config) error {
	// Parse log level
	level, err := parseLevel(cfg.Level)
	if err != nil {
		level = zapcore.InfoLevel
	}

	// Apply default values for rotation config
	if cfg.MaxSize <= 0 {
		cfg.MaxSize = 100 // Default 100 MB
	}
	if cfg.MaxAge <= 0 {
		cfg.MaxAge = 7 // Default 7 days
	}
	if cfg.MaxBackups <= 0 {
		cfg.MaxBackups = 5 // Default 5 backups
	}

	if cfg.Format == "text" {
		// Text format: use custom encoder with key=value style for structured fields
		globalLogger = buildTextLogger(level, cfg)
		return nil
	}

	// JSON format: build logger with dual output
	globalLogger = buildJSONLogger(level, cfg)
	return nil
}

// buildTextLogger creates a text logger with key=value format for structured fields
// When file is configured, logs are written to both console (with color) and file (without color)
func buildTextLogger(level zapcore.Level, cfg Config) *zap.Logger {
	// Console encoder config (with color)
	consoleEncoderConfig := zapcore.EncoderConfig{
		TimeKey:          "time",
		LevelKey:         "level",
		NameKey:          zapcore.OmitKey, // Remove logger name
		CallerKey:        "caller",
		FunctionKey:      zapcore.OmitKey,
		MessageKey:       "msg",
		StacktraceKey:    "stacktrace",
		LineEnding:       zapcore.DefaultLineEnding,
		EncodeLevel:      bracketColorLevelEncoder,      // [INFO] with color
		EncodeTime:       bracketTimeEncoder,            // [2006-01-02 15:04:05]
		EncodeDuration:   zapcore.StringDurationEncoder, // Human-readable duration
		EncodeCaller:     zapcore.ShortCallerEncoder,    // Short file path
		ConsoleSeparator: " ",                           // Use space separator
	}

	// File encoder config (without color)
	fileEncoderConfig := zapcore.EncoderConfig{
		TimeKey:          "time",
		LevelKey:         "level",
		NameKey:          zapcore.OmitKey,
		CallerKey:        "caller",
		FunctionKey:      zapcore.OmitKey,
		MessageKey:       "msg",
		StacktraceKey:    "stacktrace",
		LineEnding:       zapcore.DefaultLineEnding,
		EncodeLevel:      bracketLevelEncoder, // [INFO] without color
		EncodeTime:       bracketTimeEncoder,  // [2006-01-02 15:04:05]
		EncodeDuration:   zapcore.StringDurationEncoder,
		EncodeCaller:     zapcore.ShortCallerEncoder,
		ConsoleSeparator: " ",
	}

	// Create console encoder
	consoleEncoder := newKVConsoleEncoder(consoleEncoderConfig)
	consoleWriter := zapcore.AddSync(os.Stdout)
	consoleCore := zapcore.NewCore(consoleEncoder, consoleWriter, level)

	var core zapcore.Core
	if cfg.File != "" {
		// Ensure directory exists
		dir := filepath.Dir(cfg.File)
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create log directory: %v, using console only\n", err)
			core = consoleCore
		} else {
			// Create file encoder (without color)
			fileEncoder := newKVConsoleEncoder(fileEncoderConfig)
			// Use lumberjack for log rotation
			fileWriter := zapcore.AddSync(&lumberjack.Logger{
				Filename:   cfg.File,
				MaxSize:    cfg.MaxSize,    // megabytes
				MaxAge:     cfg.MaxAge,     // days
				MaxBackups: cfg.MaxBackups, // number of backups
				Compress:   cfg.Compress,   // compress old files
			})
			fileCore := zapcore.NewCore(fileEncoder, fileWriter, level)
			// Combine console and file cores using Tee
			core = zapcore.NewTee(consoleCore, fileCore)
		}
	} else {
		// Console only
		core = consoleCore
	}

	// Build logger with caller info and stack trace for errors
	// Note: No AddCallerSkip here, as it's only needed for package-level wrapper functions
	return zap.New(core,
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
}

// buildJSONLogger creates a JSON format logger
// When file is configured, logs are written to both console and file
func buildJSONLogger(level zapcore.Level, cfg Config) *zap.Logger {
	// JSON encoder config
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	encoder := zapcore.NewJSONEncoder(encoderConfig)
	consoleWriter := zapcore.AddSync(os.Stdout)
	consoleCore := zapcore.NewCore(encoder, consoleWriter, level)

	var core zapcore.Core
	if cfg.File != "" {
		// Ensure directory exists
		dir := filepath.Dir(cfg.File)
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create log directory: %v, using console only\n", err)
			core = consoleCore
		} else {
			// Use lumberjack for log rotation
			fileWriter := zapcore.AddSync(&lumberjack.Logger{
				Filename:   cfg.File,
				MaxSize:    cfg.MaxSize,
				MaxAge:     cfg.MaxAge,
				MaxBackups: cfg.MaxBackups,
				Compress:   cfg.Compress,
			})
			fileCore := zapcore.NewCore(encoder, fileWriter, level)
			// Combine console and file cores using Tee
			core = zapcore.NewTee(consoleCore, fileCore)
		}
	} else {
		// Console only
		core = consoleCore
	}

	// Build logger with caller info and stack trace for errors
	// Note: No AddCallerSkip here, as it's only needed for package-level wrapper functions
	return zap.New(core,
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
}

// customTimeEncoder formats time as YYYY-MM-DD HH:MM:SS for better readability
func customTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05"))
}

// bracketTimeEncoder formats time with brackets: [2006-01-02 15:04:05]
func bracketTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString("[" + t.Format("2006-01-02 15:04:05") + "]")
}

// bracketLevelEncoder formats level with brackets and color: [INFO]
func bracketLevelEncoder(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString("[" + level.CapitalString() + "]")
}

// bracketColorLevelEncoder formats level with brackets and color: [INFO] (with ANSI color)
func bracketColorLevelEncoder(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	// ANSI color codes
	var color string
	switch level {
	case zapcore.DebugLevel:
		color = "\x1b[35m" // Magenta
	case zapcore.InfoLevel:
		color = "\x1b[34m" // Blue
	case zapcore.WarnLevel:
		color = "\x1b[33m" // Yellow
	case zapcore.ErrorLevel, zapcore.DPanicLevel, zapcore.PanicLevel, zapcore.FatalLevel:
		color = "\x1b[31m" // Red
	default:
		color = "\x1b[0m" // Reset
	}
	reset := "\x1b[0m"
	enc.AppendString(color + "[" + level.CapitalString() + "]" + reset)
}

// parseLevel converts a string level to zapcore.Level
func parseLevel(level string) (zapcore.Level, error) {
	var l zapcore.Level
	err := l.UnmarshalText([]byte(level))
	return l, err
}

// Get returns the global logger instance.
// If the logger hasn't been initialized, it returns a no-op logger.
func Get() *zap.Logger {
	if globalLogger == nil {
		// Return a no-op logger if not initialized
		return zap.NewNop()
	}
	return globalLogger
}

// Sugar returns the sugared global logger for more convenient logging.
func Sugar() *zap.SugaredLogger {
	return Get().Sugar()
}

// With creates a child logger with additional fields
func With(fields ...zap.Field) *zap.Logger {
	return Get().With(fields...)
}

// Named creates a child logger with the given name
func Named(name string) *zap.Logger {
	return Get().Named(name)
}

// Debug logs a debug message
func Debug(msg string, fields ...zap.Field) {
	Get().WithOptions(zap.AddCallerSkip(1)).Debug(msg, fields...)
}

// Info logs an info message
func Info(msg string, fields ...zap.Field) {
	Get().WithOptions(zap.AddCallerSkip(1)).Info(msg, fields...)
}

// Warn logs a warning message
func Warn(msg string, fields ...zap.Field) {
	Get().WithOptions(zap.AddCallerSkip(1)).Warn(msg, fields...)
}

// Error logs an error message
func Error(msg string, fields ...zap.Field) {
	Get().WithOptions(zap.AddCallerSkip(1)).Error(msg, fields...)
}

// Fatal logs a fatal message and exits
func Fatal(msg string, fields ...zap.Field) {
	Get().WithOptions(zap.AddCallerSkip(1)).Fatal(msg, fields...)
}

// Sync flushes any buffered log entries
func Sync() error {
	if globalLogger != nil {
		return globalLogger.Sync()
	}
	return nil
}

// SetTaskLogHook sets the global task log hook for capturing task-related logs.
// This should be called after Init() and before any logging that should be captured.
// The hook will intercept logs containing review_id or report_id fields.
func SetTaskLogHook(writer TaskLogWriter) {
	if globalLogger == nil {
		return
	}

	taskLogHook = NewTaskLogHook(writer)

	// Wrap the existing core with the hook
	globalLogger = globalLogger.WithOptions(
		zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			return taskLogHook.WrapCore(core)
		}),
	)
}

// CloseTaskLogHook closes the task log hook and flushes any remaining logs.
// This should be called during application shutdown.
func CloseTaskLogHook() {
	if taskLogHook != nil {
		taskLogHook.Close()
		taskLogHook = nil
	}
}

// FlushTaskLogs flushes any buffered task logs to storage.
func FlushTaskLogs() {
	if taskLogHook != nil {
		taskLogHook.Flush()
	}
}

// WithTaskContext creates a child logger with task context fields.
// This is a convenience function to ensure consistent task identification in logs.
// Example usage:
//
//	logger := logger.WithTaskContext(model.TaskTypeReview, reviewID)
//	logger.Info("Processing review")
func WithTaskContext(taskType model.TaskType, taskID string) *zap.Logger {
	if taskType == model.TaskTypeReview {
		return Get().With(zap.String(FieldReviewID, taskID))
	} else if taskType == model.TaskTypeReport {
		return Get().With(zap.String(FieldReportID, taskID))
	}
	return Get()
}

// kvConsoleEncoder wraps the standard console encoder but formats fields as key=value
type kvConsoleEncoder struct {
	zapcore.Encoder
	cfg zapcore.EncoderConfig
}

// newKVConsoleEncoder creates a new key=value console encoder
func newKVConsoleEncoder(cfg zapcore.EncoderConfig) zapcore.Encoder {
	return &kvConsoleEncoder{
		Encoder: zapcore.NewConsoleEncoder(cfg),
		cfg:     cfg,
	}
}

// Clone creates a copy of the encoder
func (e *kvConsoleEncoder) Clone() zapcore.Encoder {
	return &kvConsoleEncoder{
		Encoder: e.Encoder.Clone(),
		cfg:     e.cfg,
	}
}

// EncodeEntry encodes a log entry with key=value format for fields
func (e *kvConsoleEncoder) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	// Create a buffer for the output
	buf := bufferpool.Get()

	// Encode time
	if e.cfg.TimeKey != "" && e.cfg.EncodeTime != nil {
		arr := &sliceArrayEncoder{}
		e.cfg.EncodeTime(entry.Time, arr)
		for _, s := range arr.elems {
			buf.AppendString(s)
			buf.AppendString(e.cfg.ConsoleSeparator)
		}
	}

	// Encode level
	if e.cfg.LevelKey != "" && e.cfg.EncodeLevel != nil {
		arr := &sliceArrayEncoder{}
		e.cfg.EncodeLevel(entry.Level, arr)
		for _, s := range arr.elems {
			buf.AppendString(s)
			buf.AppendString(e.cfg.ConsoleSeparator)
		}
	}

	// Encode caller
	if e.cfg.CallerKey != "" && entry.Caller.Defined && e.cfg.EncodeCaller != nil {
		arr := &sliceArrayEncoder{}
		e.cfg.EncodeCaller(entry.Caller, arr)
		for _, s := range arr.elems {
			buf.AppendString(s)
			buf.AppendString(e.cfg.ConsoleSeparator)
		}
	}

	// Encode message
	if e.cfg.MessageKey != "" {
		buf.AppendString(entry.Message)
	}

	// Encode fields as key=value
	for _, field := range fields {
		buf.AppendString(e.cfg.ConsoleSeparator)
		buf.AppendString(field.Key)
		buf.AppendByte('=')
		appendFieldValue(buf, field)
	}

	// Add line ending
	if e.cfg.LineEnding != "" {
		buf.AppendString(e.cfg.LineEnding)
	} else {
		buf.AppendString(zapcore.DefaultLineEnding)
	}

	return buf, nil
}

// sliceArrayEncoder is a simple array encoder that stores strings
type sliceArrayEncoder struct {
	elems []string
}

func (s *sliceArrayEncoder) AppendBool(v bool)              { s.elems = append(s.elems, fmt.Sprint(v)) }
func (s *sliceArrayEncoder) AppendByteString(v []byte)      { s.elems = append(s.elems, string(v)) }
func (s *sliceArrayEncoder) AppendComplex128(v complex128)  { s.elems = append(s.elems, fmt.Sprint(v)) }
func (s *sliceArrayEncoder) AppendComplex64(v complex64)    { s.elems = append(s.elems, fmt.Sprint(v)) }
func (s *sliceArrayEncoder) AppendFloat64(v float64)        { s.elems = append(s.elems, fmt.Sprint(v)) }
func (s *sliceArrayEncoder) AppendFloat32(v float32)        { s.elems = append(s.elems, fmt.Sprint(v)) }
func (s *sliceArrayEncoder) AppendInt(v int)                { s.elems = append(s.elems, fmt.Sprint(v)) }
func (s *sliceArrayEncoder) AppendInt64(v int64)            { s.elems = append(s.elems, fmt.Sprint(v)) }
func (s *sliceArrayEncoder) AppendInt32(v int32)            { s.elems = append(s.elems, fmt.Sprint(v)) }
func (s *sliceArrayEncoder) AppendInt16(v int16)            { s.elems = append(s.elems, fmt.Sprint(v)) }
func (s *sliceArrayEncoder) AppendInt8(v int8)              { s.elems = append(s.elems, fmt.Sprint(v)) }
func (s *sliceArrayEncoder) AppendString(v string)          { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendUint(v uint)              { s.elems = append(s.elems, fmt.Sprint(v)) }
func (s *sliceArrayEncoder) AppendUint64(v uint64)          { s.elems = append(s.elems, fmt.Sprint(v)) }
func (s *sliceArrayEncoder) AppendUint32(v uint32)          { s.elems = append(s.elems, fmt.Sprint(v)) }
func (s *sliceArrayEncoder) AppendUint16(v uint16)          { s.elems = append(s.elems, fmt.Sprint(v)) }
func (s *sliceArrayEncoder) AppendUint8(v uint8)            { s.elems = append(s.elems, fmt.Sprint(v)) }
func (s *sliceArrayEncoder) AppendUintptr(v uintptr)        { s.elems = append(s.elems, fmt.Sprint(v)) }
func (s *sliceArrayEncoder) AppendDuration(v time.Duration) { s.elems = append(s.elems, v.String()) }
func (s *sliceArrayEncoder) AppendTime(v time.Time)         { s.elems = append(s.elems, v.String()) }
func (s *sliceArrayEncoder) AppendArray(v zapcore.ArrayMarshaler) error {
	return v.MarshalLogArray(s)
}
func (s *sliceArrayEncoder) AppendObject(v zapcore.ObjectMarshaler) error {
	return nil // Not supported for simple key=value format
}
func (s *sliceArrayEncoder) AppendReflected(v interface{}) error {
	s.elems = append(s.elems, fmt.Sprint(v))
	return nil
}

// appendFieldValue appends the field value to the buffer in key=value format
func appendFieldValue(buf *buffer.Buffer, field zapcore.Field) {
	switch field.Type {
	case zapcore.StringType:
		buf.AppendString(field.String)
	case zapcore.Int64Type, zapcore.Int32Type, zapcore.Int16Type, zapcore.Int8Type:
		buf.AppendInt(field.Integer)
	case zapcore.Uint64Type, zapcore.Uint32Type, zapcore.Uint16Type, zapcore.Uint8Type:
		buf.AppendUint(uint64(field.Integer))
	case zapcore.Float64Type:
		buf.AppendFloat(math.Float64frombits(uint64(field.Integer)), 64)
	case zapcore.Float32Type:
		buf.AppendFloat(float64(math.Float32frombits(uint32(field.Integer))), 32)
	case zapcore.BoolType:
		buf.AppendBool(field.Integer == 1)
	case zapcore.DurationType:
		buf.AppendString(time.Duration(field.Integer).String())
	case zapcore.TimeType:
		if field.Interface != nil {
			buf.AppendString(time.Unix(0, field.Integer).In(field.Interface.(*time.Location)).String())
		} else {
			buf.AppendString(time.Unix(0, field.Integer).String())
		}
	case zapcore.TimeFullType:
		buf.AppendString(field.Interface.(time.Time).String())
	case zapcore.ErrorType:
		if err, ok := field.Interface.(error); ok && err != nil {
			buf.AppendString(err.Error())
		} else {
			buf.AppendString("<nil>")
		}
	case zapcore.StringerType:
		if stringer, ok := field.Interface.(fmt.Stringer); ok {
			buf.AppendString(stringer.String())
		}
	default:
		// For complex types, use fmt.Sprint
		if field.Interface != nil {
			buf.AppendString(fmt.Sprint(field.Interface))
		}
	}
}
