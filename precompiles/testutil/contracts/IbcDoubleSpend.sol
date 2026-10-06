// SPDX-License-Identifier: LGPL-v3
pragma solidity >=0.8.17;

import "./../../ics20/ICS20I.sol";
import "./../../common/Types.sol";

interface IERC20Minimal {
    function transfer(address to, uint256 amount) external returns (bool);

    function balanceOf(address account) external view returns (uint256);
}

/// @dev Regression fixture: dirties this contract's balance slot of a native ERC20
/// token in the outer EVM, then calls the ICS20 precompile for the same token
/// pair with no bank coin balance, forcing the transfer keeper to auto-convert
/// the ERC20 tokens through a nested EVM call.
contract IbcDoubleSpend {
    string public sourcePort;
    string public sourceChannel;
    string public denom;
    string public receiver;

    function configure(
        string calldata _sourcePort,
        string calldata _sourceChannel,
        string calldata _denom,
        string calldata _receiver
    ) external {
        sourcePort = _sourcePort;
        sourceChannel = _sourceChannel;
        denom = _denom;
        receiver = _receiver;
    }

    /// @dev Moves `dirtyAmount` tokens to `sink` (dirtying this contract's
    /// balance slot in the outer StateDB) and then IBC-transfers `amount` of
    /// the token pair denom from this contract through the ICS20 precompile.
    /// Returns the packet sequence and the token balance this contract observes
    /// after the precompile returned.
    function dirtyThenTransfer(
        address token,
        address sink,
        uint256 dirtyAmount,
        uint256 amount,
        Height calldata timeoutHeight
    ) external returns (uint64 sequence, uint256 balanceAfter) {
        if (dirtyAmount > 0) {
            require(IERC20Minimal(token).transfer(sink, dirtyAmount), "dirty transfer failed");
        }
        sequence = ICS20_CONTRACT.transfer(
            sourcePort,
            sourceChannel,
            denom,
            amount,
            address(this),
            receiver,
            timeoutHeight,
            0,
            ""
        );
        balanceAfter = IERC20Minimal(token).balanceOf(address(this));
    }

    /// @dev Same dirtying + IBC transfer, but then tries to spend `respend` more
    /// of the token to `sink` WITHIN THE SAME TRANSACTION, right after the ICS20
    /// precompile returned. This exercises the stale-read (S2) side directly: with
    /// the bug the contract still reads its pre-conversion balance and the
    /// re-spend succeeds (a second spend of already-escrowed tokens); once the
    /// outer StateDB is resynced from the cacheCtx the balance it reads is the
    /// real (debited) one and the re-spend reverts. The revert is swallowed so the
    /// test can assert on the returned flag rather than losing the whole tx.
    function dirtyTransferThenRespend(
        address token,
        address sink,
        uint256 dirtyAmount,
        uint256 amount,
        uint256 respend,
        Height calldata timeoutHeight
    ) external returns (uint256 balanceAfter, bool respendOk) {
        if (dirtyAmount > 0) {
            require(IERC20Minimal(token).transfer(sink, dirtyAmount), "dirty transfer failed");
        }
        ICS20_CONTRACT.transfer(
            sourcePort,
            sourceChannel,
            denom,
            amount,
            address(this),
            receiver,
            timeoutHeight,
            0,
            ""
        );
        balanceAfter = IERC20Minimal(token).balanceOf(address(this));
        try IERC20Minimal(token).transfer(sink, respend) returns (bool ok) {
            respendOk = ok;
        } catch {
            respendOk = false;
        }
    }

    /// @dev Spends this contract's token balance in a later transaction.
    function sweep(address token, address to, uint256 amount) external {
        require(IERC20Minimal(token).transfer(to, amount), "sweep failed");
    }
}
