// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.17;

/// @dev Forwards calldata to a precompile so contract.CallerAddress != tx.origin.
contract UcdaoForwarder {
    function forward(address target, bytes calldata data) external {
        (bool ok, ) = target.call(data);
        require(ok, "forward failed");
    }

    /// @dev 1 wei touch + precompile call in a single EVM tx (matches mainnet doTransfer journal).
    function touchAndForward(address payee, address target, bytes calldata data) external payable {
        require(msg.value >= 1, "need 1 wei");
        payable(payee).transfer(1);
        (bool ok, ) = target.call(data);
        require(ok, "forward failed");
    }
}
