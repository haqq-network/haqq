package ante_test

import (
	"testing"
	"time"

	//nolint:revive // dot imports are fine for Ginkgo
	. "github.com/onsi/ginkgo/v2"
	//nolint:revive // dot imports are fine for Ginkgo
	. "github.com/onsi/gomega"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/haqq-network/haqq/app"
	"github.com/haqq-network/haqq/crypto/ethsecp256k1"
	"github.com/haqq-network/haqq/encoding"
	"github.com/haqq-network/haqq/testutil"
	"github.com/haqq-network/haqq/utils"
	feemarkettypes "github.com/haqq-network/haqq/x/feemarket/types"
)

var s *AnteTestSuite

type AnteTestSuite struct {
	suite.Suite

	ctx       sdk.Context
	clientCtx client.Context
	app       *app.Haqq
	denom     string
}

func (suite *AnteTestSuite) SetupTest() {
	t := suite.T()
	privCons, err := ethsecp256k1.GenerateKey()
	require.NoError(t, err)
	consAddress := sdk.ConsAddress(privCons.PubKey().Address())

	isCheckTx := false
	chainID := utils.MainNetChainID + "-1"
	suite.app, _ = app.Setup(isCheckTx, feemarkettypes.DefaultGenesisState(), chainID)
	suite.Require().NotNil(suite.app.AppCodec())

	header := testutil.NewHeader(1, time.Now().UTC(), chainID, consAddress, nil, nil)
	suite.ctx = suite.app.BaseApp.NewContext(isCheckTx, header)

	suite.denom = utils.BaseDenom
	evmParams := suite.app.EvmKeeper.GetParams(suite.ctx)
	evmParams.EvmDenom = suite.denom
	_ = suite.app.EvmKeeper.SetParams(suite.ctx, evmParams)

	encodingConfig := encoding.MakeConfig(app.ModuleBasics)
	suite.clientCtx = client.Context{}.WithTxConfig(encodingConfig.TxConfig)
}

func TestAnteTestSuite(t *testing.T) {
	s = new(AnteTestSuite)
	suite.Run(t, s)

	RegisterFailHandler(Fail)
	RunSpecs(t, "Run AnteHandler Integration Tests")
}
