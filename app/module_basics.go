package app

import (
	"cosmossdk.io/x/evidence"
	feegrantmodule "cosmossdk.io/x/feegrant/module"
	"cosmossdk.io/x/upgrade"

	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authzmodule "github.com/cosmos/cosmos-sdk/x/authz/module"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/consensus"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	"github.com/cosmos/cosmos-sdk/x/params"
	paramsclient "github.com/cosmos/cosmos-sdk/x/params/client"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	"github.com/cosmos/cosmos-sdk/x/staking"

	"github.com/cosmos/ibc-go/modules/capability"
	ica "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts"
	ibctransfer "github.com/cosmos/ibc-go/v8/modules/apps/transfer"
	ibc "github.com/cosmos/ibc-go/v8/modules/core"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"

	"github.com/haqq-network/haqq/x/coinomics"
	"github.com/haqq-network/haqq/x/epochs"
	"github.com/haqq-network/haqq/x/erc20"
	erc20client "github.com/haqq-network/haqq/x/erc20/client"
	"github.com/haqq-network/haqq/x/evm"
	"github.com/haqq-network/haqq/x/feemarket"
	"github.com/haqq-network/haqq/x/ibc/transfer"
	"github.com/haqq-network/haqq/x/liquidvesting"
	"github.com/haqq-network/haqq/x/vesting"
)

// ModuleBasics defines the module BasicManager is in charge of setting up basic,
// non-dependant module elements, such as codec registration
// and genesis verification.
var ModuleBasics = module.NewBasicManager(
	auth.AppModuleBasic{},
	genutil.NewAppModuleBasic(genutiltypes.DefaultMessageValidator),
	bank.AppModuleBasic{},
	capability.AppModuleBasic{},
	staking.AppModuleBasic{},
	distr.AppModuleBasic{},
	gov.NewAppModuleBasic(
		[]govclient.ProposalHandler{
			paramsclient.ProposalHandler,
			// Evmos proposal types
			erc20client.RegisterCoinProposalHandler,
			erc20client.RegisterERC20ProposalHandler,
			erc20client.ToggleTokenConversionProposalHandler,
		},
	),
	params.AppModuleBasic{},
	crisis.AppModuleBasic{},
	slashing.AppModuleBasic{},
	ibc.AppModuleBasic{},
	ibctm.AppModuleBasic{},
	ica.AppModuleBasic{},
	authzmodule.AppModuleBasic{},
	feegrantmodule.AppModuleBasic{},
	upgrade.AppModuleBasic{},
	evidence.AppModuleBasic{},
	transfer.AppModuleBasic{AppModuleBasic: &ibctransfer.AppModuleBasic{}},
	vesting.AppModuleBasic{},
	liquidvesting.AppModuleBasic{},
	evm.AppModuleBasic{},
	feemarket.AppModuleBasic{},
	coinomics.AppModuleBasic{},
	erc20.AppModuleBasic{},
	epochs.AppModuleBasic{},
	consensus.AppModuleBasic{},
)
