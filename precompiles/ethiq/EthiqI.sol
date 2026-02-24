// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.18;

import "../common/Types.sol";
import "../authorization/AuthorizationI.sol";

/// @dev The ETHIQ contract's address.
address constant ETHIQ_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000900;

/// @dev The ETHIQ contract's instance.
EthiqI constant ETHIQ_CONTRACT = EthiqI(ETHIQ_PRECOMPILE_ADDRESS);

/// @dev Define all the available methods.
string constant MSG_MINT_HAQQ = "/haqq.ethiq.v1.MsgMintHaqq";
string constant MSG_MINT_HAQQ_BY_APPLICATION = "/haqq.ethiq.v1.MsgMintHaqqByApplication";

/// @author Haqq Team
/// @title Ethiq Precompile Contract
/// @dev The interface through which solidity contracts will interact with the ethiq module
/// @custom:address 0x0000000000000000000000000000000000000900
interface EthiqI is AuthorizationI {
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

    /// @dev MintHaqqByApplication defines an Event emitted when HAQQ coins are minted by application ID
    /// @param sender The address that sent the transaction
    /// @param receiver The address that received the minted aHAQQ coins
    /// @param applicationId The application ID used for minting
    /// @param haqqAmount The amount of aHAQQ coins minted
    event MintHaqqByApplication(
        address indexed sender,
        address indexed receiver,
        uint256 applicationId,
        uint256 haqqAmount
    );

    /// TRANSACTIONS

    /// @dev Mints aHAQQ coins in exchange for aISLM coins
    /// @param sender The address that will send the aISLM coins
    /// @param receiver The address that will receive the minted aHAQQ coins
    /// @param islmAmount The amount of aISLM coins to burn
    /// @return haqqAmount The amount of aHAQQ coins minted
    function mintHaqq(
        address sender,
        address receiver,
        uint256 islmAmount
    ) external returns (uint256 haqqAmount);

    /// @dev Mints aHAQQ coins by application ID
    /// @param sender The address that will send the transaction
    /// @param applicationId The application ID to use for minting
    /// @return haqqAmount The amount of aHAQQ coins minted
    function mintHaqqByApplication(
        address sender,
        uint256 applicationId
    ) external returns (uint256 haqqAmount);

    /// @dev Approves an application ID for minting HAQQ coins
    /// @param grantee The contract address which will have authorization to mint
    /// @param applicationId The application ID to approve
    /// @param methods The message type URLs of the methods to approve
    /// @return approved Boolean value to indicate if the approval was successful
    function approveApplicationID(
        address grantee,
        uint256 applicationId,
        string[] calldata methods
    ) external returns (bool approved);

    /// @dev Revokes an application ID authorization
    /// @param grantee The contract address which will have its authorization revoked
    /// @param applicationId The application ID to revoke
    /// @param methods The message type URLs of the methods to revoke
    /// @return revoked Boolean value to indicate if the revocation was successful
    function revokeApplicationID(
        address grantee,
        uint256 applicationId,
        string[] calldata methods
    ) external returns (bool revoked);

    /// QUERIES

    /// @dev Calculates the estimated amount of aHAQQ coins to be minted for a given aISLM amount
    /// @param islmAmount The amount of aISLM coins to burn
    /// @return estimatedHaqqAmount The estimated amount of aHAQQ coins to be minted
    /// @return supplyBefore The supply of aHAQQ before minting
    /// @return supplyAfter The supply of aHAQQ after minting
    /// @return pricePerUnit The price per unit of aHAQQ as a decimal string
    function calculate(
        uint256 islmAmount
    )
        external
        view
        returns (
            uint256 estimatedHaqqAmount,
            uint256 supplyBefore,
            uint256 supplyAfter,
            string memory pricePerUnit
        );
}
