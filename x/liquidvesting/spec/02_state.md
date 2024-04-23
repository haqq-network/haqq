<!--
order: 2
-->

# State

## State Objects

The `x/liquidvesting` module keeps the following objects in state:

| State Object | Description           | Key                             | Value           | Store    |
|--------------|-----------------------|---------------------------------|-----------------| -------- |
| `Denom`      | Liquid token bytecode | `[]byte{1} + []byte(baseDenom)` | `[]byte{denom}` | KV       |

### Denom

Denom aka liquid token representation of locked ISLM with unlock schedule

```go
type Denom struct {
	// base_denom main identifier for the denom, used to query it from store.
	BaseDenom string `protobuf:"bytes,1,opt,name=base_denom,json=baseDenom,proto3" json:"base_denom,omitempty"`
	// display_denom identifier used for display name for broad audience
	DisplayDenom string `protobuf:"bytes,2,opt,name=display_denom,json=displayDenom,proto3" json:"display_denom,omitempty"`
	// original_denom which liquid denom derived from
	OriginalDenom string `protobuf:"bytes,3,opt,name=original_denom,json=originalDenom,proto3" json:"original_denom,omitempty"`
	// start date
	StartTime time.Time `protobuf:"bytes,4,opt,name=start_time,json=startTime,proto3,stdtime" json:"start_time"`
	// end_date
	EndTime time.Time `protobuf:"bytes,5,opt,name=end_time,json=endTime,proto3,stdtime" json:"end_time"`
	// lockup periods
	LockupPeriods github_com_cosmos_cosmos_sdk_x_auth_vesting_types.Periods `protobuf:"bytes,6,rep,name=lockup_periods,json=lockupPeriods,proto3,castrepeated=github.com/cosmos/cosmos-sdk/x/auth/vesting/types.Periods" json:"lockup_periods"`
}
```

### Liquid token base denom

The unique identifier of a `Denom` is obtained by combining prefix `LIQUID` and numeric id which increments every time new liquid token is created e. g. `LIQUID12`

### Original denom

Original denom is keeping track of which denom liquid token derives from. In most of the cases it will be ISLM

### Start time

Defines start of unlock schedule bound to luqid token. Always match token creation date

### End time

Defines the date when liquid token schedule ends

### LockupPeriods

The main part of liquid token schedule consist of sdk vesting periods

```go
type Period struct {
	// Period duration in seconds.
	Length int64                                    `protobuf:"varint,1,opt,name=length,proto3" json:"length,omitempty"`
	// Period amount
	Amount github_com_cosmos_cosmos_sdk_types.Coins `protobuf:"bytes,2,rep,name=amount,proto3,castrepeated=github.com/cosmos/cosmos-sdk/types.Coins" json:"amount"`
}
```

## Genesis State

The `x/liquidvesting` module's `GenesisState` defines the state necessary for initializing the chain from a previous exported height. It contains the module parameters and the existing liquid token :

```go
// GenesisState defines the module's genesis state.
type GenesisState struct {
	// params defines all the paramaters of the module.
	Params       Params  `protobuf:"bytes,1,opt,name=params,proto3" json:"params"`
	// keeps track of denom ID
	DenomCounter uint64  `protobuf:"varint,2,opt,name=denomCounter,proto3" json:"denomCounter,omitempty"`
	// list of  liquid denoms
	Denoms       []Denom `protobuf:"bytes,3,rep,name=denoms,proto3" json:"denoms"`
}
```
