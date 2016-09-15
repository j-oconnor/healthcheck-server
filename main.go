package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/docker/distribution/health"
)

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

func main() {
	health.Register("tcpCheck", health.PeriodicChecker(TCPChecker("127.0.0.1:8000", time.Second*5), time.Second*5))
	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", nil)
}
