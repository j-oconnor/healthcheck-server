package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	linuxproc "github.com/c9s/goprocinfo/linux"
	"github.com/docker/distribution/health"
)

type cpuMeasure struct {
	totalCPU uint64
	idleCPU  uint64
}

func (c *cpuMeasure) SetTotalCPU(totalCPU uint64) {
	c.totalCPU = totalCPU
}

func (c *cpuMeasure) SetIdleCPU(idleCPU uint64) {
	c.idleCPU = idleCPU
}

var lastCPUMeasure cpuMeasure

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])
}

// TCPChecker attempts to open a TCP connection.
func TCPChecker(addr string, timeout time.Duration) health.Checker {
	return health.CheckFunc(func() error {
		conn, err := net.DialTimeout("tcp", addr, timeout)
		if err != nil {
			return errors.New("connection to " + addr + " failed")
		}
		conn.Close()
		return nil
	})
}

func calcCPUUsage() float64 {
	var currentCPUMeasure cpuMeasure
	stat, err := linuxproc.ReadStat("/proc/stat")
	if err != nil {
		fmt.Println("failed to read /proc/stat.  Is this a linux host?")
		return 0
	}
	//extract values needed from stat table
	currentCPUMeasure.SetTotalCPU(stat.CPUStatAll.User + stat.CPUStatAll.Nice + stat.CPUStatAll.System + stat.CPUStatAll.Idle + stat.CPUStatAll.IOWait + stat.CPUStatAll.IRQ + stat.CPUStatAll.SoftIRQ + stat.CPUStatAll.Steal + stat.CPUStatAll.Guest + stat.CPUStatAll.GuestNice)
	currentCPUMeasure.SetIdleCPU(stat.CPUStatAll.Idle)

	// catch first execution and just set last measure to calculate on next loop
	if lastCPUMeasure.idleCPU == 0 && lastCPUMeasure.totalCPU == 0 {
		lastCPUMeasure.SetTotalCPU(currentCPUMeasure.totalCPU)
		lastCPUMeasure.SetIdleCPU(currentCPUMeasure.idleCPU)
		return 0
	}
	//perform stat calculation
	diffTotalCPU := float64(currentCPUMeasure.totalCPU - lastCPUMeasure.totalCPU)
	diffIdleCPU := float64(currentCPUMeasure.idleCPU - lastCPUMeasure.idleCPU)
	cpuPerc := (1.0 - diffIdleCPU/diffTotalCPU) * 100
	lastCPUMeasure.SetTotalCPU(currentCPUMeasure.totalCPU)
	lastCPUMeasure.SetIdleCPU(currentCPUMeasure.idleCPU)
	return cpuPerc
}

// CPUChecker to check cpu usage
func CPUChecker() health.Checker {
	return health.CheckFunc(func() error {
		cpuUsage := calcCPUUsage()
		if cpuUsage > 20.0 {
			return errors.New("High CPU Usage")
		}
		return nil
	})
}

func main() {
	health.Register("tcpCheck", health.PeriodicChecker(TCPChecker("127.0.0.1:8000", time.Second*5), time.Second*5))
	health.Register("cpuCheck", health.PeriodicChecker(CPUChecker(), time.Second*5))
	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", nil)
}
