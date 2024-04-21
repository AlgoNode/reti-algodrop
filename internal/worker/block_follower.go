package worker

import (
	"context"
	"time"

	"github.com/algorand/go-algorand-sdk/v2/client/v2/common/models"
	"github.com/algorand/go-algorand-sdk/v2/types"
)

type BlockInfo struct {
	round    uint64
	proposer types.Address
	fees     types.MicroAlgos
}

type BlockFollower struct {
	nextRound uint64
	bvChan    chan *BlockInfo
}

func (w *ALGODROPWorker) parseBlock(round uint64, block *models.BlockResponse) {
	prop := ((*block.Cert)["prop"]).(map[interface{}]interface{})
	oprop := prop["oprop"].([]uint8)
	var proposer types.Address
	copy(proposer[:], oprop)
	w.Log.WithField("round", round).Infof("proposer:%s", proposer.String())

	var fees types.MicroAlgos

	for i := range block.Block.Payset {
		fees += block.Block.Payset[i].Txn.Fee
	}

	w.bvChan <- &BlockInfo{
		round:    round,
		proposer: proposer,
		fees:     fees,
	}
}

func (w *ALGODROPWorker) blockFollower(ctx context.Context) {
	//Loop until Algoverse gets cancelled
	var lastRound uint64 = 0
	w.HC.Ping("blockFollower", time.Second*15)
	w.nextRound = 1
	for {
		if ctx.Err() != nil {
			return
		}
		if w.nextRound > lastRound {
			s, err := w.C.Aapi.WaitForRoundAfter(ctx, w.nextRound-1)
			if err != nil {
				w.Log.WithError(err).Error("Error waiting for the next round")
				continue
			}
			if s.LastRound > w.nextRound {
				w.Log.Infof("Round %d, last is %d", w.nextRound, s.LastRound)
				if w.nextRound == 1 {
					w.nextRound = s.LastRound
				}
			}
			lastRound = s.LastRound
		}
		block, err := w.C.Aapi.GetBlockWithCert(ctx, w.nextRound)
		if err != nil {
			w.Log.WithError(err).Errorf("Error getting block for round %d", w.nextRound)
			continue
		}
		w.parseBlock(w.nextRound, block)
		w.nextRound++
		w.HC.Ping("blockFollower", time.Second*15)
	}
}
