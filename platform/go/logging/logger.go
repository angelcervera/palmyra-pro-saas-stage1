package logging

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Config defines the knobs for building a zap logger aligned with Google Cloud Logging expectations.
type Config struct {
	// Component identifies the emitting subsystem (e.g., "api-server").
	Component string
	// Level controls the minimum severity ("debug", "info", "warn", "error").
	Level string
}

// NewLogger builds a structured zap logger that emits Google Cloud Logging compatible fields.
func NewLogger(cfg Config) (*zap.Logger, error) {
	level := zap.NewAtomicLevel()
	if cfg.Level == "" {
		level.SetLevel(zapcore.InfoLevel)
	} else if err := level.UnmarshalText([]byte(strings.ToLower(cfg.Level))); err != nil {
		return nil, err
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "severity",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		EncodeTime:     zapcore.RFC3339NanoTimeEncoder,
		EncodeDuration: zapcore.MillisDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeLevel:    gcpLevelEncoder,
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(os.Stdout),
		level,
	)

	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	if cfg.Component != "" {
		logger = logger.With(zap.String("component", cfg.Component))
	}

	return logger, nil
}

func gcpLevelEncoder(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	switch l {
	case zapcore.DebugLevel:
		enc.AppendString("DEBUG")
	case zapcore.InfoLevel:
		enc.AppendString("INFO")
	case zapcore.WarnLevel:
		enc.AppendString("WARNING")
	case zapcore.ErrorLevel:
		enc.AppendString("ERROR")
	case zapcore.DPanicLevel, zapcore.PanicLevel:
		enc.AppendString("ALERT")
	case zapcore.FatalLevel:
		enc.AppendString("CRITICAL")
	default:
		enc.AppendString(strings.ToUpper(l.String()))
	}
}
