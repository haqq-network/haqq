package keeper_test

import (
	"errors"
	"fmt"
	"math/big"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	ibcgotesting "github.com/cosmos/ibc-go/v7/testing"
	ibcmock "github.com/cosmos/ibc-go/v7/testing/mock"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/contracts"
	"github.com/haqq-network/haqq/crypto/ethsecp256k1"
	"github.com/haqq-network/haqq/testutil"
	"github.com/haqq-network/haqq/utils"
	"github.com/haqq-network/haqq/x/erc20/keeper"
	"github.com/haqq-network/haqq/x/erc20/types"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

var erc20Denom = "erc20/0xdac17f958d2ee523a2206206994597c13d831ec7"

func (suite *KeeperTestSuite) TestOnRecvPacket() {
	// secp256k1 account
	secpPk := secp256k1.GenPrivKey()
	secpAddr := sdk.AccAddress(secpPk.PubKey().Address())
	secpAddrCosmos := sdk.MustBech32ifyAddressBytes(sdk.Bech32MainPrefix, secpAddr)

	// ethsecp256k1 account
	ethPk, err := ethsecp256k1.GenerateKey()
	suite.Require().Nil(err)
	ethsecpAddr := sdk.AccAddress(ethPk.PubKey().Address())
	ethsecpAddrHaqq := sdk.AccAddress(ethPk.PubKey().Address()).String()
	ethsecpAddrCosmos := sdk.MustBech32ifyAddressBytes(sdk.Bech32MainPrefix, ethsecpAddr)

	// Setup Cosmos <=> Haqq IBC relayer
	sourceChannel := "channel-292"
	haqqChannel := "channel-0"
	path := fmt.Sprintf("%s/%s", transfertypes.PortID, haqqChannel)

	timeoutHeight := clienttypes.NewHeight(0, 100)
	disabledTimeoutTimestamp := uint64(0)
	mockPacket := channeltypes.NewPacket(ibcgotesting.MockPacketData, 1, transfertypes.PortID, "channel-0", transfertypes.PortID, "channel-0", timeoutHeight, disabledTimeoutTimestamp)
	packet := mockPacket
	expAck := ibcmock.MockAcknowledgement

	registeredDenom := cosmosTokenBase
	coins := sdk.NewCoins(
		sdk.NewCoin(utils.BaseDenom, math.NewInt(1000)),
		sdk.NewCoin(registeredDenom, math.NewInt(1000)), // some ERC20 token
		sdk.NewCoin(ibcBase, math.NewInt(1000)),         // some IBC coin with a registered token pair
	)

	testCases := []struct {
		name             string
		malleate         func()
		ackSuccess       bool
		receiver         sdk.AccAddress
		expErc20s        *big.Int
		expCoins         sdk.Coins
		checkBalances    bool
		disableERC20     bool
		disableTokenPair bool
	}{
		{
			name: "error - non ics-20 packet",
			malleate: func() {
				packet = mockPacket
			},
			receiver:      secpAddr,
			ackSuccess:    false,
			checkBalances: false,
			expErc20s:     big.NewInt(0),
			expCoins:      coins,
		},
		{
			name: "no-op - erc20 module param disabled",
			malleate: func() {
				transfer := transfertypes.NewFungibleTokenPacketData(registeredDenom, "100", ethsecpAddrHaqq, ethsecpAddrCosmos, "")
				bz := transfertypes.ModuleCdc.MustMarshalJSON(&transfer)
				packet = channeltypes.NewPacket(bz, 1, transfertypes.PortID, sourceChannel, transfertypes.PortID, haqqChannel, timeoutHeight, 0)
			},
			receiver:      secpAddr,
			disableERC20:  true,
			ackSuccess:    true,
			checkBalances: false,
			expErc20s:     big.NewInt(0),
			expCoins:      coins,
		},
		{
			name: "error - invalid sender (no '1')",
			malleate: func() {
				transfer := transfertypes.NewFungibleTokenPacketData(registeredDenom, "100", "ISLM", ethsecpAddrCosmos, "")
				bz := transfertypes.ModuleCdc.MustMarshalJSON(&transfer)
				packet = channeltypes.NewPacket(bz, 100, transfertypes.PortID, sourceChannel, transfertypes.PortID, haqqChannel, timeoutHeight, 0)
			},
			receiver:      secpAddr,
			ackSuccess:    false,
			checkBalances: false,
			expErc20s:     big.NewInt(0),
			expCoins:      coins,
		},
		{
			name: "error - invalid sender (bad address)",
			malleate: func() {
				transfer := transfertypes.NewFungibleTokenPacketData(registeredDenom, "100", "badba1sv9m0g7ycejwr3s369km58h5qe7xj77hvcxrms", ethsecpAddrCosmos, "")
				bz := transfertypes.ModuleCdc.MustMarshalJSON(&transfer)
				packet = channeltypes.NewPacket(bz, 100, transfertypes.PortID, sourceChannel, transfertypes.PortID, haqqChannel, timeoutHeight, 0)
			},
			receiver:      secpAddr,
			ackSuccess:    false,
			checkBalances: false,
			expErc20s:     big.NewInt(0),
			expCoins:      coins,
		},
		{
			name: "error - invalid recipient (bad address)",
			malleate: func() {
				transfer := transfertypes.NewFungibleTokenPacketData(registeredDenom, "100", ethsecpAddrHaqq, "badbadhf0468jjpe6m6vx38s97z2qqe8ldu0njdyf625", "")
				bz := transfertypes.ModuleCdc.MustMarshalJSON(&transfer)
				packet = channeltypes.NewPacket(bz, 100, transfertypes.PortID, sourceChannel, transfertypes.PortID, haqqChannel, timeoutHeight, 0)
			},
			receiver:      secpAddr,
			ackSuccess:    false,
			checkBalances: false,
			expErc20s:     big.NewInt(0),
			expCoins:      coins,
		},
		{
			name: "error - sender == receiver, not from Evm channel",
			malleate: func() {
				transfer := transfertypes.NewFungibleTokenPacketData(registeredDenom, "100", ethsecpAddrHaqq, ethsecpAddrCosmos, "")
				bz := transfertypes.ModuleCdc.MustMarshalJSON(&transfer)
				packet = channeltypes.NewPacket(bz, 1, transfertypes.PortID, sourceChannel, transfertypes.PortID, "channel-100", timeoutHeight, 0)
			},
			ackSuccess:    false,
			receiver:      secpAddr,
			expErc20s:     big.NewInt(0),
			expCoins:      coins,
			checkBalances: false,
		},
		{
			name: "no-op - receiver is module account",
			malleate: func() {
				secpAddr = suite.app.AccountKeeper.GetModuleAccount(suite.ctx, "erc20").GetAddress()
				transfer := transfertypes.NewFungibleTokenPacketData(registeredDenom, "100", secpAddrCosmos, secpAddr.String(), "")
				bz := transfertypes.ModuleCdc.MustMarshalJSON(&transfer)
				packet = channeltypes.NewPacket(bz, 100, transfertypes.PortID, sourceChannel, transfertypes.PortID, haqqChannel, timeoutHeight, 0)
			},
			ackSuccess:    true,
			receiver:      secpAddr,
			expErc20s:     big.NewInt(0),
			expCoins:      coins,
			checkBalances: true,
		},
		{
			name: "no-op - base denomination",
			malleate: func() {
				// base denom should be prefixed
				sourcePrefix := transfertypes.GetDenomPrefix(transfertypes.PortID, sourceChannel)
				prefixedDenom := sourcePrefix + s.app.StakingKeeper.BondDenom(suite.ctx)
				transfer := transfertypes.NewFungibleTokenPacketData(prefixedDenom, "100", secpAddrCosmos, ethsecpAddrHaqq, "")
				bz := transfertypes.ModuleCdc.MustMarshalJSON(&transfer)
				packet = channeltypes.NewPacket(bz, 1, transfertypes.PortID, sourceChannel, transfertypes.PortID, haqqChannel, timeoutHeight, 0)
			},
			ackSuccess:    true,
			receiver:      ethsecpAddr,
			expErc20s:     big.NewInt(0),
			expCoins:      coins,
			checkBalances: true,
		},
		{
			name: "no-op - pair is not registered",
			malleate: func() {
				transfer := transfertypes.NewFungibleTokenPacketData(erc20Denom, "100", secpAddrCosmos, ethsecpAddrHaqq, "")
				bz := transfertypes.ModuleCdc.MustMarshalJSON(&transfer)
				packet = channeltypes.NewPacket(bz, 1, transfertypes.PortID, sourceChannel, transfertypes.PortID, haqqChannel, timeoutHeight, 0)
			},
			ackSuccess:    true,
			receiver:      ethsecpAddr,
			expErc20s:     big.NewInt(0),
			expCoins:      coins,
			checkBalances: true,
		},
		{
			name: "no-op - pair disabled",
			malleate: func() {
				pk1 := secp256k1.GenPrivKey()
				sourcePrefix := transfertypes.GetDenomPrefix(transfertypes.PortID, sourceChannel)
				prefixedDenom := sourcePrefix + registeredDenom
				otherSecpAddrHaqq := sdk.AccAddress(pk1.PubKey().Address()).String()
				transfer := transfertypes.NewFungibleTokenPacketData(prefixedDenom, "500", otherSecpAddrHaqq, ethsecpAddrHaqq, "")
				bz := transfertypes.ModuleCdc.MustMarshalJSON(&transfer)
				packet = channeltypes.NewPacket(bz, 1, transfertypes.PortID, sourceChannel, transfertypes.PortID, haqqChannel, timeoutHeight, 0)
			},
			ackSuccess: true,
			receiver:   ethsecpAddr,
			expErc20s:  big.NewInt(0),
			expCoins: sdk.NewCoins(
				sdk.NewCoin(utils.BaseDenom, math.NewInt(1000)),
				sdk.NewCoin(registeredDenom, math.NewInt(0)),
				sdk.NewCoin(ibcBase, math.NewInt(1000)),
			),
			checkBalances:    false,
			disableTokenPair: true,
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.mintFeeCollector = true
			suite.SetupTest() // reset

			tc.malleate()

			// Set Denom Trace
			denomTrace := transfertypes.DenomTrace{
				Path:      path,
				BaseDenom: registeredDenom,
			}
			suite.app.TransferKeeper.SetDenomTrace(suite.ctx, denomTrace)

			// Set Cosmos Channel
			channel := channeltypes.Channel{
				State:          channeltypes.INIT,
				Ordering:       channeltypes.UNORDERED,
				Counterparty:   channeltypes.NewCounterparty(transfertypes.PortID, sourceChannel),
				ConnectionHops: []string{sourceChannel},
			}
			suite.app.IBCKeeper.ChannelKeeper.SetChannel(suite.ctx, transfertypes.PortID, haqqChannel, channel)

			// Set Next Sequence Send
			suite.app.IBCKeeper.ChannelKeeper.SetNextSequenceSend(suite.ctx, transfertypes.PortID, haqqChannel, 1)

			suite.app.Erc20Keeper = keeper.NewKeeper(
				suite.app.GetKey(types.StoreKey),
				suite.app.AppCodec(),
				authtypes.NewModuleAddress(govtypes.ModuleName),
				suite.app.AccountKeeper,
				suite.app.BankKeeper,
				suite.app.EvmKeeper,
				suite.app.StakingKeeper,
				suite.app.AuthzKeeper,
				&suite.app.TransferKeeper,
			)

			// Fund receiver account with ISLM, ERC20 coins and IBC vouchers
			// We do this since we are interested in the conversion portion w/ OnRecvPacket
			err = testutil.FundAccount(suite.ctx, suite.app.BankKeeper, tc.receiver, coins)
			suite.Require().NoError(err)

			// Register Token Pair for testing
			contractAddr := suite.setupRegisterERC20Pair(1)
			id := suite.app.Erc20Keeper.GetTokenPairID(suite.ctx, contractAddr.String())
			pair, _ := suite.app.Erc20Keeper.GetTokenPair(suite.ctx, id)
			suite.Require().NotNil(pair)

			if tc.disableERC20 {
				params := suite.app.Erc20Keeper.GetParams(suite.ctx)
				params.EnableErc20 = false
				suite.app.Erc20Keeper.SetParams(suite.ctx, params) //nolint:errcheck
			}

			if tc.disableTokenPair {
				_, err := suite.app.Erc20Keeper.ToggleConversion(suite.ctx, pair.Denom)
				suite.Require().NoError(err)
			}

			// Perform IBC callback
			ack := suite.app.Erc20Keeper.OnRecvPacket(suite.ctx, packet, expAck)

			// Check acknowledgement
			if tc.ackSuccess {
				suite.Require().True(ack.Success(), string(ack.Acknowledgement()))
				suite.Require().Equal(expAck, ack)
			} else {
				suite.Require().False(ack.Success(), string(ack.Acknowledgement()))
			}

			if tc.checkBalances {
				// Check ERC20 balances
				balanceTokenAfter := suite.app.Erc20Keeper.BalanceOf(suite.ctx, contracts.ERC20MinterBurnerDecimalsContract.ABI, pair.GetERC20Contract(), common.BytesToAddress(tc.receiver.Bytes()))
				suite.Require().Equal(tc.expErc20s.Int64(), balanceTokenAfter.Int64())
				// Check Cosmos Coin Balances
				balances := suite.app.BankKeeper.GetAllBalances(suite.ctx, tc.receiver)
				suite.Require().Equal(tc.expCoins, balances)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestConvertCoinToERC20FromPacket() {
	senderAddr := "haqq1tjdjfavsy956d25hvhs3p0nw9a7pfghqm0up92"

	testCases := []struct {
		name     string
		malleate func() transfertypes.FungibleTokenPacketData
		transfer transfertypes.FungibleTokenPacketData
		expPass  bool
	}{
		{
			name: "error - invalid sender",
			malleate: func() transfertypes.FungibleTokenPacketData {
				return transfertypes.NewFungibleTokenPacketData("aISLM", "10", "", "", "")
			},
			expPass: false,
		},
		{
			name: "pass - is base denom",
			malleate: func() transfertypes.FungibleTokenPacketData {
				return transfertypes.NewFungibleTokenPacketData("aISLM", "10", senderAddr, "", "")
			},
			expPass: true,
		},
		{
			name: "pass - erc20 is disabled",
			malleate: func() transfertypes.FungibleTokenPacketData {
				// Register Token Pair for testing
				contractAddr := suite.setupRegisterERC20Pair(1)
				id := suite.app.Erc20Keeper.GetTokenPairID(suite.ctx, contractAddr.String())
				pair, _ := suite.app.Erc20Keeper.GetTokenPair(suite.ctx, id)
				suite.Require().NotNil(pair)

				params := suite.app.Erc20Keeper.GetParams(suite.ctx)
				params.EnableErc20 = false
				_ = suite.app.Erc20Keeper.SetParams(suite.ctx, params)
				return transfertypes.NewFungibleTokenPacketData(pair.Denom, "10", senderAddr, "", "")
			},
			expPass: true,
		},
		{
			name: "pass - denom is not registered",
			malleate: func() transfertypes.FungibleTokenPacketData {
				return transfertypes.NewFungibleTokenPacketData(metadataIbc.Base, "10", senderAddr, "", "")
			},
			expPass: true,
		},
		{
			name: "pass - erc20 is disabled",
			malleate: func() transfertypes.FungibleTokenPacketData {
				// Register Token Pair for testing
				contractAddr := suite.setupRegisterERC20Pair(1)
				id := suite.app.Erc20Keeper.GetTokenPairID(suite.ctx, contractAddr.String())
				pair, _ := suite.app.Erc20Keeper.GetTokenPair(suite.ctx, id)
				suite.Require().NotNil(pair)

				err := testutil.FundAccount(
					suite.ctx,
					suite.app.BankKeeper,
					sdk.MustAccAddressFromBech32(senderAddr),
					sdk.NewCoins(
						sdk.NewCoin(pair.Denom, math.NewInt(100)),
					),
				)
				suite.Require().NoError(err)

				_, err = suite.app.EvmKeeper.CallEVM(s.ctx, contracts.ERC20MinterBurnerDecimalsContract.ABI, suite.address, contractAddr, true, "mint", types.ModuleAddress, big.NewInt(10))
				suite.Require().NoError(err)

				return transfertypes.NewFungibleTokenPacketData(pair.Denom, "10", senderAddr, "", "")
			},
			expPass: true,
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.mintFeeCollector = true
			suite.SetupTest() // reset

			transfer := tc.malleate()

			err := suite.app.Erc20Keeper.ConvertCoinToERC20FromPacket(suite.ctx, transfer)
			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestOnAcknowledgementPacket() {
	var (
		data transfertypes.FungibleTokenPacketData
		ack  channeltypes.Acknowledgement
		pair types.TokenPair
	)

	// secp256k1 account
	senderPk := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(senderPk.PubKey().Address())

	receiverPk := secp256k1.GenPrivKey()
	receiver := sdk.AccAddress(receiverPk.PubKey().Address())
	fmt.Println(receiver)
	testCases := []struct {
		name     string
		malleate func()
		expERC20 *big.Int
		expPass  bool
	}{
		{
			name: "no-op - ack error sender is module account",
			malleate: func() {
				// Register Token Pair for testing
				contractAddr := suite.setupRegisterERC20Pair(1)
				id := suite.app.Erc20Keeper.GetTokenPairID(suite.ctx, contractAddr.String())
				pair, _ = suite.app.Erc20Keeper.GetTokenPair(suite.ctx, id)
				suite.Require().NotNil(pair)

				// for testing purposes we can only fund is not allowed to receive funds
				moduleAcc := suite.app.AccountKeeper.GetModuleAccount(suite.ctx, "erc20")
				sender = moduleAcc.GetAddress()
				err := testutil.FundModuleAccount(
					suite.ctx,
					suite.app.BankKeeper,
					moduleAcc.GetName(),
					sdk.NewCoins(
						sdk.NewCoin(pair.Denom, math.NewInt(100)),
					),
				)
				suite.Require().NoError(err)

				ack = channeltypes.NewErrorAcknowledgement(errors.New(""))
				data = transfertypes.NewFungibleTokenPacketData(pair.Denom, "100", sender.String(), receiver.String(), "")
			},
			expPass:  true,
			expERC20: big.NewInt(0),
		},
		{
			name: "no-op - positive ack",
			malleate: func() {
				// Register Token Pair for testing
				contractAddr := suite.setupRegisterERC20Pair(1)
				id := suite.app.Erc20Keeper.GetTokenPairID(suite.ctx, contractAddr.String())
				pair, _ = suite.app.Erc20Keeper.GetTokenPair(suite.ctx, id)
				suite.Require().NotNil(pair)

				sender = sdk.AccAddress(senderPk.PubKey().Address())

				// Fund receiver account with ISLM, ERC20 coins and IBC vouchers
				// We do this since we are interested in the conversion portion w/ OnRecvPacket
				err := testutil.FundAccount(
					suite.ctx,
					suite.app.BankKeeper,
					sender,
					sdk.NewCoins(
						sdk.NewCoin(pair.Denom, math.NewInt(100)),
					),
				)
				suite.Require().NoError(err)

				ack = channeltypes.NewResultAcknowledgement([]byte{1})
			},
			expERC20: big.NewInt(0),
			expPass:  true,
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest() // reset

			tc.malleate()

			err := suite.app.Erc20Keeper.OnAcknowledgementPacket(
				suite.ctx, channeltypes.Packet{}, data, ack,
			)
			suite.Require().NoError(err)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}

			// check balance is the same as expected
			balance := suite.app.Erc20Keeper.BalanceOf(
				suite.ctx, contracts.ERC20MinterBurnerDecimalsContract.ABI,
				pair.GetERC20Contract(),
				common.BytesToAddress(sender.Bytes()),
			)
			suite.Require().Equal(tc.expERC20.Int64(), balance.Int64())
		})
	}
}

func (suite *KeeperTestSuite) TestOnTimeoutPacket() {
	testCases := []struct {
		name     string
		malleate func() transfertypes.FungibleTokenPacketData
		transfer transfertypes.FungibleTokenPacketData
		expPass  bool
	}{
		{
			name: "no-op - sender is module account",
			malleate: func() transfertypes.FungibleTokenPacketData {
				// any module account can be passed here
				moduleAcc := suite.app.AccountKeeper.GetModuleAccount(suite.ctx, evmtypes.ModuleName)

				return transfertypes.NewFungibleTokenPacketData("", "10", moduleAcc.GetAddress().String(), "", "")
			},
			expPass: true,
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest()

			data := tc.malleate()

			err := suite.app.Erc20Keeper.OnTimeoutPacket(suite.ctx, channeltypes.Packet{}, data)
			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
