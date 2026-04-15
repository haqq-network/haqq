// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import "../../ethiq/EthiqI.sol";

contract EthiqReverter {
    uint counter = 0;

    constructor() payable {}

    function run(uint numTimes, address toAddress) external {
        counter++;

        for (uint i = 0; i < numTimes; i++) {
            try
            EthiqReverter(address(this)).performMintHaqq(
                    toAddress
                )
            {} catch {}
        }
    }

    function multipleBurnMints(
        uint numTimes,
        address toAddress
    ) external {
        counter++;

        for (uint i = 0; i < numTimes; i++) {
            EthiqReverter(address(this)).performMintHaqq(toAddress);
        }
    }

    function performMintHaqq(address toAddress) external {
        ETHIQ_CONTRACT.mintHaqq(address(this), toAddress, 1000);
        revert();
    }
}
