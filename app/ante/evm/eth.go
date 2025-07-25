package evm

import (
	"math"
	"math/big"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"
	anteutils "github.com/haqq-network/haqq/app/ante/utils"
	"github.com/haqq-network/haqq/types"
	"github.com/haqq-network/haqq/x/evm/keeper"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

// EthGasConsumeDecorator validates enough intrinsic gas for the transaction and
// gas consumption.
type EthGasConsumeDecorator struct {
	bankKeeper         anteutils.BankKeeper
	distributionKeeper anteutils.DistributionKeeper
	evmKeeper          EVMKeeper
	stakingKeeper      anteutils.StakingKeeper
	maxGasWanted       uint64
}

// NewEthGasConsumeDecorator creates a new EthGasConsumeDecorator
func NewEthGasConsumeDecorator(
	bankKeeper anteutils.BankKeeper,
	distributionKeeper anteutils.DistributionKeeper,
	evmKeeper EVMKeeper,
	stakingKeeper anteutils.StakingKeeper,
	maxGasWanted uint64,
) EthGasConsumeDecorator {
	return EthGasConsumeDecorator{
		bankKeeper,
		distributionKeeper,
		evmKeeper,
		stakingKeeper,
		maxGasWanted,
	}
}

// AnteHandle validates that the Ethereum tx message has enough to cover intrinsic gas
// (during CheckTx only) and that the sender has enough balance to pay for the gas cost.
// If the balance is not sufficient, it will be attempted to withdraw enough staking rewards
// for the payment.
//
// Intrinsic gas for a transaction is the amount of gas that the transaction uses before the
// transaction is executed. The gas is a constant value plus any cost incurred by additional bytes
// of data supplied with the transaction.
//
// This AnteHandler decorator will fail if:
// - the message is not a MsgEthereumTx
// - sender account cannot be found
// - transaction's gas limit is lower than the intrinsic gas
// - user has neither enough balance nor staking rewards to deduct the transaction fees (gas_limit * gas_price)
// - transaction or block gas meter runs out of gas
// - sets the gas meter limit
// - gas limit is greater than the block gas meter limit
func (egcd EthGasConsumeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	gasWanted := uint64(0)
	// gas consumption limit already checked during CheckTx so there's no need to
	// verify it again during ReCheckTx
	if ctx.IsReCheckTx() {
		// Use new context with gasWanted = 0
		// Otherwise, there's an error on txmempool.postCheck (tendermint)
		// that is not bubbled up. Thus, the Tx never runs on DeliverMode
		// Error: "gas wanted -1 is negative"
		// For more information, see issue #1554
		// https://github.com/evmos/ethermint/issues/1554
		newCtx := ctx.WithGasMeter(types.NewInfiniteGasMeterWithLimit(gasWanted))
		return next(newCtx, tx, simulate)
	}

	evmParams := egcd.evmKeeper.GetParams(ctx)
	evmDenom := evmParams.GetEvmDenom()
	chainCfg := evmParams.GetChainConfig()
	ethCfg := chainCfg.EthereumConfig(egcd.evmKeeper.ChainID())

	blockHeight := big.NewInt(ctx.BlockHeight())
	homestead := ethCfg.IsHomestead(blockHeight)
	istanbul := ethCfg.IsIstanbul(blockHeight)
	var events sdk.Events

	// Use the lowest priority of all the messages as the final one.
	minPriority := int64(math.MaxInt64)
	baseFee := egcd.evmKeeper.GetBaseFee(ctx, ethCfg)

	for _, msg := range tx.GetMsgs() {
		msgEthTx, ok := msg.(*evmtypes.MsgEthereumTx)
		if !ok {
			return ctx, errorsmod.Wrapf(errortypes.ErrUnknownRequest, "invalid message type %T, expected %T", msg, (*evmtypes.MsgEthereumTx)(nil))
		}
		from := msgEthTx.GetFrom()

		txData, err := evmtypes.UnpackTxData(msgEthTx.Data)
		if err != nil {
			return ctx, errorsmod.Wrap(err, "failed to unpack tx data")
		}

		if ctx.IsCheckTx() && egcd.maxGasWanted != 0 {
			// We can't trust the tx gas limit, because we'll refund the unused gas.
			if txData.GetGas() > egcd.maxGasWanted {
				gasWanted += egcd.maxGasWanted
			} else {
				gasWanted += txData.GetGas()
			}
		} else {
			gasWanted += txData.GetGas()
		}

		fees, err := keeper.VerifyFee(txData, evmDenom, baseFee, homestead, istanbul, ctx.IsCheckTx())
		if err != nil {
			return ctx, errorsmod.Wrapf(err, "failed to verify the fees")
		}

		if err = egcd.deductFee(ctx, fees, from); err != nil {
			return ctx, err
		}

		events = append(events,
			sdk.NewEvent(
				sdk.EventTypeTx,
				sdk.NewAttribute(sdk.AttributeKeyFee, fees.String()),
			),
		)

		priority := evmtypes.GetTxPriority(txData, baseFee)

		if priority < minPriority {
			minPriority = priority
		}
	}

	ctx.EventManager().EmitEvents(events)

	blockGasLimit := types.BlockGasLimit(ctx)

	// return error if the tx gas is greater than the block limit (max gas)

	// NOTE: it's important here to use the gas wanted instead of the gas consumed
	// from the tx gas pool. The latter only has the value so far since the
	// EthSetupContextDecorator, so it will never exceed the block gas limit.
	if gasWanted > blockGasLimit {
		return ctx, errorsmod.Wrapf(
			errortypes.ErrOutOfGas,
			"tx gas (%d) exceeds block gas limit (%d)",
			gasWanted,
			blockGasLimit,
		)
	}

	// Set tx GasMeter with a limit of GasWanted (i.e. gas limit from the Ethereum tx).
	// The gas consumed will be then reset to the gas used by the state transition
	// in the EVM.

	// FIXME: use a custom gas configuration that doesn't add any additional gas and only
	// takes into account the gas consumed at the end of the EVM transaction.
	newCtx := ctx.
		WithGasMeter(types.NewInfiniteGasMeterWithLimit(gasWanted)).
		WithPriority(minPriority)

	// we know that we have enough gas on the pool to cover the intrinsic gas
	return next(newCtx, tx, simulate)
}

// deductFee checks if the fee payer has enough funds to pay for the fees and deducts them.
// If the spendable balance is not enough, it tries to claim enough staking rewards to cover the fees.
func (egcd EthGasConsumeDecorator) deductFee(ctx sdk.Context, fees sdk.Coins, feePayer sdk.AccAddress) error {
	if fees.IsZero() {
		return nil
	}

	// If the account balance is not sufficient, try to withdraw enough staking rewards
	if err := anteutils.ClaimStakingRewardsIfNecessary(ctx, egcd.bankKeeper, egcd.distributionKeeper, egcd.stakingKeeper, feePayer, fees); err != nil {
		return err
	}

	if err := egcd.evmKeeper.DeductTxCostsFromUserBalance(ctx, fees, common.BytesToAddress(feePayer)); err != nil {
		return errorsmod.Wrapf(err, "failed to deduct transaction costs from user balance")
	}
	return nil
}

// EthIncrementSenderSequenceDecorator increments the sequence of the signers.
type EthIncrementSenderSequenceDecorator struct {
	ak evmtypes.AccountKeeper
}

// NewEthIncrementSenderSequenceDecorator creates a new EthIncrementSenderSequenceDecorator.
func NewEthIncrementSenderSequenceDecorator(ak evmtypes.AccountKeeper) EthIncrementSenderSequenceDecorator {
	return EthIncrementSenderSequenceDecorator{
		ak: ak,
	}
}

// AnteHandle handles incrementing the sequence of the signer (i.e. sender). If the transaction is a
// contract creation, the nonce will be incremented during the transaction execution and not within
// this AnteHandler decorator.
func (issd EthIncrementSenderSequenceDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	for _, msg := range tx.GetMsgs() {
		msgEthTx, ok := msg.(*evmtypes.MsgEthereumTx)
		if !ok {
			return ctx, errorsmod.Wrapf(errortypes.ErrUnknownRequest, "invalid message type %T, expected %T", msg, (*evmtypes.MsgEthereumTx)(nil))
		}

		txData, err := evmtypes.UnpackTxData(msgEthTx.Data)
		if err != nil {
			return ctx, errorsmod.Wrap(err, "failed to unpack tx data")
		}

		// increase sequence of sender
		acc := issd.ak.GetAccount(ctx, msgEthTx.GetFrom())
		if acc == nil {
			return ctx, errorsmod.Wrapf(
				errortypes.ErrUnknownAddress,
				"account %s is nil", common.BytesToAddress(msgEthTx.GetFrom().Bytes()),
			)
		}
		nonce := acc.GetSequence()

		// we merged the nonce verification to nonce increment, so when tx includes multiple messages
		// with same sender, they'll be accepted.
		if txData.GetNonce() != nonce {
			return ctx, errorsmod.Wrapf(
				errortypes.ErrInvalidSequence,
				"invalid nonce; got %d, expected %d", txData.GetNonce(), nonce,
			)
		}

		if err := acc.SetSequence(nonce + 1); err != nil {
			return ctx, errorsmod.Wrapf(err, "failed to set sequence to %d", acc.GetSequence()+1)
		}

		issd.ak.SetAccount(ctx, acc)
	}

	return next(ctx, tx, simulate)
}
