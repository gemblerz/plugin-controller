package controller

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
	"github.com/waggle-sensor/edge-scheduler/pkg/interfacing"
	"github.com/waggle-sensor/edge-scheduler/pkg/logger"
	"gopkg.in/cenkalti/backoff.v1"

	"github.com/shirou/gopsutil/v3/process"
)

// this program prints out logs of performance, plugin stdout/stderr, current control params, events
// to pluginctl watch myplugin to printout those to users
// later, this can be consumed by the scheduler
// or this can be streamed to a time-series database

const (
	PluginProcessStartedPath = "/app/started"
)

type ControllerConfig struct {
	EnableCPUPerformanceLogging   bool
	EnableGPUPerformanceLogging   bool
	PerformanceCollectionInterval int
	PluginProcessName             string
	AppCgroupDir                  string
	GPUMetricHost                 string
	EnableMetricsPublishing       bool
	MetricsPublishingScope        string
	RabbitMQHost                  string
	RabbitMQPort                  int
	RabbitMQUsername              string
	RabbitMQPassword              string
	RabbitMQAppID                 string
}

type Controller struct {
	config     ControllerConfig
	pluginProc *process.Process
	rmq        *interfacing.RabbitMQHandler
}

func NewController(c ControllerConfig) *Controller {
	return &Controller{
		config: c,
	}
}

// searchForPluginPID finds the plugin process ID from the process namespace and
// sets the PID in the struct. in case plugin process name is not given, it will
// search for any user process other than "pause" and "plugin-controller"
func (c *Controller) searchForPluginPID() error {
	blacklist := map[string]bool{
		"pause":             true,
		"plugin-controller": true,
	}
	pids, _ := process.Pids()
	for _, pid := range pids {
		if p, err := process.NewProcess(pid); err != nil {
			return fmt.Errorf("failed to get process %d: %s", pid, err.Error())
		} else {
			if pName, err := p.Name(); err == nil {
				logger.Info.Printf("pid %d (%s) found", pid, pName)
				if c.config.PluginProcessName != "" {
					if c.config.PluginProcessName == pName {
						logger.Info.Printf("set %d as plugin PID", p.Pid)
						c.pluginProc = p
						return nil
					}
				} else {
					if _, blacklisted := blacklist[pName]; !blacklisted {
						logger.Info.Printf("%d might be the plugin PID. setting it as plugin PID", p.Pid)
						c.pluginProc = p
						return nil
					}
				}
			}
		}
	}
	if _, err := os.Stat(PluginProcessStartedPath); err == nil {
		return &backoff.PermanentError{
			Err: fmt.Errorf("plugin might have finished its job already."),
		}
	} else {
		return fmt.Errorf("failed to find the process (%s)", c.config.PluginProcessName)
	}
}

func (c *Controller) Run() {
	logger.Info.Println("plugin controller started.")
	ch := make(chan datatype.Event)

	if c.config.EnableMetricsPublishing {
		rabbitMQURL := fmt.Sprintf("%s:%d", c.config.RabbitMQHost, c.config.RabbitMQPort)
		logger.Info.Printf("publishing metrics to %s", rabbitMQURL)
		c.rmq = interfacing.NewRabbitMQHandler(rabbitMQURL, c.config.RabbitMQUsername, c.config.RabbitMQPassword, "", c.config.RabbitMQAppID)
	}

	if c.config.PluginProcessName != "" {
		logger.Info.Printf("looking for the plugin process (%s) ...", c.config.PluginProcessName)
	} else {
		logger.Info.Printf("no plugin process name is given. looking for any user process in the process namespace")
	}

	backOffConfiguration := backoff.NewExponentialBackOff()
	// it should not stop searching for plugin PID
	backOffConfiguration.MaxElapsedTime = 0
	if err := backoff.Retry(c.searchForPluginPID, backOffConfiguration); err != nil {
		logger.Info.Println(err.Error())
		return
	}
	if c.config.EnableCPUPerformanceLogging {
		logger.Info.Println("CPU performance measurement enabled")
		if c.config.AppCgroupDir == "" {
			logger.Info.Println("plugin cgroup directory is not given.")
			c.config.AppCgroupDir = fmt.Sprintf("/proc/%d/root/sys/fs/cgroup", c.pluginProc.Pid)
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
			if pluginPidExists, err := process.PidExists(c.pluginProc.Pid); err == nil {
				if !pluginPidExists {
					logger.Info.Printf("plugin's PID (%d) does not exist", c.pluginProc.Pid)
					if _, err := os.Stat(PluginProcessStartedPath); errors.Is(err, os.ErrNotExist) {
						logger.Info.Printf("%s does not exist. the plugin has not yet started.", PluginProcessStartedPath)
					} else {
						logger.Info.Println("the plugin is terminated. plugin-controller terminates successfully.")
						return
					}
				}
			} else {
				logger.Error.Printf("failed to probe plugin PID (%d): %s", c.pluginProc.Pid, err.Error())
			}
		case e := <-ch:
			data, _ := e.EncodeMetaToJson()
			logger.Info.Printf("%s: %s", e.ToString(), data)
			if c.config.EnableMetricsPublishing {
				go c.rmq.SendWaggleMessage(e.ToWaggleMessage(), c.config.MetricsPublishingScope)
			}
		}
	}
}
