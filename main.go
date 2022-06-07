package main

import (
	"os"
	"raccoon/app"
	"raccoon/config"
	"raccoon/logger"
	"raccoon/metrics"
	"runtime/trace"
)

func main() {
	f, err := os.Create("trace.out")
	if err != nil {
		logger.Fatal("trace file error", err)
	}
	trace.Start(f)
	defer trace.Stop()
	config.Load()
	metrics.Setup()
	logger.SetLevel(config.Log.Level)
	err = app.Run()
	metrics.Close()
	if err != nil {
		logger.Fatal("init failure", err)
	}

}
