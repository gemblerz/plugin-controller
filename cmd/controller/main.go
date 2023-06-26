package main

import (
	"flag"
	"os"
	"strconv"

	"github.com/waggle-sensor/plugin-controller/pkg/controller"
)

func getenv(key string, def string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return def
}

func mustParseInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

func main() {
	var config controller.ControllerConfig
	var configPath string
	// config.Version = Version
	// flag.BoolVar(&config.Debug, "debug", false, "flag to debug")
	flag.StringVar(&configPath, "config", "", "path to config file")
	flag.BoolVar(&config.EnableCPUPerformanceLogging, "enable-cpu-performance", false, "Enable CPU performance logging")
	flag.BoolVar(&config.EnableGPUPerformanceLogging, "enable-gpu-performance", false, "Enable GPU performance logging")
	flag.IntVar(&config.PerformanceCollectionInterval, "performance-collection-interval", 5, "Interval in seconds to collect performance metrics")
	flag.BoolVar(&config.EnableMetricsPublishing, "enable-metrics-publishing", false, "Attempt to publish metrcis to RabbitMQ")
	flag.StringVar(&config.MetricsPublishingScope, "metrics-publishing-scope", getenv("WAGGLE_PUBLISHING_SCOPE", "node"), "Scope to publish metrics. Default is node")
	flag.StringVar(&config.RabbitMQHost, "rabbitmq-host", getenv("WAGGLE_PLUGIN_HOST", "rabbitmq"), "Host to RabbitMQ")
	flag.IntVar(&config.RabbitMQPort, "rabbitmq-port", mustParseInt(getenv("WAGGLE_PLUGIN_PORT", "5672")), "Port to RabbitMQ")
	flag.StringVar(&config.RabbitMQUsername, "rabbitmq-username", getenv("WAGGLE_PLUGIN_USERNAME", "plugin"), "RabbitMQ username")
	flag.StringVar(&config.RabbitMQPassword, "rabbitmq-password", getenv("WAGGLE_PLUGIN_PASSWORD", "plugin"), "RabbitMQ password")
	flag.StringVar(&config.RabbitMQAppID, "rabbitmq-app-id", getenv("WAGGLE_APP_ID", ""), "App ID for RabbitMQ publishing")
	flag.StringVar(&config.PluginProcessName, "plugin-process-name", "", "Process name of the plugin")
	// flag.StringVar(&config.AppCgroupDir, "app-cgroup-dir", "data", "Path to meta directory")
	flag.StringVar(&config.GPUMetricHost, "gpu-metric-host", getenv("GPU_METRIC_HOST", ""), "Host IP for Prometheus-formatted GPU metric")
	flag.Parse()
	c := controller.NewController(config)
	c.Run()
}
