// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.17;

/// @dev Makes a Cosmos module account dirty in the EVM StateDB without moving any
/// balance, then calls a precompile in the same transaction.
///
/// A zero-value CALL is the only way to reach a module account's state object: a
/// value-bearing one makes StateDB.Commit try to *credit* the module account, which
/// bank refuses because module accounts are blocked recipients. A zero-value CALL
/// touches the object instead, and an account that is empty (no balance, no nonce, no
/// code) is journalled as dirty by that touch alone.
contract ModuleAccountToucher {
    /// @dev Plain forward, used to set up state in an earlier transaction.
    function forward(address target, bytes calldata data) external {
        (bool ok, ) = target.call(data);
        require(ok, "forward failed");
    }

    /// @dev Zero-value touch of `account`, then a call into `target`, in one EVM tx.
    function touchZeroAndForward(address account, address target, bytes calldata data) external {
        (bool touched, ) = account.call{value: 0}("");
        require(touched, "touch failed");
        (bool ok, ) = target.call(data);
        require(ok, "forward failed");
    }

    /// @dev Precompile call first, value-bearing touch of `account` second, in one EVM tx.
    ///
    /// Order is the whole point. The precompile writes through the StateDB's cacheCtx,
    /// while getStateObject reads balances from s.ctx, so a module account credited by
    /// the precompile still reads as its pre-call balance. Touching it afterwards
    /// journals that stale value, and commit reconciles the stale view against the real
    /// bank balance.
    function forwardThenTouch(address target, bytes calldata data, address account) external payable {
        (bool ok, ) = target.call(data);
        require(ok, "forward failed");
        require(msg.value >= 1, "need 1 wei");
        payable(account).transfer(1);
    }
}
