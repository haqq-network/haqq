<!--
order: 4
-->

# Transactions

This section defines the `sdk.Msg` concrete types that result in the state transitions defined on the previous section.

## `MsgLiquidate`

A user broadcasts a `MsgLiquidate` message to liquidate locked ISLM token.

```go
type MsgLiquidate struct {
    // account for liquidation of locked vesting tokens
    LiquidateFrom string `protobuf:"bytes,1,opt,name=liquidate_from,json=liquidateFrom,proto3" json:"liquidate_from,omitempty"`
    // account to send resulted liquid token
    LiquidateTo string `protobuf:"bytes,2,opt,name=liquidate_to,json=liquidateTo,proto3" json:"liquidate_to,omitempty"`
    // amount of tokens subject for liquidation
    Amount types.Coin `protobuf:"bytes,3,opt,name=amount,proto3" json:"amount"`
}
```

Message stateless validation fails if:

- Amount is not positive
- LiquidateFrom bech32 address is invalid
- LiquidateTo bech32 address is invalid

## `MsgRedeem`

A user broadcasts a `MsgRedeem` message to redeem liquid token to locked ISLM.

```go
type MsgRedeem struct {
    RedeemFrom string `protobuf:"bytes,1,opt,name=redeem_from,json=redeemFrom,proto3" json:"redeem_from,omitempty"`
    // destination address for vesting tokens
    RedeemTo string `protobuf:"bytes,2,opt,name=redeem_to,json=redeemTo,proto3" json:"redeem_to,omitempty"`
    // amount of vesting tokens to redeem from liquidation module
    Amount types.Coin `protobuf:"bytes,3,opt,name=amount,proto3" json:"amount"`
}
```

Message stateless validation fails if:

- Amount is not positive
- RedeemFrom bech32 address is invalid
- RedeemTo bech32 address is invalid
