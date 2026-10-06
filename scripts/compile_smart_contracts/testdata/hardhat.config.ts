import { defineConfig } from "hardhat/config";

export default defineConfig({
  solidity: {
    version: "0.8.20",
    // Hardhat 2 compiled 0.8.20 for paris by default, and the committed
    // artifacts were built that way. Hardhat 3 takes solc's own default,
    // shanghai, whose PUSH0 opcode the EVM only runs with EIP-3855 enabled.
    settings: { evmVersion: "paris" },
  },
  paths: {
    sources: "./solidity",
  },
});
