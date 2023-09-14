package ante

import (
	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	ibcante "github.com/cosmos/ibc-go/v7/modules/core/ante"
	ibckeeper "github.com/cosmos/ibc-go/v7/modules/core/keeper"

	evmoscosmosante "github.com/evmos/evmos/v14/app/ante/cosmos"
	evmosevmante "github.com/evmos/evmos/v14/app/ante/evm"
	evmosanteutils "github.com/evmos/evmos/v14/app/ante/utils"
	evmtypes "github.com/evmos/evmos/v14/x/evm/types"

	haqqcosmosante "github.com/haqq-network/haqq/app/ante/cosmos"
	haqqevmante "github.com/haqq-network/haqq/app/ante/evm"
	vestingtypes "github.com/haqq-network/haqq/x/vesting/types"
)

// HandlerOptions defines the list of module keepers required to run the Evmos
// AnteHandler decorators.
type HandlerOptions struct {
	Cdc                    codec.BinaryCodec
	AccountKeeper          evmtypes.AccountKeeper
	BankKeeper             evmtypes.BankKeeper
	DistributionKeeper     evmosanteutils.DistributionKeeper
	IBCKeeper              *ibckeeper.Keeper
	StakingKeeper          vestingtypes.StakingKeeper
	FeeMarketKeeper        evmosevmante.FeeMarketKeeper
	EvmKeeper              evmosevmante.EVMKeeper
	FeegrantKeeper         ante.FeegrantKeeper
	ExtensionOptionChecker ante.ExtensionOptionChecker
	SignModeHandler        authsigning.SignModeHandler
	SigGasConsumer         func(meter sdk.GasMeter, sig signing.SignatureV2, params authtypes.Params) error
	MaxTxGasWanted         uint64
	TxFeeChecker           evmosanteutils.TxFeeChecker
}

// Validate checks if the keepers are defined
func (options HandlerOptions) Validate() error {
	if options.Cdc == nil {
		return errorsmod.Wrap(errortypes.ErrLogic, "codec is required for AnteHandler")
	}
	if options.AccountKeeper == nil {
		return errorsmod.Wrap(errortypes.ErrLogic, "account keeper is required for AnteHandler")
	}
	if options.BankKeeper == nil {
		return errorsmod.Wrap(errortypes.ErrLogic, "bank keeper is required for AnteHandler")
	}
	if options.IBCKeeper == nil {
		return errorsmod.Wrap(errortypes.ErrLogic, "ibc keeper is required for AnteHandler")
	}
	if options.StakingKeeper == nil {
		return errorsmod.Wrap(errortypes.ErrLogic, "staking keeper is required for AnteHandler")
	}
	if options.FeeMarketKeeper == nil {
		return errorsmod.Wrap(errortypes.ErrLogic, "fee market keeper is required for AnteHandler")
	}
	if options.EvmKeeper == nil {
		return errorsmod.Wrap(errortypes.ErrLogic, "evm keeper is required for AnteHandler")
	}
	if options.SigGasConsumer == nil {
		return errorsmod.Wrap(errortypes.ErrLogic, "signature gas consumer is required for AnteHandler")
	}
	if options.SignModeHandler == nil {
		return errorsmod.Wrap(errortypes.ErrLogic, "sign mode handler is required for AnteHandler")
	}
	if options.DistributionKeeper == nil {
		return errorsmod.Wrap(errortypes.ErrLogic, "distribution keeper is required for AnteHandler")
	}
	if options.TxFeeChecker == nil {
		return errorsmod.Wrap(errortypes.ErrLogic, "tx fee checker is required for AnteHandler")
	}
	return nil
}

// newCosmosAnteHandler creates the default ante handler for Ethereum transactions
func newEthAnteHandler(options HandlerOptions) sdk.AnteHandler {
	return sdk.ChainAnteDecorators(
		// outermost AnteDecorator. SetUpContext must be called first
		evmosevmante.NewEthSetUpContextDecorator(options.EvmKeeper),
		// Check eth effective gas price against the node's minimal-gas-prices config
		evmosevmante.NewEthMempoolFeeDecorator(options.EvmKeeper),
		// Check eth effective gas price against the global MinGasPrice
		evmosevmante.NewEthMinGasPriceDecorator(options.FeeMarketKeeper, options.EvmKeeper),
		evmosevmante.NewEthValidateBasicDecorator(options.EvmKeeper),
		evmosevmante.NewEthSigVerificationDecorator(options.EvmKeeper),
		evmosevmante.NewEthAccountVerificationDecorator(options.AccountKeeper, options.EvmKeeper),
		evmosevmante.NewCanTransferDecorator(options.EvmKeeper),
		haqqevmante.NewEthVestingTransactionDecorator(options.AccountKeeper, options.BankKeeper, options.EvmKeeper),
		evmosevmante.NewEthGasConsumeDecorator(options.BankKeeper, options.DistributionKeeper, options.EvmKeeper, options.StakingKeeper, options.MaxTxGasWanted),
		evmosevmante.NewEthIncrementSenderSequenceDecorator(options.AccountKeeper),
		evmosevmante.NewGasWantedDecorator(options.EvmKeeper, options.FeeMarketKeeper),
		// emit eth tx hash and index at the very last ante handler.
		evmosevmante.NewEthEmitEventDecorator(options.EvmKeeper),
	)
}

