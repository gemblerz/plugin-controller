package controller

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
	"github.com/waggle-sensor/edge-scheduler/pkg/interfacing"
	"github.com/waggle-sensor/edge-scheduler/pkg/logger"
)

type CPUPerformanceLogging struct {
	CgroupDir         string
	Notifier          *interfacing.Notifier
	quit              chan struct{}
	interval          int
	lastTotalCPUUsed  float64
	lastTotalCPUUsedT time.Time

	promCPUSecondsPerCPU *prometheus.Desc
	promCPUSeconds       *prometheus.Desc
	promMemoryWorkingSet *prometheus.Desc
}

func NewCPUPerformanceLogging(c ControllerConfig) *CPUPerformanceLogging {
	return &CPUPerformanceLogging{
		CgroupDir:         c.AppCgroupDir,
		Notifier:          interfacing.NewNotifier(),
		quit:              make(chan struct{}),
		interval:          c.PerformanceCollectionInterval,
		lastTotalCPUUsed:  0,
		lastTotalCPUUsedT: time.Now(),

		promCPUSecondsPerCPU: prometheus.NewDesc(
			"plugin_per_cpu_seconds_total",
			"Cumulative plugin cpu time consumped per cpu core in seconds",
			[]string{"cpu"},
			nil,
		),
		promCPUSeconds: prometheus.NewDesc(
			"plugin_cpu_seconds_total",
			"Cumulative plugin cpu time consumped in seconds",
			nil,
			nil,
		),
		promMemoryWorkingSet: prometheus.NewDesc(
			"plugin_memory_workingset_bytes",
			"Amount of working set memory in bytes",
			nil,
			nil,
		),
	}
}

func (c *CPUPerformanceLogging) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.promCPUSecondsPerCPU
	ch <- c.promCPUSeconds
	ch <- c.promMemoryWorkingSet
}

func (c *CPUPerformanceLogging) Collect(ch chan<- prometheus.Metric) {
	if values, err := c.ReadCPUSecondsPerCPU(); err != nil {
		logger.Error.Printf("Error on ReadCPUSecondsPerCPU: %s", err.Error())
	} else {
		total := 0.
		for index, cpuSecond := range values {
			ch <- prometheus.MustNewConstMetric(
				c.promCPUSecondsPerCPU,
				prometheus.CounterValue,
				cpuSecond,
				fmt.Sprint(index),
			)
			total += cpuSecond
		}
		ch <- prometheus.MustNewConstMetric(
			c.promCPUSeconds,
			prometheus.CounterValue,
			total,
		)
	}
	if workingSet, err := c.ReadMemory(); err != nil {
		logger.Error.Printf("Error on ReadCPUSecondsPerCPU: %s", err.Error())
	} else {
		ch <- prometheus.MustNewConstMetric(
			c.promMemoryWorkingSet,
			prometheus.GaugeValue,
			workingSet,
		)
	}
}

func (c *CPUPerformanceLogging) ReadCPUSecondsPerCPU() ([]float64, error) {
	cpuacctSubDir := "cpu,cpuacct"
	cpuacctUsagePerCPUFile := "cpuacct.usage_percpu"
	buffer, err := os.ReadFile(path.Join(path.Join(c.CgroupDir, cpuacctSubDir, cpuacctUsagePerCPUFile)))
	if err != nil {
		return []float64{}, err
	}
	values := strings.Split(strings.TrimSpace(string(buffer)), " ")
	out := make([]float64, len(values))
	for index, strNanoSeconds := range values {
		if strNanoSeconds == "" {
			logger.Error.Printf("%v has an empty string. skipping", values)
			continue
		}
		nanoSeconds, err := strconv.ParseUint(strings.TrimSpace(strNanoSeconds), 10, 64)
		if err != nil {
			return out, err
		}
		out[index] = float64(nanoSeconds) / 1e9
	}
	return out, nil
}

// ReadCPUPerc reads cpuacct.stat file and returns an averaged per-second CPU utiltizaiton since
// the last read. The expected format is
//
// user x
//
// system y
func (c *CPUPerformanceLogging) ReadCPUPerc() (float64, error) {
	values, err := c.ReadCPUSecondsPerCPU()
	if err != nil {
		return 0., err
	}
	total := 0.
	for _, v := range values {
		total += v
	}
	if total < 0.1 {
		return 0., nil
	}
	delta := total - c.lastTotalCPUUsed
	deltaT := time.Since(c.lastTotalCPUUsedT).Seconds()
	c.lastTotalCPUUsedT = time.Now()
	// delta := (total - c.lastTotalCPUUsed) / uint64(deltaT)
	return delta / deltaT * 100, nil
}

// ReadMemory returns current container workingset memory in bytes
// workingset memory is the amount that cannot be evicted and
// calculated by total used memory - total inactive file
func (c *CPUPerformanceLogging) ReadMemory() (float64, error) {
	memorySubDir := "memory"
	totalUsedMemoryFile := "memory.usage_in_bytes"
	statFile := "memory.stat"
	totalUsedMemoryByte, err := os.ReadFile(path.Join(c.CgroupDir, memorySubDir, totalUsedMemoryFile))
	if err != nil {
		return 0, err
	}
	totalUsedMemory, err := strconv.ParseUint(strings.TrimSpace(string(totalUsedMemoryByte)), 10, 64)
	if err != nil {
		return 0, err
	}
	statBuffer, err := os.ReadFile(path.Join(c.CgroupDir, memorySubDir, statFile))
	if err != nil {
		return 0, err
	}
	totalInactive, err := getUintValueFromMatch(statBuffer, `total_inactive_file [0-9]+`)
	if err != nil {
		return 0, err
	}
	return float64(totalUsedMemory - totalInactive), nil
}

func (c *CPUPerformanceLogging) Stop() {
	c.quit <- struct{}{}
}

func (c *CPUPerformanceLogging) Run() {
	ticker := time.NewTicker(time.Duration(c.interval) * time.Second)
	for {
		select {
		case <-ticker.C:
			if mem, err := c.ReadMemory(); err == nil {
				e := datatype.NewEventBuilder(datatype.EventPluginPerfMem).
					AddValue(mem).
					Build()
				c.Notifier.Notify(e)
			} else {
				logger.Error.Println(err.Error())
			}
			if cpu, err := c.ReadCPUPerc(); err == nil {
				e := datatype.NewEventBuilder(datatype.EventPluginPerfCPU).
					AddValue(cpu).
					Build()
				c.Notifier.Notify(e)
			} else {
				logger.Error.Println(err.Error())
			}
		case <-c.quit:
			ticker.Stop()
			return
		}
	}
}

func getUintValueFromMatch(buf []byte, matchString string) (uint64, error) {
	re := regexp.MustCompile(matchString)
	matches := re.FindStringSubmatch(string(buf[:]))
	if len(matches) != 1 {
		return 0, fmt.Errorf("failed to get total_inactive_file value from %s", buf)
	}
	sp := strings.Split(matches[0], " ")
	if len(sp) != 2 {
		return 0, fmt.Errorf("failed to split value from %s", matches[0])
	}
	n, err := strconv.ParseUint(sp[1], 10, 64)
	if err != nil {
		return 0, err
	}
	return n, nil
}
