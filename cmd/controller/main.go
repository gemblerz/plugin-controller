package main

import (
	"flag"
	"os"

	"github.com/waggle-sensor/plugin-controller/pkg/controller"
)

func getenv(key string, def string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return def
}

func main() {
	var config controller.ControllerConfig
	var configPath string
	// config.Version = Version
	// flag.BoolVar(&config.Debug, "debug", false, "flag to debug")
	flag.StringVar(&configPath, "config", "", "path to config file")
	flag.BoolVar(&config.EnableCPUPerformanceLogging, "enable-cpu-performance", false, "Enable CPU performance logging")
	flag.BoolVar(&config.EnableGPUPerformanceLogging, "enable-gpu-performance", false, "Enable GPU performance logging")
	flag.StringVar(&config.AppCgroupDir, "app-cgroup-dir", "data", "Path to meta directory")
	flag.StringVar(&config.GPUMetricHost, "gpu-metric-host", getenv("GPU_METRIC_HOST", ""), "Host IP for Prometheus-formatted GPU metric")
	flag.Parse()
	c := controller.NewController(config)
	c.Run()
}