// newCosmosAnteHandler creates the default ante handler for Cosmos transactions
func newCosmosAnteHandler(options HandlerOptions) sdk.AnteHandler {
	return sdk.ChainAnteDecorators(
		evmoscosmosante.RejectMessagesDecorator{}, // reject MsgEthereumTxs
		evmoscosmosante.NewAuthzLimiterDecorator( // disable the Msg types that cannot be included on an authz.MsgExec msgs field
			sdk.MsgTypeURL(&evmtypes.MsgEthereumTx{}),
			sdk.MsgTypeURL(&sdkvesting.MsgCreateVestingAccount{}),
		),
		ante.NewSetUpContextDecorator(),
		ante.NewExtensionOptionsDecorator(options.ExtensionOptionChecker),
		ante.NewValidateBasicDecorator(),
		ante.NewTxTimeoutHeightDecorator(),
		ante.NewValidateMemoDecorator(options.AccountKeeper),
		evmoscosmosante.NewMinGasPriceDecorator(options.FeeMarketKeeper, options.EvmKeeper),
		ante.NewConsumeGasForTxSizeDecorator(options.AccountKeeper),
		evmoscosmosante.NewDeductFeeDecorator(options.AccountKeeper, options.BankKeeper, options.DistributionKeeper, options.FeegrantKeeper, options.StakingKeeper, options.TxFeeChecker),
		haqqcosmosante.NewVestingDelegationDecorator(options.AccountKeeper, options.StakingKeeper, options.BankKeeper, options.Cdc),
		// SetPubKeyDecorator must be called before all signature verification decorators
		ante.NewSetPubKeyDecorator(options.AccountKeeper),
		ante.NewValidateSigCountDecorator(options.AccountKeeper),
		ante.NewSigGasConsumeDecorator(options.AccountKeeper, options.SigGasConsumer),
		ante.NewSigVerificationDecorator(options.AccountKeeper, options.SignModeHandler),
		ante.NewIncrementSequenceDecorator(options.AccountKeeper),
		ibcante.NewRedundantRelayDecorator(options.IBCKeeper),
		evmosevmante.NewGasWantedDecorator(options.EvmKeeper, options.FeeMarketKeeper),
	)
}

// newCosmosAnteHandlerEip712 creates the ante handler for transactions signed with EIP712
func newLegacyCosmosAnteHandlerEip712(options HandlerOptions) sdk.AnteHandler {
	return sdk.ChainAnteDecorators(
		evmoscosmosante.RejectMessagesDecorator{}, // reject MsgEthereumTxs
		evmoscosmosante.NewAuthzLimiterDecorator( // disable the Msg types that cannot be included on an authz.MsgExec msgs field
			sdk.MsgTypeURL(&evmtypes.MsgEthereumTx{}),
			sdk.MsgTypeURL(&sdkvesting.MsgCreateVestingAccount{}),
		),
		ante.NewSetUpContextDecorator(),
		ante.NewValidateBasicDecorator(),
		ante.NewTxTimeoutHeightDecorator(),
		evmoscosmosante.NewMinGasPriceDecorator(options.FeeMarketKeeper, options.EvmKeeper),
		ante.NewValidateMemoDecorator(options.AccountKeeper),
		ante.NewConsumeGasForTxSizeDecorator(options.AccountKeeper),
		evmoscosmosante.NewDeductFeeDecorator(options.AccountKeeper, options.BankKeeper, options.DistributionKeeper, options.FeegrantKeeper, options.StakingKeeper, options.TxFeeChecker),
		haqqcosmosante.NewVestingDelegationDecorator(options.AccountKeeper, options.StakingKeeper, options.BankKeeper, options.Cdc),
		// SetPubKeyDecorator must be called before all signature verification decorators
		ante.NewSetPubKeyDecorator(options.AccountKeeper),
		ante.NewValidateSigCountDecorator(options.AccountKeeper),
		ante.NewSigGasConsumeDecorator(options.AccountKeeper, options.SigGasConsumer),
		// Note: signature verification uses EIP instead of the cosmos signature validator
		//nolint: staticcheck
		evmoscosmosante.NewLegacyEip712SigVerificationDecorator(options.AccountKeeper, options.SignModeHandler),
		ante.NewIncrementSequenceDecorator(options.AccountKeeper),
		ibcante.NewRedundantRelayDecorator(options.IBCKeeper),
		evmosevmante.NewGasWantedDecorator(options.EvmKeeper, options.FeeMarketKeeper),
	)
}
