package controller

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
	"github.com/waggle-sensor/edge-scheduler/pkg/interfacing"
	"github.com/waggle-sensor/edge-scheduler/pkg/logger"
)

type GPUPerformanceLogging struct {
	GPUMetricHost string
	Notifier      *interfacing.Notifier
	quit          chan struct{}
	interval      int
}

func NewGPUPerformanceLogging(c ControllerConfig) *GPUPerformanceLogging {
	return &GPUPerformanceLogging{
		GPUMetricHost: c.GPUMetricHost,
		Notifier:      interfacing.NewNotifier(),
		quit:          make(chan struct{}),
		interval:      c.PerformanceCollectionInterval,
	}
}

// getGPUMetric returns
func (g *GPUPerformanceLogging) getGPUMetric() (float64, error) {
	s, err := url.JoinPath(fmt.Sprintf("http://%s:9101", g.GPUMetricHost), "metrics")
	if err != nil {
		return 0, err
	}
	resp, err := http.Get(s)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	re := regexp.MustCompile(`gpu_average_load1s [0-9.]+`)
	matches := re.FindStringSubmatch(string(body[:]))
	if len(matches) != 1 {
		return 0, fmt.Errorf("failed to get total_inactive_file value from %s", body[:])
	}
	sp := strings.Split(matches[0], " ")
	if len(sp) != 2 {
		return 0, fmt.Errorf("failed to split value from %s", matches[0])
	}
	gpuUtil, err := strconv.ParseFloat(sp[1], 64)
	if err != nil {
		return 0, err
	}
	// gpuUtil reported from wes-jetson-exporter ranges from [0., 1.]
	return gpuUtil * 100., nil
}

func (g *GPUPerformanceLogging) Stop() {
	g.quit <- struct{}{}
}

func (g *GPUPerformanceLogging) Run() {
	ticker := time.NewTicker(time.Duration(g.interval) * time.Second)
	for {
		select {
		case <-ticker.C:
			if u, err := g.getGPUMetric(); err == nil {
				e := datatype.NewEventBuilder(datatype.EventPluginPerfGPU).
					AddValue(u).
					Build()
				g.Notifier.Notify(e)
			} else {
				logger.Error.Println(err.Error())
			}
		case <-g.quit:
			ticker.Stop()
			return
		}
	}
}
