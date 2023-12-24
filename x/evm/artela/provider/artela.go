package provider

import (
	"context"
	"errors"
	"math/big"

	"google.golang.org/protobuf/types/known/anypb"

	asptypes "github.com/artela-network/aspect-core/types"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/aspect/cosmos"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	ethereum "github.com/ethereum/go-ethereum/core/types"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/artela-network/artela/x/evm/artela/contract"
	"github.com/artela-network/artela/x/evm/artela/types"
	statedb "github.com/artela-network/artela/x/evm/states"
)

var _ cosmos.AspectCosmosProvider = (*ArtelaProvider)(nil)

type ArtelaProvider struct {
	service  *contract.AspectService
	storeKey storetypes.StoreKey
}

func NewArtelaProvider(storeKey storetypes.StoreKey,
	getBlockHeight types.GetLastBlockHeight,
	logger log.Logger,
) *ArtelaProvider {
	service := contract.NewAspectService(storeKey, getBlockHeight, logger)

	return &ArtelaProvider{service, storeKey}
}

func (j *ArtelaProvider) TxToPointRequest(sdkCtx sdk.Context, from common.Address, transaction *ethereum.Transaction, txIndex int64, baseFee *big.Int, innerTx *asptypes.EthStackTransaction) (*asptypes.EthTxAspect, error) {
	ethTransaction, err := asptypes.NewEthTransaction(from, transaction, common.BytesToHash(sdkCtx.HeaderHash().Bytes()), sdkCtx.BlockHeight(), txIndex, baseFee, sdkCtx.ChainID())
	if err != nil {
		return nil, err
	}
	return &asptypes.EthTxAspect{
		Tx:          ethTransaction,
		CurrInnerTx: innerTx,
		GasInfo:     &asptypes.GasInfo{},
	}, nil
}

func (j *ArtelaProvider) CreateTxPointRequest(sdkCtx sdk.Context, msg sdk.Msg, txIndex int64, baseFee *big.Int, innerTx *asptypes.EthStackTransaction) (*asptypes.EthTxAspect, error) {
	ethMsg := types.ConvertMsgEthereumTx(msg)
	tx := ethMsg.AsTransaction()

	// Determine the signer. For replay-protected transactions, use the most permissive
	// signer, because we assume that signers are backwards-compatible with old
	// transactions. For non-protected transactions, the homestead signer signer is used
	// because the return value of ChainId is zero for those transactions.
	var signer ethtypes.Signer
	if tx.Protected() {
		signer = ethtypes.LatestSignerForChainID(tx.ChainId())
	} else {
		signer = ethtypes.HomesteadSigner{}
	}
	from, err := ethtypes.Sender(signer, tx)
	if err != nil {
		return nil, err
	}

	ethTransaction, err := asptypes.NewEthTransaction(from, tx, common.BytesToHash(sdkCtx.HeaderHash().Bytes()), sdkCtx.BlockHeight(), txIndex, baseFee, sdkCtx.ChainID())
	if err != nil {
		return nil, err
	}
	return &asptypes.EthTxAspect{
		Tx:          ethTransaction,
		CurrInnerTx: innerTx,
		GasInfo:     &asptypes.GasInfo{},
	}, nil
}

func (j *ArtelaProvider) CreateTxPointRequestWithData(data []byte) (*asptypes.EthTxAspect, error) {
	anyData, err := anypb.New(&asptypes.BytesData{Data: data})
	if err != nil {
		return nil, err
	}
	return &asptypes.EthTxAspect{
		Tx:          nil,
		CurrInnerTx: nil,
		CallData:    anyData,
		GasInfo:     &asptypes.GasInfo{},
	}, nil
}

func (j *ArtelaProvider) CreateBlockPointRequest(sdkCtx sdk.Context) *asptypes.EthBlockAspect {
	header := types.ConvertEthBlockHeader(sdkCtx.BlockHeader())
	return &asptypes.EthBlockAspect{Header: header, GasInfo: &asptypes.GasInfo{
		GasWanted: 0,
		GasUsed:   0,
		Gas:       0,
	}}
}

func (j *ArtelaProvider) CreateTxPointRequestInEvm(sdkCtx sdk.Context, msg *core.Message, txConfig statedb.TxConfig, innerTx *asptypes.EthStackTransaction) *asptypes.EthTxAspect {
	chainId := sdkCtx.ChainID()
	blockHash := common.BytesToHash(sdkCtx.HeaderHash().Bytes())
	blockHeight := sdkCtx.BlockHeight()
	ethTx := asptypes.NewEthTransactionByMessage(msg, txConfig.TxHash, chainId, blockHash, blockHeight, uint8(txConfig.TxType))
	return &asptypes.EthTxAspect{
		Tx:          ethTx,
		CurrInnerTx: innerTx,
		GasInfo: &asptypes.GasInfo{
			GasWanted: 0,
			GasUsed:   0,
			Gas:       0,
		},
	}
}

func (ArtelaProvider) FilterAspectTx(tx sdk.Msg) bool {
	if tx.ValidateBasic() != nil {
		return false
	}
	isEthTx := types.IsEthTx(tx)
	if !isEthTx {
		return false
	}
	ethTx := types.ConvertEthTx(tx)
	if ethTx == nil || ethTx.To() == nil || asptypes.IsAspectContractAddr(ethTx.To()) {
		return false
	}

	return true
}

func (j *ArtelaProvider) GetTxBondAspects(ctx context.Context, address common.Address, point asptypes.PointCut) ([]*asptypes.AspectCode, error) {
	if ctx == nil {
		return nil, errors.New("invalid Context")
	}
	aspectCtx, ok := ctx.(*types.AspectRuntimeContext)
	if !ok {
		return nil, errors.New("failed to unwrap AspectRuntimeContext from context.Context")
	}
	return j.service.GetAspectsForJoinPoint(aspectCtx.CosmosContext(), address, point)
}

func (j *ArtelaProvider) GetAccountVerifiers(ctx context.Context, address common.Address) ([]*asptypes.AspectCode, error) {
	if ctx == nil {
		return nil, errors.New("invalid Context")
	}
	aspectCtx, ok := ctx.(*types.AspectRuntimeContext)
	if !ok {
		return nil, errors.New("failed to unwrap AspectRuntimeContext from context.Context")
	}
	return j.service.GetAccountVerifiers(aspectCtx.CosmosContext(), address)
}

func (j *ArtelaProvider) GetBlockBondAspects(ctx context.Context) ([]*asptypes.AspectCode, error) {
	if ctx == nil {
		return nil, errors.New("invalid Context")
	}
	aspectCtx, ok := ctx.(*types.AspectRuntimeContext)
	if !ok {
		return nil, errors.New("failed to unwrap AspectRuntimeContext from context.Context")
	}
	return j.service.GetAspectForBlock(aspectCtx.CosmosContext())
}

func (j *ArtelaProvider) GetAspectAccount(ctx context.Context, aspectId common.Address) (*common.Address, error) {
	if ctx == nil {
		return nil, errors.New("invalid Context")
	}
	aspectCtx, ok := ctx.(*types.AspectRuntimeContext)
	if !ok {
		return nil, errors.New("failed to unwrap AspectRuntimeContext from context.Context")
	}
	return j.service.GetAspectAccount(aspectCtx.CosmosContext(), aspectId)
}

func (j *ArtelaProvider) GetLatestBlock() int64 {
	return j.service.GetBlockHeight()
}
