// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.18;

import "../authorization/AuthorizationI.sol";

/// @dev The UCDAO contract's address.
address constant UCDAO_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000901;

/// @dev The UCDAO contract's instance.
UcdaoI constant UCDAO_CONTRACT = UcdaoI(UCDAO_PRECOMPILE_ADDRESS);

/// @dev Define all the available methods.
string constant MSG_CONVERT_TO_HAQQ = "/haqq.ucdao.v1.MsgConvertToHaqq";
string constant MSG_TRANSFER_OWNERSHIP = "/haqq.ucdao.v1.MsgTransferOwnershipWithAmount";

/// @author Haqq Team
/// @title UCDAO Precompile Contract
/// @dev The interface through which solidity contracts will interact with the ucdao module
/// @custom:address 0x0000000000000000000000000000000000000901
interface UcdaoI is AuthorizationI {
  /// @dev MintHaqq defines an Event emitted when HAQQ coins are minted
  /// @param sender The address that sent the aISLM coins
  /// @param receiver The address that received the minted aHAQQ coins
  /// @param islmAmount The amount of aISLM coins burned
  /// @param haqqAmount The amount of aHAQQ coins minted
  event MintHaqq(
    address indexed sender,
    address indexed receiver,
    uint256 islmAmount,
    uint256 haqqAmount
  );

  /// TRANSACTIONS

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

