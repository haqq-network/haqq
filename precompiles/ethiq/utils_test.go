package ethiq_test

import (
	"fmt"
	"math/big"
	"slices"
	"time"

	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	"github.com/haqq-network/haqq/precompiles/ethiq"
	"github.com/haqq-network/haqq/testutil/integration/haqq/grpc"
	testkeyring "github.com/haqq-network/haqq/testutil/integration/haqq/keyring"
	ethiqtypes "github.com/haqq-network/haqq/x/ethiq/types"

	//nolint:revive // dot imports are fine for Ginkgo
	. "github.com/onsi/gomega"

	"github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/haqq-network/haqq/precompiles/authorization"
	"github.com/haqq-network/haqq/precompiles/testutil"
	"github.com/haqq-network/haqq/testutil/integration/haqq/factory"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

func (s *PrecompileTestSuite) SetupApproval(
	granterPriv types.PrivKey,
	grantee common.Address,
	amount *big.Int,
	msgTypes []string,
) {
	precompileAddr := s.precompile.Address()
	txArgs := evmtypes.EvmTxArgs{
		To: &precompileAddr,
	}
	approveArgs := factory.CallArgs{
		ContractABI: s.precompile.ABI,
		MethodName:  authorization.ApproveMethod,
		Args: []interface{}{
			grantee, amount, msgTypes,
		},
	}

	logCheckArgs := testutil.LogCheckArgs{
		ABIEvents: s.precompile.Events,
		ExpEvents: []string{authorization.EventTypeApproval},
		ExpPass:   true,
	}

	res, _, err := s.factory.CallContractAndCheckLogs(
		granterPriv,
		txArgs, approveArgs,
		logCheckArgs,
	)
	Expect(err).To(BeNil(), "error while calling the contract to approve")
	Expect(s.network.NextBlock()).To(BeNil())

	// Check if the approval event is emitted
	granterAddr := common.BytesToAddress(granterPriv.PubKey().Address().Bytes())
	testutil.CheckAuthorizationEvents(
		s.precompile.Events[authorization.EventTypeApproval],
		s.precompile.Address(),
		granterAddr,
		grantee,
		res,
		s.network.GetContext().BlockHeight()-1, // use last committed block height
		msgTypes,
		amount,
	)
}

// SetupApprovalWithContractCalls is a helper function used to setup the allowance for the given spender.
func (s *PrecompileTestSuite) SetupApprovalWithContractCalls(
	granter testkeyring.Key,
	txArgs evmtypes.EvmTxArgs,
	approvalArgs factory.CallArgs,
) {
	msgTypes, ok := approvalArgs.Args[1].([]string)
	Expect(ok).To(BeTrue(), "failed to convert msgTypes to []string")
	expAmount, ok := approvalArgs.Args[2].(*big.Int)
	Expect(ok).To(BeTrue(), "failed to convert amount to big.Int")

	logCheckArgs := testutil.LogCheckArgs{
		ABIEvents: s.precompile.Events,
		ExpEvents: []string{authorization.EventTypeApproval},
		ExpPass:   true,
	}

	_, _, err := s.factory.CallContractAndCheckLogs(
		granter.Priv,
		txArgs,
		approvalArgs,
		logCheckArgs,
	)
	Expect(err).To(BeNil(), "error while approving: %v", err)
	Expect(s.network.NextBlock()).To(BeNil())

	// iterate over args
	var (
		authzMint      *ethiqtypes.MintHaqqAuthorization
		authzMintByApp *ethiqtypes.MintHaqqByApplicationIDAuthorization
	)
	for _, msgType := range msgTypes {
		authz, expirationTime, err := CheckAuthorization(s.grpcHandler, s.network.GetEncodingConfig().InterfaceRegistry, msgType, *txArgs.To, granter.Addr)
		Expect(err).To(BeNil())
		Expect(authz).ToNot(BeNil(), "expected authorization to be set")

		switch msgType {
		case ethiq.MintHaqqMsgURL:
			authzMint, ok = authz.(*ethiqtypes.MintHaqqAuthorization)
			Expect(ok).To(BeTrue())
			Expect(authzMint.SpendLimit.Amount.String()).To(Equal(expAmount.String()), "expected different allowance")
		case ethiq.MsgMintHaqqByApplicationMsgURL:
			authzMintByApp, ok = authz.(*ethiqtypes.MintHaqqByApplicationIDAuthorization)
			Expect(ok).To(BeTrue())
			Expect(slices.Contains(authzMintByApp.ApplicationsList, expAmount.Uint64())).To(BeTrue(), "expected application ID to be allowed")
		default:
			s.Fail("msg type %s is not supported", msgType)
		}
		Expect(expirationTime).ToNot(BeNil(), "expected expiration time to not be nil")
	}
}

// ExpectAuthorization is a helper function for tests using the Ginkgo BDD style tests, to check that the
// authorization is correctly set.
func (s *PrecompileTestSuite) ExpectAuthorization(msgTypeURL string, grantee, granter common.Address, spendLimit *sdk.Coin, allowedAppID uint64) {
	grantedAuthz, expirationTime, err := CheckAuthorization(s.grpcHandler, s.network.GetEncodingConfig().InterfaceRegistry, msgTypeURL, grantee, granter)
	Expect(err).To(BeNil())
	Expect(grantedAuthz).ToNot(BeNil(), "expected authorization to be set")
	Expect(grantedAuthz.MsgTypeURL()).To(Equal(msgTypeURL), "expected different authorization type")
	if mintHaqqAuthz, ok := grantedAuthz.(*ethiqtypes.MintHaqqAuthorization); ok {
		Expect(mintHaqqAuthz.SpendLimit).To(Equal(spendLimit), "expected different spend limit")
	}
	if mintHaqqByAppIDAuthz, ok := grantedAuthz.(*ethiqtypes.MintHaqqByApplicationIDAuthorization); ok {
		Expect(slices.Contains(mintHaqqByAppIDAuthz.ApplicationsList, allowedAppID)).To(BeTrue(), "expected application ID to be in the list")
	}
	Expect(expirationTime).ToNot(BeNil(), "expected expiration time to be not be nil")
}

func CheckAuthorization(gh grpc.Handler, ir cdctypes.InterfaceRegistry, msgTypeURL string, grantee, granter common.Address) (authz.Authorization, *time.Time, error) {
	grants, err := gh.GetGrants(sdk.AccAddress(grantee.Bytes()).String(), sdk.AccAddress(granter.Bytes()).String())
	if err != nil {
		return nil, nil, err
	}

	if len(grants) == 0 {
		return nil, nil, fmt.Errorf("no authorizations found for grantee %s and granter %s", grantee, granter)
	}

	var (
		expGrant *authz.Grant
		auth     authz.Authorization
	)
	for _, g := range grants {
		if err = ir.UnpackAny(g.Authorization, &auth); err != nil {
			return nil, nil, err
		}
		mintHaqqAuthz, isMintHaqqAuthz := auth.(*ethiqtypes.MintHaqqAuthorization)
		mintHaqqByAppIDAuthz, isMintHaqqByAppIDAuthz := auth.(*ethiqtypes.MintHaqqByApplicationIDAuthorization)
		if !isMintHaqqAuthz && !isMintHaqqByAppIDAuthz {
			return nil, nil, fmt.Errorf("invalid authorization type. Expected: ethiqtypes.MintHaqqAuthorization or ethiqtypes.MintHaqqByApplicationIDAuthorization, got: %T", auth)
		}

		if mintHaqqAuthz.MsgTypeURL() == msgTypeURL {
			expGrant = g
			break
		}
		if mintHaqqByAppIDAuthz.MsgTypeURL() == msgTypeURL {
			expGrant = g
			break
		}
	}

	if expGrant == nil {
		return nil, nil, fmt.Errorf("invalid authorization type. Expected: %s, got: %s", msgTypeURL, auth.MsgTypeURL())
	}

	return auth, expGrant.Expiration, nil
}
