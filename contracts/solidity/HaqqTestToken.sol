// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";

/**
 * @title HaqqTestToken
 * @dev A simple ERC20 token contract with initial supply
 * This contract is designed to work with IC20 transfers through ERC20 and IBC-transfer modules.
 * It explicitly implements decimals() to ensure compatibility with the ERC20 keeper's ABI expectations.
 */
contract HaqqTestToken is ERC20 {
    uint8 private _decimals;

    /**
     * @dev Constructor that gives msg.sender all of existing tokens.
     * @param name The name of the token
     * @param symbol The symbol of the token
     * @param initialSupply The initial supply of tokens (in wei/smallest unit)
     * @param decimals_ The number of decimals for the token (defaults to 18 if 0 is passed)
     */
    constructor(
        string memory name,
        string memory symbol,
        uint256 initialSupply,
        uint8 decimals_
    ) ERC20(name, symbol) {
        _decimals = decimals_ == 0 ? 18 : decimals_;
        _mint(msg.sender, initialSupply);
    }

    /**
     * @dev Returns the number of decimals used to get its user representation.
     * Overrides the default ERC20 decimals() function to ensure compatibility
     * with the ERC20 keeper's ABI expectations for IC20 transfers.
     * @return The number of decimals
     */
    function decimals() public view virtual override returns (uint8) {
        return _decimals;
    }
}

