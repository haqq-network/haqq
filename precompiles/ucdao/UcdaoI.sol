// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.18;

/// @dev The UCDAO contract's address.
address constant UCDAO_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000901;

/// @dev The UCDAO contract's instance.
UcdaoI constant UCDAO_CONTRACT = UcdaoI(UCDAO_PRECOMPILE_ADDRESS);

/**
 * @title UCDAO Interface
 * @dev Interface for interacting with the ucdao module via precompile.
 */
interface UcdaoI {
  /// @dev ConvertToHaqq allows a holder to convert aISLM tokens to aHAQQ tokens.
  /// @param sender the hex address of the sender (ucdao holder)
  /// @param receiver the bech32-mapped recipient address as hex
  /// @param islmAmount the amount of aISLM to convert
  /// @return haqqAmount the amount of aHAQQ minted
  function convertToHaqq(
    address sender,
    address receiver,
    uint256 islmAmount
  ) external returns (uint256 haqqAmount);

  /// @dev TransferOwnership transfers all DAO balances from the owner to the new owner.
  /// @param owner the current owner (ucdao holder)
  /// @param newOwner the new owner
  function transferOwnership(
    address owner,
    address newOwner
  ) external;

  /// @dev TransferOwnershipWithAmount transfers a specific set of DAO balances from the owner
  /// to the new owner.
  /// @param owner the current owner (ucdao holder)
  /// @param newOwner the new owner
  /// @param denoms the list of Cosmos denoms to transfer
  /// @param amounts the corresponding amounts per denom
  function transferOwnershipWithAmount(
    address owner,
    address newOwner,
    string[] memory denoms,
    uint256[] memory amounts
  ) external;
}

