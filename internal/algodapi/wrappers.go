package algodapi

import (
	"context"

	"github.com/algorand/go-algorand-sdk/v2/client/v2/common/models"
	"github.com/algorand/go-algorand-sdk/v2/encoding/msgpack"
	"github.com/algorand/go-algorand-sdk/v2/types"
	"github.com/ssgreg/repeat"
)

func (api *AlgodAPI) SuggestedParams(ctx context.Context) (params types.SuggestedParams, err error) {
	err = backoffMe(ctx, 10, func() error {
		var terr error
		params, terr = api.RateLimit().SuggestedParams().Do(ctx)
		return httpFastFail(terr)
	})
	return
}

func (api *AlgodAPI) Status(ctx context.Context) (status models.NodeStatus, err error) {
	err = backoffMe(ctx, 10, func() error {
		var terr error
		status, terr = api.RateLimit().Status().Do(ctx)
		return httpFastFail(terr)
	})
	return
}

func (api *AlgodAPI) WaitForRoundAfter(ctx context.Context, round uint64) (status models.NodeStatus, err error) {
	err = backoffMe(ctx, 10, func() error {
		var terr error
		status, terr = api.RateLimit().StatusAfterBlock(round).Do(ctx)
		return httpFastFailExcept404(terr)
	})
	return
}

func (api *AlgodAPI) GetBlockWithCert(ctx context.Context, round uint64) (*models.BlockResponse, error) {
	var block models.BlockResponse

	err := backoffMe(ctx, 10, func() error {
		s, terr := api.RateLimit().BlockRaw(round).Do(ctx)
		if terr != nil {
			return httpFastFailExcept404(terr)
		}
		terr = msgpack.Decode(s, &block)
		if terr != nil {
			api.log.WithError(terr).Errorf("error decoding block for round %d", round)
			return repeat.HintStop(terr)
		}
		return nil
	})
	return &block, err
}

func (api *AlgodAPI) GetApp(ctx context.Context, appId uint64) (app models.Application, err error) {
	err = backoffMe(ctx, 10, func() error {
		var terr error
		app, terr = api.RateLimit().GetApplicationByID(appId).Do(ctx)
		return httpFastFail(terr)
	})
	return
}

func (api *AlgodAPI) GetAppBox(ctx context.Context, appId uint64, box []byte) (resp models.Box, err error) {
	err = backoffMe(ctx, 10, func() error {
		var terr error
		resp, terr = api.RateLimit().GetApplicationBoxByName(appId, box).Do(ctx)
		return httpFastFail(terr)
	})
	return
}
