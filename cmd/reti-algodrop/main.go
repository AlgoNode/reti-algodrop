// Copyright (C) 2022 AlgoNode Org.
//
// reti-algodrop is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// reti-algodrop is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with reti-algodrop.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/algonode/reti-algodrop/internal/algodapi"
	"github.com/algonode/reti-algodrop/internal/config"
	"github.com/algonode/reti-algodrop/internal/gqlapi"
	_ "github.com/algonode/reti-algodrop/internal/worker"
	"github.com/algonode/reti-algodrop/internal/worker/common"

	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stderr)
	log.SetLevel(log.InfoLevel)
}

func main() {

	slog := log.StandardLogger()

	//load config
	cfg, err := config.LoadConfig()
	if err != nil {
		log.WithError(err).Error()
		return
	}

	//make us a nice cancellable context
	//set Ctrl-C as the cancell trigger
	ctx, cf := context.WithCancel(context.Background())
	defer cf()
	{
		cancelCh := make(chan os.Signal, 1)
		signal.Notify(cancelCh, syscall.SIGTERM, syscall.SIGINT)
		go func() {
			<-cancelCh
			log.Error("stopping streamer")
			cf()
		}()
	}

	aapi, err := algodapi.Make(ctx, cfg.Algod, slog)
	if err != nil {
		log.Panic(err)
	}

	gapi, err := gqlapi.Make(ctx, cfg.GQL, slog)
	if err != nil {
		log.Panic(err)
	}

	wc := &common.WorkerAPIs{
		Gapi: gapi,
		Aapi: aapi,
		Cfg:  &cfg,
	}

	slog.Info("[BOOT] workers")
	if err := common.BootWorkers(ctx, wc, slog); err != nil {
		slog.Error(err)
		cf()
	}

	// workers := []common.Worker{
	// 	worker.ALGODROPWorkerNew(ctx, apis, slog),
	// 	worker.VOTESCANWorkerNew(ctx, apis, slog),
	// }

	// for _, w := range workers {
	// 	if err := w.Config(ctx); err != nil {
	// 		log.Panic(err)
	// 	}
	// }

	// for _, w := range workers {
	// 	if err := w.Spawn(ctx); err != nil {
	// 		log.Panic(err)
	// 	}
	// }

	go common.HealthServer(ctx, slog)

	<-ctx.Done()
	time.Sleep(time.Second)

	log.Error("Bye!")

}
