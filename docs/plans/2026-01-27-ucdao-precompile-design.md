# UCDAO Precompile Design

## Overview

The UCDAO Precompile exposes the UCDAO module's functionality to EVM smart contracts, enabling DAO funding and ownership transfer operations from Solidity. It includes full allowance support using Cosmos SDK's authz module.

**Precompile Address:** `0x0000000000000000000000000000000000000805`

## Requirements

- Expose all UCDAO operations to EVM contracts
- Support authorization (allowances) for transfer operations
- Use amount-based, per-denom allowances via Cosmos authz
- Follow existing precompile patterns (staking, ICS20, ERC20)

## Solidity Interface

```solidity
interface IUCDAO {
    // ===== AUTHORIZATION Methods =====
    
    /// @dev Approve a spender to transfer ownership on caller's behalf
    /// @param spender Address authorized to transfer
    /// @param amount Coins they can transfer (per denom)
    /// @return success Whether approval succeeded
    function approve(
        address spender,
        Coin[] calldata amount
    ) external returns (bool success);
    
    /// @dev Revoke all allowances for a spender
    function revoke(address spender) external returns (bool success);
    
    /// @dev Increase existing allowance
    function increaseAllowance(
        address spender,
        Coin[] calldata amount
    ) external returns (bool success);
    
    /// @dev Decrease existing allowance
    function decreaseAllowance(
        address spender,
        Coin[] calldata amount
    ) external returns (bool success);
    
    /// @dev Query allowance granted to spender by owner
    function allowance(
        address owner,
        address spender
    ) external view returns (Coin[] memory remaining);

    // ===== CORE Methods =====
    
    /// @dev Fund the DAO (caller deposits coins)
    function fund(Coin[] calldata amount) external returns (bool success);
    
    /// @dev Transfer all ownership to new owner
    function transferOwnership(
        address newOwner
    ) external returns (Coin[] memory transferred);
    
    /// @dev Transfer ownership by ratio (0 < ratio <= 1e18 representing 0-100%)
    function transferOwnershipWithRatio(
        address newOwner,
        uint256 ratio
    ) external returns (Coin[] memory transferred);
    
    /// @dev Transfer specific amount of ownership
    function transferOwnershipWithAmount(
        address newOwner,
        Coin[] calldata amount
    ) external returns (Coin[] memory transferred);

    // ===== QUERY Methods =====
    
    /// @dev Get balance of specific denom for address
    function balance(address account, string calldata denom) 
        external view returns (uint256);
    
    /// @dev Get all balances for address
    function allBalances(address account) 
        external view returns (Coin[] memory);
    
    /// @dev Get total DAO balance
    function totalBalance() external view returns (Coin[] memory);
    
    /// @dev Check if module is enabled
    function enabled() external view returns (bool);

    // ===== EVENTS =====
    
    event Approval(
        address indexed owner,
        address indexed spender,
        Coin[] amount
    );
    
    event Revocation(
        address indexed owner,
        address indexed spender
    );
    
    event Fund(address indexed depositor, Coin[] amount);
    
    event TransferOwnership(
        address indexed from,
        address indexed to,
        Coin[] amount
    );
}

struct Coin {
    string denom;
    uint256 amount;
}
```

## Authorization Model

### Cosmos Authz Integration

Uses `banktypes.SendAuthorization` from Cosmos SDK, which supports:
- Multi-denom `SpendLimit` (per-denom allowances)
- Proper `Accept()` method that decrements limits
- Integration with existing authz infrastructure

### Message URLs

```go
var (
    TransferOwnershipMsg           = sdk.MsgTypeURL(&ucdaotypes.MsgTransferOwnership{})
    TransferOwnershipWithRatioMsg  = sdk.MsgTypeURL(&ucdaotypes.MsgTransferOwnershipWithRatio{})
    TransferOwnershipWithAmountMsg = sdk.MsgTypeURL(&ucdaotypes.MsgTransferOwnershipWithAmount{})
)
```

### Authorization Flow

1. **Approve**: Creates `SendAuthorization` grant via `authzKeeper.SaveGrant()`
2. **Transfer on behalf**: Checks authorization, verifies amount ≤ allowance, executes transfer, updates grant
3. **Revoke**: Deletes grant via `authzKeeper.DeleteGrant()`

## File Structure

```
precompiles/ucdao/
├── abi.json              # Solidity ABI definition (embedded)
├── ucdao.go              # Main precompile: constructor, Run(), IsTransaction()
├── types.go              # ABI input/output structs
├── tx.go                 # Transaction methods: Fund, TransferOwnership*
├── query.go              # Query methods: Balance, AllBalances, TotalBalance, Enabled
├── approve.go            # Authorization: Approve, Revoke, Increase/DecreaseAllowance
├── events.go             # Event emission helpers
└── ucdao_test.go         # Unit tests
```

## Precompile Struct

```go
type Precompile struct {
    cmn.Precompile                    // Base: ABI, AuthzKeeper, gas configs
    ucdaoKeeper    ucdaokeeper.Keeper // UCDAO module keeper
    bankKeeper     bankkeeper.Keeper  // For Fund operation
}
```

## Method Classification

| Method | Type | Modifies State | Needs Authorization |
|--------|------|----------------|---------------------|
| `approve` | TX | Yes (authz) | No (granter is caller) |
| `revoke` | TX | Yes (authz) | No (granter is caller) |
| `increaseAllowance` | TX | Yes (authz) | No (granter is caller) |
| `decreaseAllowance` | TX | Yes (authz) | No (granter is caller) |
| `allowance` | Query | No | No |
| `fund` | TX | Yes | No (funds from caller) |
| `transferOwnership` | TX | Yes | **Yes** (if caller ≠ owner) |
| `transferOwnershipWithRatio` | TX | Yes | **Yes** (if caller ≠ owner) |
| `transferOwnershipWithAmount` | TX | Yes | **Yes** (if caller ≠ owner) |
| `balance` | Query | No | No |
| `allBalances` | Query | No | No |
| `totalBalance` | Query | No | No |
| `enabled` | Query | No | No |

## Edge Cases

| Case | Behavior |
|------|----------|
| Approve with amount=0 | Revoke existing authorization (if any) |
| Approve with MaxUint256 | Unlimited authorization (no spend limit) |
| Transfer more than allowance | Reject with "insufficient allowance" |
| Transfer when module disabled | Reject with "module disabled" |
| Self-transfer (owner=newOwner) | Allow (no-op but valid) |
| Caller is owner | Skip authorization check entirely |
| Expired authorization | Reject with "authorization expired" |

## Events

| Event | When Emitted |
|-------|--------------|
| `Approval(owner, spender, amount)` | On approve, increaseAllowance, decreaseAllowance |
| `Revocation(owner, spender)` | On revoke or approve with amount=0 |
| `Fund(depositor, amount)` | On successful fund |
| `TransferOwnership(from, to, amount)` | On any successful transfer |

## Implementation Notes

1. **Gas Costs**: Use `storetypes.KVGasConfig()` for state operations
2. **Expiration**: Default 1 year (`time.Hour * 24 * 365`)
3. **Ratio Precision**: Use 1e18 scale (1e18 = 100%)
4. **Denom Validation**: Only `aISLM` and `aLIQUID*` patterns allowed (enforced by keeper)
