package worker

import (
	"context"

	"github.com/algonode/reti-algodrop/internal/worker/common"
	"github.com/sirupsen/logrus"
)

const (
	SingletonALGODROP = "algodrop"
)

type ALGODROPWorker struct {
	common.WorkerCommon
	RewardSender
	BlockFollower
}

func ALGODROPWorkerNew(ctx context.Context, apis *common.WorkerAPIs, log *logrus.Logger) common.Worker {
	return &ALGODROPWorker{
		WorkerCommon: common.MakeCommonWorker(SingletonALGODROP, ctx, apis, log, false, false),
		RewardSender: *RewardSenderNew(ctx, apis, log),
	}
}

func (w *ALGODROPWorker) Config(ctx context.Context) error {
	if v, ok := w.C.Cfg.WSnglt[SingletonALGODROP]; !ok || !v {
		w.Log.Infof("%s disabled, skipping configuration", SingletonALGODROP)
		return nil
	}

	w.bvChan = make(chan *BlockInfo, 1000)
	w.Enable()

	return nil
}

func (w *ALGODROPWorker) Spawn(ctx context.Context) error {
	if v, ok := w.C.Cfg.WSnglt[SingletonALGODROP]; !ok || !v {
		w.Log.Infof("%s disabled, not spawning", SingletonALGODROP)
		return nil
	}
	go w.algoDropper(ctx)
	go w.blockFollower(ctx)
	go w.cacheUpdater(ctx)
	return nil
}

func init() {
	common.RegisterWorker(SingletonALGODROP, ALGODROPWorkerNew)
}
