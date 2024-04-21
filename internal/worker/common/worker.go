package common

import (
	"context"
	"sync"

	"github.com/algonode/reti-algodrop/internal/algodapi"
	"github.com/algonode/reti-algodrop/internal/config"
	"github.com/algonode/reti-algodrop/internal/gqlapi"
	"github.com/sirupsen/logrus"
)

type Worker interface {
	Spawn(ctx context.Context) error
	Config(ctx context.Context) error
	Health() bool
}

type WorkerAPIs struct {
	Gapi *gqlapi.GqlAPI
	Aapi *algodapi.AlgodAPI
	Cfg  *config.NotifierConfig
}

type WorkerCommon struct {
	sync.RWMutex
	syncWorker bool
	Name       string
	Ctx        context.Context
	C          *WorkerAPIs
	Log        *logrus.Entry
	realtime   bool
	HC         *HealthChecks
}

func (w *WorkerCommon) Config(ctx context.Context) error {
	w.Log.Panic("Abstract worker called")
	return nil
}

func (w *WorkerCommon) Spawn(ctx context.Context) error {
	w.Log.Panic("Abstract worker called")
	return nil
}

type WorkerInitCB func(context.Context, *WorkerAPIs, *logrus.Logger) Worker

type RegisteredWorker struct {
	name   string
	worker Worker
	cb     WorkerInitCB
}

var regWorkers []RegisteredWorker = []RegisteredWorker{}

func RegisterWorker(name string, cb WorkerInitCB) {
	regWorkers = append(regWorkers, RegisteredWorker{name: name, cb: cb})
}

func MakeCommonWorker(name string, ctx context.Context, c *WorkerAPIs, log *logrus.Logger, sync bool, realtime bool) WorkerCommon {
	return WorkerCommon{
		Name:       name,
		Ctx:        ctx,
		C:          c,
		syncWorker: sync,
		Log:        log.WithFields(logrus.Fields{"wrk": name}),
		realtime:   realtime,
		HC: &HealthChecks{
			HCS: make(map[string]*HealthCheck),
			log: log,
		},
	}
}

func BootWorkers(ctx context.Context, c *WorkerAPIs, slog *logrus.Logger) error {
	slog.Infof("Booting %d workers", len(regWorkers))
	for i, w := range regWorkers {
		slog.Tracef("Initializing worker '%s", w.name)
		regWorkers[i].worker = w.cb(ctx, c, slog)
		slog.Infof("Initialized worker '%s' ", w.name)
	}
	for _, w := range regWorkers {
		if w.worker == nil {
			slog.Warnf("nil worker %s", w.name)
			continue
		}
		if err := w.worker.Config(ctx); err != nil {
			return err
		}
	}
	for _, w := range regWorkers {
		if w.worker == nil {
			slog.Warnf("nil worker %s", w.name)
			continue
		}
		if err := w.worker.Spawn(ctx); err != nil {
			return err
		}
	}
	return nil
}
