<!--
order: 1
-->

# Chain ID

Learn about the Haqq chain-id format {synopsis}

## Official Chain IDs

:::: tabs
::: tab Testnets

| Name                              | Chain ID                                              | Identifier | EIP155 Number                                 | Version Number                                      |
|-----------------------------------|-------------------------------------------------------|------------|-----------------------------------------------|-----------------------------------------------------|
| Haqq TestNow | `haqq_112357-1` | `haqq`    | `112357` | `1.0.0` |
| Haqq TestEdge                | `haqq_53211-1` | `haqq`    | `53211` | `1.0.0`                                                 |

:::
::: tab Mainnet

| Name                                            | Chain ID                                      | Identifier | EIP155 Number                         | Version Number                            |
|-------------------------------------------------|-----------------------------------------------|------------|---------------------------------------|-------------------------------------------|
| Haqq | `haqq_11235-1` | `haqq`    | `11235` | {{ $themeConfig.project.version_number }} |
:::
::::


## The Chain Identifier

Every chain must have a unique identifier or `chain-id`. Tendermint requires each application to
define its own `chain-id` in the [genesis.json fields](https://docs.tendermint.com/master/spec/core/genesis.html#genesis-fields). However, in order to comply with both EIP155 and Cosmos standard for chain upgrades, Haqq-compatible chains must implement a special structure for their chain identifiers.

## Structure

The Haqq Chain ID contains 3 main components

- **Identifier**: Unstructured string that defines the name of the application.
- **EIP155 Number**: Immutable [EIP155](https://github.com/ethereum/EIPs/blob/master/EIPS/eip-155.md) `CHAIN_ID` that defines the replay attack protection number.
- **Version Number**: Is the version number (always positive) that the chain is currently running.
This number **MUST** be incremented every time the chain is upgraded or forked in order to avoid network or consensus errors.

### Format

The format for specifying and Haqq compatible chain-id in genesis is the following:

```bash
{identifier}_{EIP155}-{version}
```

The following table provides an example where the second row corresponds to an upgrade from the first one:

| ChainID        | Identifier | EIP155 Number | Version Number |
|----------------|------------|---------------|----------------|
| `haqq_11235-1` | haqq      | 11235          | 1              |
| `haqq_11235-2` | haqq      | 11235          | 2              |
| `...`          | ...        | ...           | ...            |
| `haqq_11235-N` | haqq      | 11235          | N              |
