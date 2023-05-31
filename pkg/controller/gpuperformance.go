package controller

import (
	"github.com/waggle-sensor/edge-scheduler/pkg/interfacing"
)

type GPUPerformanceLogging struct {
	GPUMetricHost string
	Notifier      *interfacing.Notifier
}

func NewGPUPerformanceLogging(c ControllerConfig) *GPUPerformanceLogging {
	return &GPUPerformanceLogging{
		GPUMetricHost: c.GPUMetricHost,
		Notifier:      interfacing.NewNotifier(),
	}
}

func (g *GPUPerformanceLogging) Run() {

}
