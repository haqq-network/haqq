package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/haqq-network/haqq/utils"
)

func (suite *KeeperTestSuite) TestInitGenesis() {
	// check calculated epochMintProvision at genesis
	epochMintProvision := suite.app.CoinomicsKeeper.GetEraTargetMint(suite.ctx)
	expMintProvision := sdk.NewCoin(utils.BaseDenom, sdk.NewInt(0))
	suite.Require().Equal(expMintProvision, epochMintProvision)
}
