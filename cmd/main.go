package main

import (
	zl "protocol/logger"
)

func main() {
	//logger, _ := zap.NewProduction()
	//defer logger.Sync()
	//logger.Info("log info")
	//
	//log.Print("log info")
	//
	//config := zap.NewProductionConfig()
	//config.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	//logger2, _ := config.Build()
	//defer logger2.Sync()
	//logger2.Error("log error")
	//
	//logrus.SetLevel(logrus.ErrorLevel)
	//logrus.Error("log error")

	z := &zl.Config{
		JSON:            false,
		Level:           "debug",
		ComponentLevels: map[string]string{},
	}

	zl.InitLogConfig(z, "OpsLink")
	zl.Infow("123")
	zl.Warnw("warn", nil, "k1", "v1")
	test1()

	l := zl.InitL()
	l.GetLogger().Infow("2")
}

func test1() {
	test2()
	zl.Infow("1")
}

func test2() {
	zl.Infow("2")
}
