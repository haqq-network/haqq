// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.18;

import "../common/Types.sol";
import "../authorization/IICS20Authorization.sol";

/// @dev The ICS20I contract's address.
address constant ICS20_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000802;

/// @dev The ICS20 contract's instance.
ICS20I constant ICS20_CONTRACT = ICS20I(ICS20_PRECOMPILE_ADDRESS);

/// @dev Denom contains the base denomination for ICS20 fungible tokens and the
/// source tracing information path.
struct Denom {
    /// base denomination of the relayed fungible token.
    string base;
    /// trace contains a list of hops for multi-hop transfers.
    Hop[] trace;
}

/// @dev Hop defines a port ID, channel ID pair specifying where
/// tokens must be forwarded next in a multi-hop transfer.
struct Hop {
    string portId;
    string channelId;
}

/// @author Evmos Team
/// @title ICS20 Transfer Precompiled Contract
/// @dev The interface through which solidity contracts will interact with IBC Transfer (ICS20)
/// @custom:address 0x0000000000000000000000000000000000000802
interface ICS20I is IICS20Authorization {
    /// @dev Emitted when an ICS-20 transfer is executed.
    /// @param sender The address of the sender.
    /// @param receiver The address of the receiver.
    /// @param sourcePort The source port of the IBC transaction.
    /// @param sourceChannel The source channel of the IBC transaction.
    /// @param denom The denomination of the tokens transferred.
    /// @param amount The amount of tokens transferred.
    /// @param memo The IBC transaction memo.
    event IBCTransfer(
        address indexed sender,
        string indexed receiver,
        string sourcePort,
        string sourceChannel,
        string denom,
        uint256 amount,
        string memo
    );

    /// @dev Transfer defines a method for performing an IBC transfer.
    /// @param sourcePort the port on which the packet will be sent
    /// @param sourceChannel the channel by which the packet will be sent
    /// @param denom the denomination of the Coin to be transferred to the receiver
    /// @param amount the amount of the Coin to be transferred to the receiver
    /// @param sender the hex address of the sender
    /// @param receiver the bech32 address of the receiver
    /// @param timeoutHeight the timeout height relative to the current block height.
    /// The timeout is disabled when set to 0
    /// @param timeoutTimestamp the timeout timestamp in absolute nanoseconds since unix epoch.
    /// The timeout is disabled when set to 0
    /// @param memo optional memo
    /// @return nextSequence sequence number of the transfer packet sent
    function transfer(
        string memory sourcePort,
        string memory sourceChannel,
        string memory denom,
        uint256 amount,
        address sender,
        string memory receiver,
        Height memory timeoutHeight,
        uint64 timeoutTimestamp,
        string memory memo
    ) external returns (uint64 nextSequence);

    /// @dev denoms Defines a method for returning all denoms.
    /// @param pageRequest Defines the pagination parameters to for the request.
    function denoms(
        PageRequest memory pageRequest
    )
        external
        view
        returns (
            Denom[] memory denoms,
            PageResponse memory pageResponse
        );

    /// @dev Denom defines a method for returning a denom.
    function denom(
        string memory hash
    ) external view returns (Denom memory denom);

    /// @dev DenomHash defines a method for returning a hash of the denomination info.
    function denomHash(
        string memory trace
    ) external view returns (string memory hash);

}
