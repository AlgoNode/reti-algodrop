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

package algodapi

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/algonode/reti-algodrop/internal/config"
	"github.com/algorand/go-algorand-sdk/v2/client/v2/algod"
	"github.com/sirupsen/logrus"
	"github.com/ssgreg/repeat"
	"go.uber.org/ratelimit"
)

type AlgodAPI struct {
	cfg    *config.NodeConfig
	log    *logrus.Logger
	Client *algod.Client
	rl     ratelimit.Limiter
}

func Make(ctx context.Context, acfg *config.NodeConfig, log *logrus.Logger) (*AlgodAPI, error) {

	// Create an algod client
	algodClient, err := algod.MakeClient(acfg.Address, acfg.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to make algod client: %s\n", err)
	}

	return &AlgodAPI{
		cfg:    acfg,
		log:    log,
		Client: algodClient,
		rl:     ratelimit.New(acfg.RateLimit, ratelimit.WithoutSlack),
	}, nil

}

func backoff(ctx context.Context, limit int) repeat.Operation {
	return repeat.Compose(
		repeat.StopOnSuccess(),
		repeat.LimitMaxTries(limit),
		repeat.WithDelay(
			repeat.FullJitterBackoff(50*time.Millisecond).Set(),
			repeat.SetContext(ctx),
		),
	)
}

func backoffMe(ctx context.Context, limit int, op func() error) error {
	return repeat.Repeat(repeat.Fn(op), backoff(ctx, limit))
}

func extractWrappedHttpStatus(err error) int {
	str := err.Error()
	if strings.HasPrefix(str, "HTTP ") {
		statusCode, err := strconv.Atoi(str[5:])
		if err != nil {
			return -1
		}
		return statusCode
	}
	return -1
}

func isFastFailStatus(status int) bool {
	return status >= 400 && status <= 423 && status != 408
}

func httpFastFail(err error) error {
	if err != nil {
		status := extractWrappedHttpStatus(err)
		if isFastFailStatus(status) {
			return repeat.HintStop(err)
		}
		return repeat.HintTemporary(err)
	}
	return nil
}

func httpFastFailExcept404(err error) error {
	if err != nil {
		status := extractWrappedHttpStatus(err)
		if isFastFailStatus(status) && status != 404 {
			return repeat.HintStop(err)
		}
		return repeat.HintTemporary(err)
	}
	return nil
}

func (api *AlgodAPI) RateLimit() *algod.Client {
	api.rl.Take()
	return api.Client
}
