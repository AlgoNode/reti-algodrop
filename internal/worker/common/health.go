package common

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

var healthCheck = flag.Bool("hc", false, "check if microservice is healthy")

type HealthCheck struct {
	Name     string
	LastOK   time.Time
	Deadline time.Time
}
type HealthChecks struct {
	sync.Mutex
	enabled bool
	log     *logrus.Logger
	HCS     map[string]*HealthCheck
}

func (w *WorkerCommon) Enable() {
	w.HC.enabled = true
}

func (hcs *HealthChecks) Ping(name string, deadline time.Duration) {
	hcs.Lock()
	defer hcs.Unlock()
	hc, found := hcs.HCS[name]
	if !found {
		hc = &HealthCheck{
			Name: name,
		}
		hcs.HCS[name] = hc
		hcs.log.Infof("Created healthcheck %s with deadline %s", name, deadline)
	}
	hc.LastOK = time.Now()
	hc.Deadline = hc.LastOK.Add(deadline)
}

var gHealthScore int64 = 0

func KumaHttpPing(log *logrus.Logger, status string, msg string) {
	kuma := os.Getenv("KUMAPING")
	if kuma == "" {
		return
	}
	requestURL := fmt.Sprintf("%s?status=%s&msg=%s&ping=", kuma, url.QueryEscape(status), url.QueryEscape(msg))
	_, err := http.Get(requestURL)
	if err != nil {
		log.Errorf("error making http request: %s\n", err)
	}
}

func WorkersHealthy(log *logrus.Logger) bool {
	for _, w := range regWorkers {
		if w.worker == nil {
			continue
		}
		if !w.worker.Health() {
			if log != nil {
				gHealthScore--
				if gHealthScore < -5 {
					log.Panic("HealthScore")
				}
				log.Errorf("HealthScore: %d", gHealthScore)
			}
			KumaHttpPing(log, "down", w.name)
			return false
		}
	}
	gHealthScore = 0
	if log != nil {
		KumaHttpPing(log, "up", "OK")
	}
	return true
}

func (w *WorkerCommon) Health() bool {
	if !w.HC.enabled {
		return true
	}
	w.HC.Lock()
	defer w.HC.Unlock()
	for _, a := range w.HC.HCS {
		if time.Now().After(a.Deadline) {
			w.Log.Errorf("healthCheck %s of worker %s failed", a.Name, w.Name)
			return false
		}
	}
	w.Log.Infof("Worker %s healthy", w.Name)
	return true
}

func HealthServer(ctx context.Context, log *logrus.Logger) {
	http.HandleFunc("/livez", httpHealthCheckFunc)
	http.HandleFunc("/readyz", httpHealthCheckFunc)
	http.HandleFunc("/healthz", httpHealthCheckFunc)
	go http.ListenAndServe(":11314", nil)
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			WorkersHealthy(log)
		}
	}
}

func httpHealthCheckFunc(w http.ResponseWriter, req *http.Request) {
	if !WorkersHealthy(nil) {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func httpHealthCheck() bool {
	requestURL := fmt.Sprintf("http://localhost:%d/livez", 11314)
	res, err := http.Get(requestURL)
	if err != nil {
		return false
	}
	return res.StatusCode == http.StatusOK
}

func ProcessHealthCheck() {
	flag.Parse()

	if healthCheck != nil && *healthCheck {
		if httpHealthCheck() {
			os.Exit(0)
		}
		os.Exit(1)
	}
}
