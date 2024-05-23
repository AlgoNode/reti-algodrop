package worker

import (
	"context"
	"time"

	"github.com/algorand/go-algorand-sdk/v2/client/v2/common/models"
	"github.com/algorand/go-algorand-sdk/v2/crypto"
	"github.com/algorand/go-algorand-sdk/v2/types"
)

type OAccount struct {
	Algos float64
}

func (w *ALGODROPWorker) updateOnlineCache(ctx context.Context) {
	if w.SParams == nil {
		return
	}
	accounts, err := w.C.Gapi.OnlineAccounts(ctx, uint64(w.SParams.FirstRoundValid))
	if err != nil || accounts == nil {
		w.Log.Errorf("error getting online accounts: %s", err)
		return
	}
	var stake float64
	var oa OnlineAccts = make(OnlineAccts, len(accounts.Accounts.Nodes))
	for i := range accounts.Accounts.Nodes {
		algos := accounts.Accounts.Nodes[i].Algos
		stake += algos
		var addr types.Address
		if err := addr.UnmarshalText([]byte(accounts.Accounts.Nodes[i].Addr)); err != nil {
			w.Log.Errorf("invalid address %s : %s", accounts.Accounts.Nodes[i].Addr, err)
		} else {
			oa[addr] = algos
		}
	}
	w.setOAccts(oa)
	w.Log.Infof("Got %d online accounts with total stake %fA", len(accounts.Accounts.Nodes), stake)
}

func (w *ALGODROPWorker) updateSuggestedParams(ctx context.Context) {
	txParams, err := w.C.Aapi.SuggestedParams(ctx)
	if err != nil {
		w.Log.WithError(err).Error("Error getting suggested tx params")
		return
	}
	w.Log.Infof("Suggested first round is %d", txParams.FirstRoundValid)
	txParams.FirstRoundValid--
	txParams.LastRoundValid = txParams.FirstRoundValid + DefaultValidRoundRange
	txParams.Fee = types.MicroAlgos(txParams.MinFee)
	txParams.FlatFee = true
	w.Lock()
	w.SParams = &txParams
	w.Unlock()
}

func (w *ALGODROPWorker) updateAccountInfo(ctx context.Context) {
	aInfo, err := w.C.Aapi.Client.AccountInformation(w.SenderAddr).Do(ctx)
	if err != nil {
		w.Log.WithError(err).Panic("Error getting account Info")
		return
	}
	w.Log.Infof("Account balance is %.3fA", float64(aInfo.Amount)/1000000.0)
	w.Lock()
	w.SenderInfo = &aInfo
	w.Unlock()
}

func getTealInt(gs []models.TealKeyValue, key string) uint64 {
	for _, kv := range gs {
		if kv.Key == key {
			return kv.Value.Uint
		}
	}
	return 0
}

func (w *ALGODROPWorker) updateReti(ctx context.Context) {
	var rinfo RetiInfo
	onl := w.getOAccts()
	app, err := w.C.Aapi.GetApp(ctx, RetiAppId)
	if err != nil {
		w.Log.WithError(err).Error("error getting Reti app global state")
		return
	}
	rinfo.Validators = getTealInt(app.Params.GlobalState, RetiKeyValidators)
	rinfo.Staked = types.MicroAlgos(getTealInt(app.Params.GlobalState, RetiKeyStaked))
	rinfo.Stakers = getTealInt(app.Params.GlobalState, RetiKeyStakers)
	w.Log.Infof("Reti validators:%d, staked:%f, stakers:%d", rinfo.Validators, rinfo.Staked.ToAlgos(), rinfo.Stakers)
	var op OnlinePools = make(OnlinePools, rinfo.Validators*2)

	for i := uint64(1); i <= rinfo.Validators; i++ {
		vi, err := w.getRetiValidatorInfo(ctx, i)
		if err != nil {
			w.Log.WithError(err).Errorf("vid:%d", i)
			continue
		}
		w.Log.Infof("vid:%d/%d Token:%d, Min:%.0f", i, vi.Config.ID, vi.Config.RewardTokenID, float64(vi.Config.MinEntryStake)/1_000_000.0)

		if vi.Config.SunsettingOn > 0 {
			w.Log.Warnf("vid:%d sunsetting at %d", i, vi.Config.SunsettingOn)
			continue
		}
		for pi := 0; pi < int(vi.State.NumPools); pi++ {
			// !suspended && online && hasStakers
			p := &vi.Pools[pi]
			paddr := crypto.GetApplicationAddress(p.PoolAppId)
			if p.TotalStakers == 0 {
				w.Log.Warnf("vid:%d pool:%d empty", i, pi)
				continue
			}
			if !onl.IsOnline(paddr) {
				w.Log.Warnf("vid:%d pool:%d addr:%s not online", i, pi, paddr)
				continue
			}
			w.Log.Infof("EligiblePool:%d@%d App:%d Addr:%s Stakers:%d Staked:%dA",
				pi, vi.Config.ID, p.PoolAppId, paddr, p.TotalStakers, p.TotalAlgoStaked/1000000)
			op[paddr] = float64(p.TotalAlgoStaked) / 1000000.0
		}
	}
	w.setOPools(op)
}

func (w *ALGODROPWorker) cacheUpdater(ctx context.Context) {
	//Loop until Algoverse gets cancelled
	for {
		if ctx.Err() != nil {
			return
		}
		w.HC.Ping("cacheUpdater", time.Second*time.Duration(w.C.Cfg.ADrop.Sleep)*3)
		w.updateAccountInfo(ctx)
		w.updateSuggestedParams(ctx)
		w.updateOnlineCache(ctx)
		w.updateReti(ctx)
		select {
		case <-ctx.Done():
		case <-time.After(time.Second * time.Duration(w.C.Cfg.ADrop.Sleep)):
		}
	}
}
