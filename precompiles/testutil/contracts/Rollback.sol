// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Rollback drives the StateDB paths PR #460 flagged: a
// stateful precompile's RunSetup flushes the dirty EVM state into the cache
// context before it validates the selector, and nothing journals that flush
// when the precompile then fails.
contract Rollback {
    address constant STAKING = 0x0000000000000000000000000000000000000800;
    bytes4 constant BOGUS = 0xdeadbeef;
    string constant ROLLED_BACK = "rolled back";

    constructor() payable {}

    receive() external payable {}

    // payThenPoke pays `amount` to `recipient`, optionally calls the staking
    // precompile with a selector it does not know, and then always reverts, so
    // the payment must be undone.
    function payThenPoke(
        address payable recipient,
        uint256 amount,
        bool poke
    ) external {
        require(msg.sender == address(this), "self only");

        (bool sent, ) = recipient.call{value: amount}("");
        require(sent, "payment failed");

        if (poke) {
            (bool ok, ) = STAKING.call{gas: 200_000}(
                abi.encodeWithSelector(BOGUS)
            );
            require(!ok, "bogus selector accepted");
        }

        revert(ROLLED_BACK);
    }

    // rollbackThenPay runs payThenPoke in a child frame, catches its revert and
    // then moves 1 wei of its own so its balance is dirty at commit time.
    function rollbackThenPay(
        address payable recipient,
        uint256 amount,
        address payable sink,
        bool poke
    ) external {
        try this.payThenPoke(recipient, amount, poke) {
            revert("child must revert");
        } catch Error(string memory reason) {
            require(
                keccak256(bytes(reason)) == keccak256(bytes(ROLLED_BACK)),
                reason
            );
        }

        (bool sent, ) = sink.call{value: 1}("");
        require(sent, "sink payment failed");
    }

    // valueToPrecompile sends native value along with a call to the staking
    // precompile and swallows the failure.
    function valueToPrecompile(uint256 amount) external {
        (bool ok, ) = STAKING.call{value: amount, gas: 200_000}(
            abi.encodeWithSelector(BOGUS)
        );
        require(!ok, "value call to precompile succeeded");
    }
}
