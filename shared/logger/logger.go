package logger

import (
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	instance *zap.Logger
	once     sync.Once
)

func New() *zap.Logger {
	once.Do(func() {
		env := os.Getenv("APP_ENV")
		var cfg zap.Config
		if env == "production" {
			cfg = zap.NewProductionConfig()
		} else {
			cfg = zap.NewDevelopmentConfig()
			cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		}
		var err error
		instance, err = cfg.Build(zap.AddCallerSkip(1))
		if err != nil {
			panic("failed to initialize logger: " + err.Error())
		}
	})
	return instance
}

func Sync() {
	if instance != nil {
		_ = instance.Sync()
	}
}
