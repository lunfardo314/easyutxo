package testutil

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewSimpleLogger(debug bool) *zap.SugaredLogger {
	cfg := zap.NewDevelopmentConfig()
	cfg.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("04:05.000")
	log, err := cfg.Build()
	if err != nil {
		panic(err)
	}
	lvl := zapcore.InfoLevel
	if debug {
		lvl = zapcore.DebugLevel
	}
	log = log.WithOptions(zap.IncreaseLevel(lvl), zap.AddStacktrace(zapcore.FatalLevel))
	return log.Sugar()
}
