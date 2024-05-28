package worker

import (
	"bytes"
	"context"
	"encoding/binary"

	"github.com/algorand/go-algorand-sdk/v2/types"
	"github.com/mixcode/binarystruct"
)

const (
	RetiAppId         = 673404372
	RetiKeyValidators = "bnVtVg=="
	RetiKeyStaked     = "c3Rha2Vk"
	RetiKeyStakers    = "bnVtU3Rha2Vycw=="
)

const MAX_NODES = 8
const MAX_POOLS_PER_NODE = 3
const MAX_POOLS = MAX_NODES * MAX_POOLS_PER_NODE

const MIN_PAYOUT_MINS = 1
const MAX_PAYOUT_MINS = 10080
const MAX_POOLS_PER_STAKER = 6

type RetiInfo struct {
	Validators uint64
	Staked     types.MicroAlgos
	Stakers    uint64
}

type ValidatorInfo struct {
	Config              ValidatorConfig
	State               ValidatorCurState
	Pools               [24]PoolInfo
	TokenPayoutRatio    PoolTokenPayoutRatio
	NodePoolAssignments NodePoolAssignmentConfig
}

type ValidatorConfig struct {
	ID                         uint64    `binary:"uint64"`
	Owner                      [32]byte  `binary:"[]uint8"`
	Manager                    [32]byte  `binary:"[]uint8"`
	NFDForInfo                 uint64    `binary:"uint64"`
	EntryGatingType            uint8     `binary:"uint8"`
	EntryGatingAddress         [32]byte  `binary:"[]uint8"`
	EntryGatingAssets          [4]uint64 `binary:"[]uint64"`
	GatingAssetMinBalance      uint64    `binary:"uint64"`
	RewardTokenID              uint64    `binary:"uint64"`
	RewardPerPayout            uint64    `binary:"uint64"`
	EpochRoundLength           uint32    `binary:"uint32"`
	PercentToValidator         uint32    `binary:"uint32"`
	ValidatorCommissionAddress [32]byte  `binary:"[]uint8"`
	MinEntryStake              uint64    `binary:"uint64"`
	MaxAlgoPerPool             uint64    `binary:"uint64"`
	PoolsPerNode               uint8     `binary:"uint8"`
	SunsettingOn               uint64    `binary:"uint64"`
	SunsettingTo               uint64    `binary:"uint64"`
}

type ValidatorCurState struct {
	NumPools uint16 `binary:"uint16"`
	//why 64bit ?
	TotalStakers        uint64 `binary:"uint64"`
	TotalAlgoStaked     uint64 `binary:"uint64"`
	RewardTokenHeldBack uint64 `binary:"uint64"`
}

type PoolInfo struct {
	PoolAppId       uint64 `binary:"uint64"`
	TotalStakers    uint16 `binary:"uint16"`
	TotalAlgoStaked uint64 `binary:"uint64"`
}

type NodeConfig struct {
	PoolAppIds [3]uint64 `binary:"[]uint64"`
}

type NodePoolAssignmentConfig struct {
	Nodes [8]NodeConfig
}

type PoolTokenPayoutRatio struct {
	PoolPctOfWhole   [24]uint64 `binary:"[]uint64"`
	UpdatedForPayout uint64     `binary:"uint64"`
}

func getValidatorListBoxName(id uint64) []byte {
	prefix := []byte("v")
	ibytes := make([]byte, 8)
	binary.BigEndian.PutUint64(ibytes, id)
	return bytes.Join([][]byte{prefix, ibytes[:]}, nil)
}

func (w *ALGODROPWorker) getRetiValidatorInfo(ctx context.Context, vid uint64) (*ValidatorInfo, error) {
	var vi ValidatorInfo
	box, err := w.C.Aapi.GetAppBox(ctx, RetiAppId, getValidatorListBoxName(vid))
	if err != nil {
		w.Log.WithError(err).Error("error getting info for validator %d", vid)
		return nil, err
	}
	binarystruct.Unmarshal(box.Value, binarystruct.BigEndian, &vi)
	return &vi, nil
}
