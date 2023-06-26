package controller

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
	"github.com/waggle-sensor/edge-scheduler/pkg/interfacing"
	"github.com/waggle-sensor/edge-scheduler/pkg/logger"
)

type CPUPerformanceLogging struct {
	CgroupDir         string
	Notifier          *interfacing.Notifier
	quit              chan struct{}
	interval          int
	lastTotalCPUUsed  uint64
	lastTotalCPUUsedT time.Time
}

func NewCPUPerformanceLogging(c ControllerConfig) *CPUPerformanceLogging {
	return &CPUPerformanceLogging{
		CgroupDir:         c.AppCgroupDir,
		Notifier:          interfacing.NewNotifier(),
		quit:              make(chan struct{}),
		interval:          c.PerformanceCollectionInterval,
		lastTotalCPUUsed:  0,
		lastTotalCPUUsedT: time.Now(),
	}
}

// readCPUPerc reads cpuacct.stat file and returns an averaged per-second CPU utiltizaiton since
// the last read. The expected format is
//
// user x
//
// system y
func (c *CPUPerformanceLogging) readCPUPerc() (float64, error) {
	cpuacctSubDir := "cpu,cpuacct"
	cpuacctStatFile := "cpuacct.stat"
	totalCPUUsedBuffer, err := os.ReadFile(path.Join(c.CgroupDir, cpuacctSubDir, cpuacctStatFile))
	if err != nil {
		return 0, err
	}
	user, err := getUintValueFromMatch(totalCPUUsedBuffer, `user [0-9]+`)
	if err != nil {
		return 0, err
	}
	system, err := getUintValueFromMatch(totalCPUUsedBuffer, `system [0-9]+`)
	if err != nil {
		return 0, err
	}
	total := user + system
	if total < 1 {
		return 0, nil
	}
	delta := total - c.lastTotalCPUUsed
	deltaT := time.Now().Sub(c.lastTotalCPUUsedT).Seconds()
	// delta := (total - c.lastTotalCPUUsed) / uint64(deltaT)
	return float64(delta) / float64(deltaT), nil
}

// readMemory returns current container workingset memory in bytes
// workingset memory is calculated by total used memory - total inactive file
func (c *CPUPerformanceLogging) readMemory() (uint64, error) {
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
	return totalUsedMemory - totalInactive, nil
}

func (c *CPUPerformanceLogging) Stop() {
	c.quit <- struct{}{}
}

func (c *CPUPerformanceLogging) Run() {
	ticker := time.NewTicker(time.Duration(c.interval) * time.Second)
	for {
		select {
		case <-ticker.C:
			if mem, err := c.readMemory(); err == nil {
				e := datatype.NewEventBuilder(datatype.EventPluginPerfMem).
					AddValue(mem).
					Build()
				c.Notifier.Notify(e)
			} else {
				logger.Error.Println(err.Error())
			}
			if cpu, err := c.readCPUPerc(); err == nil {
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
