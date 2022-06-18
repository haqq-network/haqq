<!-- This file is auto-generated. Please do not modify it yourself. -->
# Protobuf Documentation
<a name="top"></a>

## Table of Contents

- [haqq/mint/v1/mint.proto](#haqq/mint/v1/mint.proto)
    - [Minter](#haqq.mint.v1.Minter)
    - [Params](#haqq.mint.v1.Params)
  
- [haqq/mint/v1/genesis.proto](#haqq/mint/v1/genesis.proto)
    - [GenesisState](#haqq.mint.v1.GenesisState)
  
- [haqq/mint/v1/query.proto](#haqq/mint/v1/query.proto)
    - [QueryAnnualProvisionsRequest](#haqq.mint.v1.QueryAnnualProvisionsRequest)
    - [QueryAnnualProvisionsResponse](#haqq.mint.v1.QueryAnnualProvisionsResponse)
    - [QueryCurrentBlockProvisionAmountRequest](#haqq.mint.v1.QueryCurrentBlockProvisionAmountRequest)
    - [QueryCurrentBlockProvisionAmountResponse](#haqq.mint.v1.QueryCurrentBlockProvisionAmountResponse)
    - [QueryCurrentEpochEndDateTimeRequest](#haqq.mint.v1.QueryCurrentEpochEndDateTimeRequest)
    - [QueryCurrentEpochEndDateTimeResponse](#haqq.mint.v1.QueryCurrentEpochEndDateTimeResponse)
    - [QueryCurrentEpochNumberRequest](#haqq.mint.v1.QueryCurrentEpochNumberRequest)
    - [QueryCurrentEpochNumberResponse](#haqq.mint.v1.QueryCurrentEpochNumberResponse)
    - [QueryCurrentEpochStartBlockHeightRequest](#haqq.mint.v1.QueryCurrentEpochStartBlockHeightRequest)
    - [QueryCurrentEpochStartBlockHeightResponse](#haqq.mint.v1.QueryCurrentEpochStartBlockHeightResponse)
    - [QueryCurrentEpochStartDateTimeRequest](#haqq.mint.v1.QueryCurrentEpochStartDateTimeRequest)
    - [QueryCurrentEpochStartDateTimeResponse](#haqq.mint.v1.QueryCurrentEpochStartDateTimeResponse)
    - [QueryCurrentEpochTotalProvisionRequest](#haqq.mint.v1.QueryCurrentEpochTotalProvisionRequest)
    - [QueryCurrentEpochTotalProvisionResponse](#haqq.mint.v1.QueryCurrentEpochTotalProvisionResponse)
    - [QueryInflationRequest](#haqq.mint.v1.QueryInflationRequest)
    - [QueryInflationResponse](#haqq.mint.v1.QueryInflationResponse)
    - [QueryParamsRequest](#haqq.mint.v1.QueryParamsRequest)
    - [QueryParamsResponse](#haqq.mint.v1.QueryParamsResponse)
  
    - [Query](#haqq.mint.v1.Query)
  
- [Scalar Value Types](#scalar-value-types)



<a name="haqq/mint/v1/mint.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## haqq/mint/v1/mint.proto



<a name="haqq.mint.v1.Minter"></a>

### Minter
Minter represents the minting state.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `current_epoch_number` | [uint32](#uint32) |  | current epoch number |
| `current_epoch_start_dt` | [string](#string) |  | current epoch start date and time (ISO 8601 string) |
| `current_epoch_start_block_height` | [int64](#int64) |  | current epoch start block height |
| `current_epoch_total_provision` | [string](#string) |  | current epoch total provision amount |






<a name="haqq.mint.v1.Params"></a>

### Params
Params holds parameters for the mint module.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `mint_denom` | [string](#string) |  | type of coin to mint |
| `initial_supply` | [string](#string) |  | the initial supply amount |
| `maximum_supply` | [string](#string) |  | the maximum supply amount |
| `epoch_count` | [uint32](#uint32) |  | the total epoch count |
| `epoch_duration` | [string](#string) |  | the total supply amount |
| `inter_epoch_mint_change_coefficient` | [string](#string) |  | the total supply amount |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->

 <!-- end services -->



<a name="haqq/mint/v1/genesis.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## haqq/mint/v1/genesis.proto



<a name="haqq.mint.v1.GenesisState"></a>

### GenesisState
GenesisState defines the mint module's genesis state.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `minter` | [Minter](#haqq.mint.v1.Minter) |  | minter is a space for holding current inflation information. |
| `params` | [Params](#haqq.mint.v1.Params) |  | params defines all the paramaters of the module. |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->

 <!-- end services -->



<a name="haqq/mint/v1/query.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## haqq/mint/v1/query.proto



<a name="haqq.mint.v1.QueryAnnualProvisionsRequest"></a>

### QueryAnnualProvisionsRequest
QueryAnnualProvisionsRequest is the request type for the
Query/AnnualProvisions RPC method.






<a name="haqq.mint.v1.QueryAnnualProvisionsResponse"></a>

### QueryAnnualProvisionsResponse
QueryAnnualProvisionsResponse is the response type for the
Query/AnnualProvisions RPC method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `annual_provisions` | [bytes](#bytes) |  | annual_provisions is the current minting annual provisions value. |






<a name="haqq.mint.v1.QueryCurrentBlockProvisionAmountRequest"></a>

### QueryCurrentBlockProvisionAmountRequest
QueryCurrentBlockProvisionAmountRequest is the request type for the
Query/CurrentBlockProvisionAmount RPC method.






<a name="haqq.mint.v1.QueryCurrentBlockProvisionAmountResponse"></a>

### QueryCurrentBlockProvisionAmountResponse
QueryCurrentBlockProvisionAmountResponse is the response type for the
Query/CurrentBlockProvisionAmount RPC method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `current_block_provision_amount` | [bytes](#bytes) |  | current_block_provision_amount is the current block calculated provisions amount. |






<a name="haqq.mint.v1.QueryCurrentEpochEndDateTimeRequest"></a>

### QueryCurrentEpochEndDateTimeRequest
QueryCurrentEpochEndDateTimeRequest is the request type for the
Query/CurrentEpochEndDateTime RPC method.






<a name="haqq.mint.v1.QueryCurrentEpochEndDateTimeResponse"></a>

### QueryCurrentEpochEndDateTimeResponse
QueryCurrentEpochEndDateTimeResponse is the response type for the
Query/CurrentEpochEndDateTime RPC method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `current_epoch_end_date_time` | [string](#string) |  | current_epoch_end_date_time is the current minting epoch end date time value. |






<a name="haqq.mint.v1.QueryCurrentEpochNumberRequest"></a>

### QueryCurrentEpochNumberRequest
QueryCurrentEpochNumberRequest is the request type for the
Query/CurrentEpochNumber RPC method.






<a name="haqq.mint.v1.QueryCurrentEpochNumberResponse"></a>

### QueryCurrentEpochNumberResponse
QueryCurrentEpochNumberResponse is the response type for the
Query/CurrentEpochNumber RPC method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `current_epoch_number` | [uint32](#uint32) |  | current_epoch_number is the current minting epoch number value. |






<a name="haqq.mint.v1.QueryCurrentEpochStartBlockHeightRequest"></a>

### QueryCurrentEpochStartBlockHeightRequest
QueryCurrentEpochStartBlockHeightRequest is the request type for the
Query/CurrentEpochStartBlockHeight RPC method.






<a name="haqq.mint.v1.QueryCurrentEpochStartBlockHeightResponse"></a>

### QueryCurrentEpochStartBlockHeightResponse
QueryCurrentEpochStartBlockHeightResponse is the response type for the
Query/CurrentEpochStartBlockHeight RPC method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `current_epoch_start_block_height` | [int64](#int64) |  | current_epoch_start_block_height is the current minting epoch start block height. |






<a name="haqq.mint.v1.QueryCurrentEpochStartDateTimeRequest"></a>

### QueryCurrentEpochStartDateTimeRequest
QueryCurrentEpochStartDateTimeRequest is the request type for the
Query/CurrentEpochStartDateTime RPC method.






<a name="haqq.mint.v1.QueryCurrentEpochStartDateTimeResponse"></a>

### QueryCurrentEpochStartDateTimeResponse
QueryCurrentEpochStartDateTimeResponse is the response type for the
Query/CurrentEpochStartDateTime RPC method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `current_epoch_start_date_time` | [string](#string) |  | current_epoch_start_date_time is the current minting epoch start date time value. |






<a name="haqq.mint.v1.QueryCurrentEpochTotalProvisionRequest"></a>

### QueryCurrentEpochTotalProvisionRequest
QueryCurrentEpochTotalProvisionRequest is the request type for the
Query/urrentEpochTotalProvision RPC method.






<a name="haqq.mint.v1.QueryCurrentEpochTotalProvisionResponse"></a>

### QueryCurrentEpochTotalProvisionResponse
QueryurrentEpochTotalProvisionResponse is the response type for the
Query/urrentEpochTotalProvision RPC method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `current_epoch_total_provision` | [bytes](#bytes) |  | current_epoch_total_provision is the current minting epoch provisions value. |






<a name="haqq.mint.v1.QueryInflationRequest"></a>

### QueryInflationRequest
QueryInflationRequest is the request type for the Query/Inflation RPC method.






<a name="haqq.mint.v1.QueryInflationResponse"></a>

### QueryInflationResponse
QueryInflationResponse is the response type for the Query/Inflation RPC
method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `inflation` | [bytes](#bytes) |  | inflation is the current minting inflation value. |






<a name="haqq.mint.v1.QueryParamsRequest"></a>

### QueryParamsRequest
QueryParamsRequest is the request type for the Query/Params RPC method.






<a name="haqq.mint.v1.QueryParamsResponse"></a>

### QueryParamsResponse
QueryParamsResponse is the response type for the Query/Params RPC method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `params` | [Params](#haqq.mint.v1.Params) |  | params defines the parameters of the module. |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->


<a name="haqq.mint.v1.Query"></a>

### Query
Query provides defines the gRPC querier service.

| Method Name | Request Type | Response Type | Description | HTTP Verb | Endpoint |
| ----------- | ------------ | ------------- | ------------| ------- | -------- |
| `Params` | [QueryParamsRequest](#haqq.mint.v1.QueryParamsRequest) | [QueryParamsResponse](#haqq.mint.v1.QueryParamsResponse) | Params returns the total set of minting parameters. | GET|/haqq/mint/v1/params|
| `CurrentEpochNumber` | [QueryCurrentEpochNumberRequest](#haqq.mint.v1.QueryCurrentEpochNumberRequest) | [QueryCurrentEpochNumberResponse](#haqq.mint.v1.QueryCurrentEpochNumberResponse) | CurrentEpochNumber current epoch number value. | GET|/haqq/mint/v1/current_epoch_number|
| `CurrentEpochStartDateTime` | [QueryCurrentEpochStartDateTimeRequest](#haqq.mint.v1.QueryCurrentEpochStartDateTimeRequest) | [QueryCurrentEpochStartDateTimeResponse](#haqq.mint.v1.QueryCurrentEpochStartDateTimeResponse) | CurrentEpochStartDateTime current epoch start date time value. | GET|/haqq/mint/v1/current_epoch_start_datetime|
| `CurrentEpochEndDateTime` | [QueryCurrentEpochEndDateTimeRequest](#haqq.mint.v1.QueryCurrentEpochEndDateTimeRequest) | [QueryCurrentEpochEndDateTimeResponse](#haqq.mint.v1.QueryCurrentEpochEndDateTimeResponse) | CurrentEpochEndDateTime current epoch start date time value. | GET|/haqq/mint/v1/current_epoch_end_datetime|
| `CurrentEpochStartBlockHeight` | [QueryCurrentEpochStartBlockHeightRequest](#haqq.mint.v1.QueryCurrentEpochStartBlockHeightRequest) | [QueryCurrentEpochStartBlockHeightResponse](#haqq.mint.v1.QueryCurrentEpochStartBlockHeightResponse) | CurrentEpochStartBlockHeight current epoch start block height value. | GET|/haqq/mint/v1/current_epoch_start_block_height|
| `CurrentBlockProvisionAmount` | [QueryCurrentBlockProvisionAmountRequest](#haqq.mint.v1.QueryCurrentBlockProvisionAmountRequest) | [QueryCurrentBlockProvisionAmountResponse](#haqq.mint.v1.QueryCurrentBlockProvisionAmountResponse) | CurrentBlockProvisionAmount current block provision amount value. | GET|/haqq/mint/v1/current_block_provision_amount|

 <!-- end services -->



## Scalar Value Types

| .proto Type | Notes | C++ | Java | Python | Go | C# | PHP | Ruby |
| ----------- | ----- | --- | ---- | ------ | -- | -- | --- | ---- |
| <a name="double" /> double |  | double | double | float | float64 | double | float | Float |
| <a name="float" /> float |  | float | float | float | float32 | float | float | Float |
| <a name="int32" /> int32 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint32 instead. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="int64" /> int64 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint64 instead. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="uint32" /> uint32 | Uses variable-length encoding. | uint32 | int | int/long | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="uint64" /> uint64 | Uses variable-length encoding. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum or Fixnum (as required) |
| <a name="sint32" /> sint32 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int32s. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sint64" /> sint64 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int64s. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="fixed32" /> fixed32 | Always four bytes. More efficient than uint32 if values are often greater than 2^28. | uint32 | int | int | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="fixed64" /> fixed64 | Always eight bytes. More efficient than uint64 if values are often greater than 2^56. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum |
| <a name="sfixed32" /> sfixed32 | Always four bytes. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sfixed64" /> sfixed64 | Always eight bytes. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="bool" /> bool |  | bool | boolean | boolean | bool | bool | boolean | TrueClass/FalseClass |
| <a name="string" /> string | A string must always contain UTF-8 encoded or 7-bit ASCII text. | string | String | str/unicode | string | string | string | String (UTF-8) |
| <a name="bytes" /> bytes | May contain any arbitrary sequence of bytes. | string | ByteString | str | []byte | ByteString | string | String (ASCII-8BIT) |

