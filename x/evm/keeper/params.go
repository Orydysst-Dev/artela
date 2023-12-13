package keeper

import (
	"fmt"

	"github.com/artela-network/artela/x/evm/txs/support"
	"github.com/artela-network/artela/x/evm/types"
	cosmos "github.com/cosmos/cosmos-sdk/types"
	ethlog "github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

// GetParams returns the total set of evm parameters.
func (k Keeper) GetParams(ctx cosmos.Context) (params support.Params) {
	store := ctx.KVStore(k.storeKey)

	bz := store.Get(types.KeyPrefixParams)
	if len(bz) == 0 {
		return k.GetLegacyParams(ctx)
	}

	k.cdc.MustUnmarshal(bz, &params)
	return
}

func (k *Keeper) GetChainConfig(ctx cosmos.Context) *params.ChainConfig {
	params := k.GetParams(ctx)
	if k.ChainID() == nil {
		k.WithChainID(ctx)
	}
	ethCfg := params.ChainConfig.EthereumConfig(k.ChainID())
	return ethCfg
}

// SetParams sets the EVM params each in their individual key for better get performance
func (k Keeper) SetParams(ctx cosmos.Context, params support.Params) error {
	if err := params.Validate(); err != nil {
		return err
	}

	store := ctx.KVStore(k.storeKey)
	bz, err := k.cdc.Marshal(&params)
	if err != nil {
		return err
	}

	ethlog.Info("SetParams, key:", "KeyPrefixParams", "value:", fmt.Sprintf("%+v", params))
	store.Set(types.KeyPrefixParams, bz)
	return nil
}

// GetLegacyParams returns param set for version before migrate
func (k Keeper) GetLegacyParams(ctx cosmos.Context) support.Params {
	var params support.Params
	k.ss.GetParamSetIfExists(ctx, &params)
	return params
}
