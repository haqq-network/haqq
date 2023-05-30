<!--
order: 2
-->

# Faucet for TestEdge2 network

Check how to obtain testnet tokens from the Haqq faucet {synopsis}

The Haqq TestEdge2 Faucet distributes small amounts of ISLM to anyone who can provide a valid testnet address for free.

::: tip
Follow the [Metamask](./../../guides/keys-wallets/metamask.md) guide for more info on how to setup your wallet account.
:::

## Request Tokens on Telegram

<!-- markdown-link-check-disable-next-line -->
1. Join to our `TestEdge2` faucet channel at [https://t.me/haqq_faucet](https://t.me/haqq_faucet)
2. Simply write `!faucet <your hex address>` to receive `10 ISLM`

## Request Tokens on Discord

<!-- markdown-link-check-disable-next-line -->
1. Join our Discord at [https://discord.gg/islamic-coin](https://discord.gg/islamic-coin)
2. Verify your account.
4. Go to the HAQQERS category and click on the #faucet channel at [#faucet](https://discord.com/channels/989535240882114581/1075694966183043083).
3. Simply write `!faucet <your hex address>` to receive 10 ISLM.

## Request Tokens on Web

<!-- markdown-link-check-disable-next-line -->
1. Sign in to the MetaMask extension. 
2. Visit the [TestEdge-2 Faucet](https://testedge2.haqq.network) to request tokens for the testnet.
3. Click on the MetaMask button in the connect wallet section.
4. A popup window with your MetaMask account will appear. Choose the account you want to connect with the Faucet. Then, click on the Next button and the Connect button to establish the connection.
5. Click on the Login with Github button to authorize via your GitHub account.
6. Click on the Sign up with Github button to approve authorization.
7. Complete the reCAPTCHA test to confirm that you are not a robot.
8. Click on the Request Tokens button to receive 1000 ISLM.

## Rate limits

### Web
To prevent the faucet account from draining the available funds, the Haqq TestEdge2 faucet imposes a maximum number of requests for a given period of time. By default, the faucet service accepts only one request per day per address. You can request ISLM from the faucet for each address only once every 24 hours. If you try to request multiple times within the 24-hour cooldown phase, no transaction will be initiated. Please try again in 24 hours.

### Discord / Telegram

You can request ISLM from the faucet only once every hour. If you try to request multiple times within the one-hour cooldown period, no transaction will be initiated. Please try again in one hour.


## Amount

For each request, the faucet transfers 1,000 ISLM on the `Web` and 10 ISLM on `Telegram` and `Discord` to the specified address.

::: danger
These are test coins on a test network and do not have any real value for regular users. They are designed for developers to test and develop their applications without using real money. Please note that these test coins are not transferable to the main network and cannot be exchanged for real money or other cryptocurrencies. They are solely for testing purposes on the test network.
:::

<!-- # Faucet-localnet

The faucet is a web application with the goal of distributing small amounts of Ether in private and test networks.

## Features

* Allow to configure the funding account via private key or keystore
* Asynchronous processing Txs to achieve parallel execution of user requests
* Rate limiting by ETH address and IP address as a precaution against spam
* Prevent X-Forwarded-For spoofing by specifying the count of reverse proxies

## Get started

### Prerequisites

* Go (1.16 or later)
* Node.js

### Installation

1. Clone the repository and navigate to the app’s directory
```bash
git clone https://github.com/haqq-network/faucet-testnet.git
cd faucet-testnet
```

2. Bundle Frontend web with Rollup
```bash
npm run build
```

3. Build Go project 
```bash
go build -o faucet-testnet
```

## Usage

**Use private key to fund users**

```bash
./faucet-testnet -httpport 8080 -wallet.provider http://localhost:8545 -wallet.privkey privkey
```

**Use keystore to fund users**

```bash
./faucet-testnet -httpport 8080 -wallet.provider http://localhost:8545 -wallet.keyjson keystore -wallet.keypass password.txt
```

### Configuration

You can configure the funder by using environment variables instead of command-line flags as follows:
```bash
export WEB3_PROVIDER=rpc endpoint
export PRIVATE_KEY=hex private key
```

or

```bash
export WEB3_PROVIDER=rpc endpoint
export KEYSTORE=keystore path
echo "your keystore password" > `pwd`/password.txt
```

Then run the faucet application without the wallet command-line flags:
```bash
./faucet-testnet -httpport 8080
```

**Optional Flags**

The following are the available command-line flags(excluding above wallet flags):

| Flag           | Description                                      | Default Value
| -------------- | ------------------------------------------------ | -------------
| -httpport      | Listener port to serve HTTP connection           | 8080
| -proxycount    | Count of reverse proxies in front of the server  | 0
| -queuecap      | Maximum transactions waiting to be sent          | 100
| -faucet.amount | Number of Ethers to transfer per user request    | 1
| -faucet.minutes| Number of minutes to wait between funding rounds | 1440
| -faucet.name   | Network name to display on the frontend          | testnet

### Docker deployment

```bash
docker run -d -p 8080:8080 -e WEB3_PROVIDER=rpc endpoint -e PRIVATE_KEY=hex private key haqq-network/faucet-testnet:1.1.0
```

or

```bash
docker run -d -p 8080:8080 -e WEB3_PROVIDER=rpc endpoint -e KEYSTORE=keystore path -v `pwd`/keystore:/app/keystore -v `pwd`/password.txt:/app/password.txt haqq-network/faucet-testnet:1.1.0
```

### Heroku deployment

```bash
heroku create
heroku buildpacks:add heroku/nodejs
heroku buildpacks:add heroku/go
heroku config:set WEB3_PROVIDER=rpc endpoint
heroku config:set PRIVATE_KEY=hex private key
git push heroku main
heroku open
```

or

<a href="https://heroku.com/deploy">
  <img src="https://www.herokucdn.com/deploy/button.svg" alt="Deploy" style="width:20%;">
</a>


> tip: Free web dyno goes to sleep and discards in-memory rate limiting records after 30 minutes of inactivity, so `faucet.minutes` configuration greater than 30 doesn't work properly in the free Heroku plan.

## License

Distributed under the MIT License. See LICENSE for more information.
-->