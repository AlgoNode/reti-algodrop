package worker

import (
	"context"
	"math/rand"
	"reflect"

	"github.com/algonode/reti-algodrop/internal/worker/common"
	"github.com/algorand/go-algorand-sdk/v2/client/v2/common/models"
	"github.com/algorand/go-algorand-sdk/v2/crypto"
	"github.com/algorand/go-algorand-sdk/v2/mnemonic"
	"github.com/algorand/go-algorand-sdk/v2/transaction"
	"github.com/algorand/go-algorand-sdk/v2/types"
	"github.com/sirupsen/logrus"
)

const DefaultValidRoundRange = 100

type OnlineAccts map[types.Address]float64
type OnlinePools map[types.Address]float64

type RewardSender struct {
	Account     *crypto.Account
	AuthAccount *crypto.Account
	SenderAddr  string
	SenderInfo  *models.Account
	SParams     *types.SuggestedParams
	OAccts      OnlineAccts
	OPools      OnlinePools
}

func (oa OnlineAccts) IsOnline(addr types.Address) bool {
	_, ok := oa[addr]
	return ok
}

func (w *ALGODROPWorker) getOAccts() OnlineAccts {
	w.RLock()
	defer w.RUnlock()
	return w.OAccts
}

func (w *ALGODROPWorker) setOAccts(oa OnlineAccts) {
	w.Lock()
	defer w.Unlock()
	w.OAccts = oa
}

func (w *ALGODROPWorker) getOPools() OnlinePools {
	w.RLock()
	defer w.RUnlock()
	return w.OPools
}

func (w *ALGODROPWorker) setOPools(op OnlinePools) {
	w.Lock()
	defer w.Unlock()
	w.OPools = op
}

func (w *ALGODROPWorker) getSenderInfo() *models.Account {
	w.RLock()
	defer w.RUnlock()
	return w.SenderInfo
}

func (w *ALGODROPWorker) getSuggestedParams() *types.SuggestedParams {
	w.RLock()
	defer w.RUnlock()
	return w.SParams
}

func RewardSenderNew(ctx context.Context, c *common.WorkerAPIs, log *logrus.Logger) *RewardSender {
	cfg := c.Cfg
	pkstr, ok := cfg.PKeys[cfg.ADrop.PKey]
	if !ok {
		log.Fatal("Missing private key for ", cfg.ADrop.PKey)
	}

	pk, err := mnemonic.ToPrivateKey(pkstr)
	if err != nil {
		log.Fatal("Error importing private key", err)
		return nil
	}

	account, err := crypto.AccountFromPrivateKey(pk)
	if err != nil {
		log.Fatal("Error creating account object", err)
		return nil
	}
	log.Infof("Expired key notifier account is %s", account.Address.String())

	apkstr, ok := cfg.PKeys[cfg.ADrop.AKey]
	if !ok {
		log.Fatal("Missing private key for ", cfg.ADrop.AKey)
	}

	apk, err := mnemonic.ToPrivateKey(apkstr)
	if err != nil {
		log.Fatal("Error importing private key", err)
		return nil
	}

	auth_account, err := crypto.AccountFromPrivateKey(apk)
	if err != nil {
		log.Fatal("Error creating account object", err)
		return nil
	}
	log.Infof("Expired key notifier auth_account is %s", auth_account.Address.String())

	return &RewardSender{
		Account:     &account,
		AuthAccount: &auth_account,
		SenderAddr:  account.Address.String(),
	}
}

func (w *ALGODROPWorker) drop(ctx context.Context, addr types.Address, amount types.MicroAlgos) {
	addrStr := addr.String()
	w.Log.Infof("Rewarding account %s with %.6f Algo", addrStr, amount.ToAlgos())

	txParams := w.getSuggestedParams()

	txn, err := transaction.MakePaymentTxn(
		w.SenderAddr,
		addrStr,
		uint64(amount), []byte("Incentives RWD simulator"), "", *txParams)

	if err != nil {
		w.Log.WithError(err).Errorf("error creating transaction for snd:%s rcv:%s", w.SenderAddr, addrStr)
		return
	}
	// probability of sending the same consecutive SND/RCV/Amount TX in is high, add random lease
	// this happens because of cached SuggestedParams
	crypto.RandomBytes(txn.Lease[:])

	// Sign the transaction
	_, signedTxn, err := crypto.SignTransaction(w.AuthAccount.PrivateKey, txn)
	if err != nil {
		w.Log.WithError(err).Error("Error signing transaction")
		return
	}
	sendResponse, err := w.C.Aapi.Client.SendRawTransaction(signedTxn).Do(ctx)
	if err != nil {
		w.Log.WithError(err).Error("Error sending transaction")
		return
	}
	w.Log.Infof("Submitted transaction %s", sendResponse)
}

func (w *ALGODROPWorker) dropAlgo(ctx context.Context, bv *BlockInfo) {
	onlPoolsMap := w.getOPools()
	if len(onlPoolsMap) == 0 {
		w.Log.Warnf("No online pool data yet, prize is gone.")
	}
	var winner types.Address
	prize := bv.fees
	// is proposer a Reti Pool ?
	_, hit := onlPoolsMap[bv.proposer]
	if hit {
		winner = bv.proposer
		prize += types.MicroAlgos(w.C.Cfg.ADrop.Reward)
	} else {
		keys := reflect.ValueOf(onlPoolsMap).MapKeys()
		if len(keys) == 0 {
			w.Log.Warnf("No online pool data yet, prize is gone.")
			return
		}
		winner = keys[rand.Intn(len(keys))].Interface().(types.Address)
	}
	w.drop(ctx, winner, prize)
}

func (w *ALGODROPWorker) algoDropper(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}
		select {
		case <-ctx.Done():
			close(w.bvChan)
			return
		case bv, ok := <-w.bvChan:
			if !ok {
				close(w.bvChan)
				return
			}
			w.dropAlgo(ctx, bv)
		}
	}
}
