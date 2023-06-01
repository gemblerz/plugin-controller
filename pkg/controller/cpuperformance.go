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
	CgroupDir string
	Notifier  *interfacing.Notifier
	quit      chan struct{}
}

func NewCPUPerformanceLogging(c ControllerConfig) *CPUPerformanceLogging {
	return &CPUPerformanceLogging{
		CgroupDir: c.AppCgroupDir,
		Notifier:  interfacing.NewNotifier(),
		quit:      make(chan struct{}),
	}
}

// readMemory returns current container workingset memory in bytes
// workingset memory is calculated by total used memory - total inactive file
func (c *CPUPerformanceLogging) readMemory() (int, error) {
	memorySubDir := "memory"
	totalUsedMemoryFile := "memory.usage_in_bytes"
	statFile := "memory.stat"
	totalUsedMemoryByte, err := os.ReadFile(path.Join(c.CgroupDir, memorySubDir, totalUsedMemoryFile))
	if err != nil {
		return 0, err
	}
	totalUsedMemory, err := strconv.Atoi(strings.TrimSpace(string(totalUsedMemoryByte)))
	if err != nil {
		return 0, err
	}
	statBuffer, err := os.ReadFile(path.Join(c.CgroupDir, memorySubDir, statFile))
	if err != nil {
		return 0, err
	}

	re := regexp.MustCompile(`total_inactive_file [0-9]+`)
	matches := re.FindStringSubmatch(string(statBuffer[:]))
	if len(matches) != 1 {
		return 0, fmt.Errorf("failed to get total_inactive_file value from %s", statBuffer)
	}
	sp := strings.Split(matches[0], " ")
	if len(sp) != 2 {
		return 0, fmt.Errorf("failed to split value from %s", matches[0])
	}
	totalInactive, err := strconv.Atoi(sp[1])
	if err != nil {
		return 0, err
	}
	return totalUsedMemory - totalInactive, nil
}

func (c *CPUPerformanceLogging) Stop() {
	c.quit <- struct{}{}
}

func (c *CPUPerformanceLogging) Run() {
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ticker.C:
			if mem, err := c.readMemory(); err == nil {
				e := datatype.NewEventBuilder(datatype.EventPluginPerfMem).
					AddEntry("memory", strconv.Itoa(mem)).
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
