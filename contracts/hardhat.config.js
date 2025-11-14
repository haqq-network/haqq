require("@nomicfoundation/hardhat-ethers");

/** @type import('hardhat/config').HardhatUserConfig */
module.exports = {
  solidity: '0.8.20',
  paths: {
    sources: './solidity'
  },
  networks: {
    localhost: {
      url: 'http://127.0.0.1:8545',
      chainId: 121799,
      ...(process.env.PRIVATE_KEY && { accounts: [process.env.PRIVATE_KEY] })
    }
  }
}
