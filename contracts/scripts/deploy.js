const hre = require("hardhat");

// Pre-defined wallet address to receive all tokens
const RECIPIENT_WALLET = "[YOUR_WALLET_ADDRESS]";

async function main() {
  console.log("Deploying HaqqTestToken contract...\n");

  // Get the deployer account
  const [deployer] = await hre.ethers.getSigners();
  if (!deployer) {
    throw new Error("No deployer account found. Make sure your network is configured correctly.");
  }

  const deployerBalance = await hre.ethers.provider.getBalance(deployer.address);
  console.log("Deploying contracts with account:", deployer.address);
  console.log("Account balance:", hre.ethers.formatEther(deployerBalance), "ETH\n");

  if (deployerBalance === 0n) {
    throw new Error("Deployer account has no balance. Please fund the account.");
  }

  // Contract parameters
  const tokenName = "Haqq Test Token";
  const tokenSymbol = "HTT";
  const initialSupply = hre.ethers.parseEther("1000000"); // 1,000,000 tokens with 18 decimals

  console.log("Contract parameters:");
  console.log("  Name:", tokenName);
  console.log("  Symbol:", tokenSymbol);
  console.log("  Initial Supply:", hre.ethers.formatEther(initialSupply), tokenSymbol);
  console.log("\nDeploying contract...\n");

  // Deploy the contract
  const HaqqTestToken = await hre.ethers.getContractFactory("HaqqTestToken");
  const haqqTestToken = await HaqqTestToken.deploy(tokenName, tokenSymbol, initialSupply);

  console.log("Waiting for deployment transaction to be mined...");
  await haqqTestToken.waitForDeployment();
  const contractAddress = await haqqTestToken.getAddress();

  console.log("\n✅ Deployment successful!\n");
  console.log("Contract Address:", contractAddress);
  console.log("Token Name:", await haqqTestToken.name());
  console.log("Token Symbol:", await haqqTestToken.symbol());
  console.log("Decimals:", await haqqTestToken.decimals());
  console.log("\nDeployer address:", deployer.address);
  console.log("Total Supply:", hre.ethers.formatEther(await haqqTestToken.totalSupply()), tokenSymbol);
  const deployerTokenBalance = await haqqTestToken.balanceOf(deployer.address);
  console.log("Deployer Balance:", hre.ethers.formatEther(deployerTokenBalance), tokenSymbol);
  console.log("\nContract verified on network!");

  // Transfer all tokens to the pre-defined wallet address
  if (deployerTokenBalance > 0n) {
    console.log("\n" + "=".repeat(60));
    console.log("Transferring all tokens to recipient wallet...");
    console.log("Recipient Address:", RECIPIENT_WALLET);
    console.log("Amount to transfer:", hre.ethers.formatEther(deployerTokenBalance), tokenSymbol);
    
    const transferTx = await haqqTestToken.transfer(RECIPIENT_WALLET, deployerTokenBalance);
    console.log("Transfer transaction hash:", transferTx.hash);
    console.log("Waiting for transaction to be mined...");
    
    await transferTx.wait();
    
    console.log("\n✅ Transfer successful!\n");
    console.log("Recipient Balance:", hre.ethers.formatEther(await haqqTestToken.balanceOf(RECIPIENT_WALLET)), tokenSymbol);
    console.log("Deployer Balance:", hre.ethers.formatEther(await haqqTestToken.balanceOf(deployer.address)), tokenSymbol);
  } else {
    console.log("\n⚠️  No tokens to transfer. Deployer balance is zero.");
  }
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });

