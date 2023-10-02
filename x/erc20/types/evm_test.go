package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	erc20types "github.com/evmos/evmos/v14/x/erc20/types"
)

func TestNewERC20Data(t *testing.T) {
	data := erc20types.NewERC20Data("test", "ERC20", uint8(18))
	exp := erc20types.ERC20Data{Name: "test", Symbol: "ERC20", Decimals: 0x12}
	require.Equal(t, exp, data)
}
