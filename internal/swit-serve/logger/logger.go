package logger

import "go.uber.org/zap"

var (
	Log *zap.Logger
)

func Init() {
	logger, _ := zap.NewProduction()
	Log = logger
}
