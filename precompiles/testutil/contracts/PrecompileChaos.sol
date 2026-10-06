// SPDX-License-Identifier: LGPL-v3
pragma solidity >=0.8.17;

/// @dev Fixture for the precompile invariant property test. It runs a program of
/// steps - native value transfers, raw calls to precompiles or any other address,
/// and nested sub-programs that may revert - so the test can drive arbitrary
/// interleavings of EVM state changes, precompile calls and reverts at any call
/// depth, and check that none of them breaks the chain's accounting.
contract PrecompileChaos {
    struct Step {
        // 0 = send native value, 1 = raw call, 2 = nested program
        uint8 kind;
        address target;
        uint256 value;
        // calldata of a raw call, abi.encode(Step[], bool) of a nested program
        bytes data;
        // gas forwarded to a raw call, 0 forwards all of it
        uint64 gas;
    }

    error ProgramReverted();

    receive() external payable {}

    /// @dev Runs the steps in order and, if revertAtEnd is set, reverts the whole
    /// frame afterwards. Failed transfers and calls are ignored, and so are reverts
    /// of nested programs, which run in a call frame of their own.
    function run(Step[] calldata steps, bool revertAtEnd) external payable {
        for (uint256 i = 0; i < steps.length; i++) {
            Step calldata step = steps[i];
            if (step.kind == 0) {
                (bool sent, ) = payable(step.target).call{value: step.value}("");
                sent;
            } else if (step.kind == 1) {
                uint256 gasLimit = step.gas == 0 ? gasleft() : step.gas;
                (bool ok, ) = step.target.call{value: step.value, gas: gasLimit}(step.data);
                ok;
            } else {
                (Step[] memory inner, bool innerRevert) = abi.decode(step.data, (Step[], bool));
                try this.run(inner, innerRevert) {} catch {}
            }
        }
        if (revertAtEnd) {
            revert ProgramReverted();
        }
    }
}
