<!--
order: 2
-->

# MetaMask

Connect your Metamask wallet with Haqq {synopsis}

- [Download Metamask](https://metamask.io/download/) {prereq}

The [MetaMask](https://metamask.io/) browser extension is a wallet for accessing Ethereum-enabled applications and managing user identities. It can be used to connect to {{ $themeConfig.project.name }} through the official testnet or via a locally-running {{ $themeConfig.project.name }} node.

::: tip
If you are planning on developing on Haqq locally and you haven’t already set up your own local node, refer to [the quickstart tutorial](../../quickstart/run_node.md), or follow the instructions in the [GitHub repository](https://github.com/haqq-network).
:::

## Adding a New Network

Open the MetaMask extension on your browser, you may have to log in to your MetaMask account if you are not already. Then click the top right circle and go to `Settings` > `Networks` > `Add Network` and fill the form as shown below.

::: tip
You can also lookup the [EIP155](https://github.com/ethereum/EIPs/blob/master/EIPS/eip-155.md) `Chain ID` by referring to [chainlist.org](https://chainlist.org/). Alternatively, to get the full Chain ID from Genesis, check the [Chain ID](./../../basics/chain_id.md) documentation page.
:::

![metamask networks settings](./../img/metamask_network_settings.png)

Here is the list of fields that you can use to paste on Metamask:

:::: tabs
::: tab Mainnet

- **Network Name:** `Haqq Network`
- **New RPC URL:** `https://rpc.eth.haqq.network`
- **Chain ID:** `haqq_11235-1`
- **Currency Symbol (optional):** `{{ $themeConfig.project.testnet_ticker }}`
- **Block Explorer URL (optional):** `{{ $themeConfig.project.block_explorer_url }}`
:::

::: tab Testnet

- **Network Name:** `Haqq Network // TestEdge`
- **New RPC URL:** `https://rpc.eth.testedge.haqq.network`
- **Chain ID:** `53211`
- **Currency Symbol (optional):** `{{ $themeConfig.project.testnet_ticker }}`
- **Block Explorer URL (optional):** `{{ $themeConfig.project.block_explorer_url }}`
:::

::: tab Local Node

- **Network Name:** `{{ $themeConfig.project.name }}`
- **New RPC URL:** `{{ $themeConfig.project.rpc_url_local }}`
- **Chain ID:** `{{ $themeConfig.project.testnet_chain_id }}`
- **Currency Symbol (optional):** `{{ $themeConfig.project.testnet_ticker }}`
- **Block Explorer URL (optional):** `n/a`
:::
::::

## Import Account to Metamask
### Manual Import

Close the `Settings`, go to `My Accounts` (top right circle) and select `Import Account`. You should see an image like the following one:

![metamask manual import account page](./../img/metamask_import.png)

Now you can export your private key from the terminal using the following command. Again, make sure to replace `mykey` with the name of the key that you want to export and use the correct `keyring-backend`:

```bash
haqqd keys unsafe-export-eth-key mykey --keyring-backend test
```

Go back to the browser and select the `Private Key` option. Then paste the private key exported from the `unsafe-export-eth-key` command.

Your account balance should show up as `1 {{ $themeConfig.project.testnet_ticker }}` and do transfers as usual.

::: tip
If it takes some time to load the balance of the account, change the network to `Main Ethereum Network` (or any other than `Localhost 8545` or `{{ $themeConfig.project.name }}`) and then switch back to `{{ $themeConfig.project.name }}`.
:::

## Reset Account

If you used your Metamask account for a legacy testnet/mainnet upgrade, you will need to reset your account in order to use it with the new network. This will clear your account's transaction history, but it won't change the balances in your accounts or require you to re-enter your `Secret Recovery Phrase`.

::: warning
Make sure you download your [account state](#download-account-state) to persist public account addresses and transactions before clearing your wallet accounts.
:::

Go to `Settings` > `Advanced`  and click the `Reset Account` button as shown below:

![Metamask Account Reset](./../img/reset_account.png)

## Download Account State

To see your Metamask logs, click the top right circle and go to `Settings` > `Advanced` > `State Logs`. If you search through the JSON file for the account address you'll find the transaction history.
