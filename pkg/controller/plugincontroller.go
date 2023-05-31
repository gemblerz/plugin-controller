package controller

import (
	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
	"github.com/waggle-sensor/edge-scheduler/pkg/logger"
)

// this program prints out logs of performance, plugin stdout/stderr, current control params, events
// to pluginctl watch myplugin to printout those to users
// later, this can be consumed by the scheduler
// or this can be streamed to a time-series database

type ControllerConfig struct {
	EnableCPUPerformanceLogging bool
	EnableGPUPerformanceLogging bool
	AppCgroupDir                string
	GPUMetricHost               string
}

type Controller struct {
	config ControllerConfig
}

func NewController(c ControllerConfig) *Controller {
	return &Controller{
		config: c,
	}
}

func (c *Controller) Run() {
	ch := make(chan datatype.Event)
	if c.config.EnableCPUPerformanceLogging {
		logger.Info.Println("CPU performance measurement enabled")
		p := NewCPUPerformanceLogging(c.config)
		p.Notifier.Subscribe(ch)
		go p.Run()
	}
	if c.config.EnableGPUPerformanceLogging {
		logger.Info.Println("GPU performance measurement enabled")
		g := NewGPUPerformanceLogging(c.config)
		g.Notifier.Subscribe(ch)
		go g.Run()
	}

	for e := range ch {
		data, _ := e.EncodeMetaToJson()
		logger.Info.Printf("%s: %s", e.ToString(), data)
	}
}
