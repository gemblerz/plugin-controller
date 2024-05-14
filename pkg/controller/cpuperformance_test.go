package controller

import (
	"fmt"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"gotest.tools/v3/assert"
)

func setupTest(t *testing.T) func() {
	cgroupCPUPath := "/tmp/test/cgroup/cpu,cpuacct"
	if err := os.MkdirAll(cgroupCPUPath, os.ModePerm); err != nil {
		t.Fatal(err)
	}

	cgroupMemoryPath := "/tmp/test/cgroup/memory"
	if err := os.MkdirAll(cgroupMemoryPath, os.ModePerm); err != nil {
		t.Fatal(err)
	}

	stat := []byte(`cache 12816384
	rss 11882496
	rss_huge 2097152
	mapped_file 6733824
	dirty 4096
	writeback 0
	swap 0
	pgpgin 13466403
	pgpgout 13460884
	pgfault 20175479
	pgmajfault 86
	inactive_anon 7626752
	active_anon 4173824
	inactive_file 6606848
	active_file 6209536
	unevictable 0
	hierarchical_memory_limit 31457280
	hierarchical_memsw_limit 31457280
	total_cache 12816384
	total_rss 11882496
	total_rss_huge 2097152
	total_mapped_file 6733824
	total_dirty 4096
	total_writeback 0
	total_swap 0
	total_pgpgin 13466403
	total_pgpgout 13460884
	total_pgfault 20175479
	total_pgmajfault 86
	total_inactive_anon 7626752
	total_active_anon 4173824
	total_inactive_file 6606848
	total_active_file 6209536
	total_unevictable 0
	recent_rotated_anon 2591
	recent_rotated_file 286
	recent_scanned_anon 2595
	recent_scanned_file 546
		`)
	if err := os.WriteFile(path.Join(cgroupMemoryPath, "memory.stat"), stat, 0644); err != nil {
		t.Fatal(err)
	}

	total_memory := []byte(`28065792`)
	if err := os.WriteFile(path.Join(cgroupMemoryPath, "memory.usage_in_bytes"), total_memory, 0644); err != nil {
		t.Fatal(err)
	}
	return func() {
		// We may delete what we created...
	}
}

func TestReadCPU(t *testing.T) {
	cgroupCPUPath := "/tmp/test/cgroup/cpu,cpuacct"
	if err := os.MkdirAll(cgroupCPUPath, os.ModePerm); err != nil {
		t.Fatal(err)
	}

	c := NewCPUPerformanceLogging(ControllerConfig{
		AppCgroupDir: "/tmp/test/cgroup",
	})
	groundTruth := []uint64{
		0,
		0,
		0,
		0,
		0,
		0,
	}
	stat := []byte(strings.Trim(strings.Join(strings.Split(fmt.Sprint(groundTruth), " "), " "), "[]"))
	if err := os.WriteFile(path.Join(cgroupCPUPath, "cpuacct.usage_percpu"), stat, 0644); err != nil {
		t.Fatal(err)
	}
	c.ReadCPUPerc()

	groundTruthAfter := []uint64{
		1e9,
		1e9,
		1e9,
		1e9,
		1e9,
		1e9,
	}
	stat = []byte(strings.Trim(strings.Join(strings.Split(fmt.Sprint(groundTruthAfter), " "), " "), "[]"))
	if err := os.WriteFile(path.Join(cgroupCPUPath, "cpuacct.usage_percpu"), stat, 0644); err != nil {
		t.Fatal(err)
	}

	time.Sleep(1 * time.Second)
	cpuUtil, _ := c.ReadCPUPerc()
	t.Log(cpuUtil)
	// We pretend that we used 1 second on each of the 6 cores within a second.
	// This is equivalent of 600% utilization. With runtime error on this test,
	// the value should be close to 600% +- 10%
	assert.Assert(t, 600-cpuUtil < 10)
}

func TestReadCPUPerCPU(t *testing.T) {
	cgroupCPUPath := "/tmp/test/cgroup/cpu,cpuacct"
	if err := os.MkdirAll(cgroupCPUPath, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	c := NewCPUPerformanceLogging(ControllerConfig{
		AppCgroupDir: "/tmp/test/cgroup",
	})
	groundTruth := []uint64{
		825037625632,
		359430490816,
		37867331808,
		39416937312,
		38447872416,
		40916037344,
	}
	stat := []byte(strings.Trim(strings.Join(strings.Split(fmt.Sprint(groundTruth), " "), " "), "[]"))
	if err := os.WriteFile(path.Join(cgroupCPUPath, "cpuacct.usage_percpu"), stat, 0644); err != nil {
		t.Fatal(err)
	}
	values, err := c.ReadCPUSecondsPerCPU()
	if err != nil {
		t.Fatal(err)
	}
	for i, v := range values {
		e := float64(groundTruth[i]) / 1e9
		if e-v > 0.1 {
			t.Errorf("expected %f, received %f", e, v)
		}
	}
}

func TestReadMemory(t *testing.T) {
	cgroupMemoryPath := "/tmp/test/cgroup/memory"
	if err := os.MkdirAll(cgroupMemoryPath, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	stat := []byte(`cache 12816384
rss 11882496
rss_huge 2097152
mapped_file 6733824
dirty 4096
writeback 0
swap 0
pgpgin 13466403
pgpgout 13460884
pgfault 20175479
pgmajfault 86
inactive_anon 7626752
active_anon 4173824
inactive_file 6606848
active_file 6209536
unevictable 0
hierarchical_memory_limit 31457280
hierarchical_memsw_limit 31457280
total_cache 12816384
total_rss 11882496
total_rss_huge 2097152
total_mapped_file 6733824
total_dirty 4096
total_writeback 0
total_swap 0
total_pgpgin 13466403
total_pgpgout 13460884
total_pgfault 20175479
total_pgmajfault 86
total_inactive_anon 7626752
total_active_anon 4173824
total_inactive_file 6606848
total_active_file 6209536
total_unevictable 0
recent_rotated_anon 2591
recent_rotated_file 286
recent_scanned_anon 2595
recent_scanned_file 546
	`)
	if err := os.WriteFile(path.Join(cgroupMemoryPath, "memory.stat"), stat, 0644); err != nil {
		t.Fatal(err)
	}
	total_memory := []byte(`28065792`)
	if err := os.WriteFile(path.Join(cgroupMemoryPath, "memory.usage_in_bytes"), total_memory, 0644); err != nil {
		t.Fatal(err)
	}
	c := NewCPUPerformanceLogging(ControllerConfig{
		AppCgroupDir: "/tmp/test/cgroup",
	})
	memory, _ := c.ReadMemory()
	assert.Equal(t, memory, 21458944.)
}

func TestPrometheusEndpoint(t *testing.T) {
	setupTest(t)
	c := NewCPUPerformanceLogging(ControllerConfig{
		AppCgroupDir: "/tmp/test/cgroup",
	})
	expected := 8
	if got := testutil.CollectAndCount(c); got != expected {
		t.Errorf("unexpected metric count, got %d, want %d", got, expected)
	}
}
