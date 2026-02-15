package logger

import "go.uber.org/zap"

var Log *zap.Logger

func Init(env string) {
	if env == "dev" {
		Log, _ = zap.NewDevelopment()
	} else {
		Log, _ = zap.NewProduction()
	}
}
