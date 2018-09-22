package common

import (
	"go.uber.org/zap"
)

// Logger zap logger
var Logger *zap.SugaredLogger

func init() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	Logger = logger.Sugar()
}
