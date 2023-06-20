package controller

import (
	"fmt"
	"time"

	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
	"github.com/waggle-sensor/edge-scheduler/pkg/logger"

	"github.com/shirou/gopsutil/v3/process"
)

// this program prints out logs of performance, plugin stdout/stderr, current control params, events
// to pluginctl watch myplugin to printout those to users
// later, this can be consumed by the scheduler
// or this can be streamed to a time-series database

type ControllerConfig struct {
	EnableCPUPerformanceLogging bool
	EnableGPUPerformanceLogging bool
	PluginProcessName           string
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
	logger.Info.Println("plugin controller started.")
	ch := make(chan datatype.Event)
	pids, _ := process.Pids()
	var pluginProc *process.Process
	logger.Info.Printf("looking for the plugin process (%s) ...", c.config.PluginProcessName)
	for _, pid := range pids {
		if p, err := process.NewProcess(pid); err != nil {
			logger.Error.Printf("failed to get process %d: %s", pid, err.Error())
		} else {
			if pName, err := p.Name(); err == nil {
				logger.Info.Printf("pid %d (%s) found", pid, pName)
				if c.config.PluginProcessName == pName {
					pluginProc = p
					break
				}
			}
		}
	}
	if pluginProc == nil {
		panic(fmt.Sprintf("failed to find the plugin process (%s)", c.config.PluginProcessName))
	}
	if c.config.EnableCPUPerformanceLogging {
		logger.Info.Println("CPU performance measurement enabled")
		if c.config.AppCgroupDir == "" {
			logger.Info.Println("plugin cgroup directory is not given.")
			c.config.AppCgroupDir = fmt.Sprintf("/proc/%d/root/sys/fs/cgroup", pluginProc.Pid)
			logger.Info.Printf("plugin cgroup path found: %s", c.config.AppCgroupDir)
		}
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
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ticker.C:
			if pluginPidExists, err := process.PidExists(pluginProc.Pid); err == nil {
				if !pluginPidExists {
					logger.Info.Printf("plugin's PID (%d) does not exist", pluginProc.Pid)
					logger.Info.Println("plugin is terminated. plugin-controller terminates successfully.")
					return
				}
			} else {
				logger.Error.Printf("failed to probe plugin PID (%d): %s", pluginProc.Pid, err.Error())
			}
		case e := <-ch:
			data, _ := e.EncodeMetaToJson()
			logger.Info.Printf("%s: %s", e.ToString(), data)
		}
	}
}
