// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.17;

import "../EthiqI.sol" as ethiq;

/// @title EthiqCallerTwo
/// @author Haqq Network Core Team
/// @dev This contract is used to test external contract calls to the ethiq precompile.
contract EthiqCallerTwo {
    /// counter is used to test the state persistence bug, when EVM and Cosmos state were both
    /// changed in the same function.
    uint256 public counter;

    /// @dev This function calls the ethiq precompile's approve method.
    /// @param _addr The address to approve.
    /// @param _methods The methods to approve.
    function testApprove(
        address _addr,
        string[] calldata _methods,
        uint256 _amount
    ) public {
        bool success = ethiq.ETHIQ_CONTRACT.approve(
            _addr,
            _amount,
            _methods
        );
        require(success, "Failed to approve ethiq methods");
    }

    /// @dev This function showcased, that there was a bug in the EVM implementation, that occurred when
    /// Cosmos state is modified in the same transaction as state information inside
    /// the EVM.
    /// @param _addr Address of the burner
    /// @param _amount Amount to burn
    /// @param _before Boolean to specify if funds should be transferred to burner before the precompile call
    /// @param _after Boolean to specify if funds should be transferred to burner after the precompile call
    function testMintHaqqWithCounterAndTransfer(
        address payable _addr,
        uint256 _amount,
        bool _before,
        bool _after
    ) public {
        if (_before) {
            counter++;
            (bool sent, ) = _addr.call{value: 15}("");
            require(sent, "Failed to send Ether to burner");
        }
        ethiq.ETHIQ_CONTRACT.mintHaqq(_addr, _addr, _amount);
        if (_after) {
            counter++;
            (bool sent, ) = _addr.call{value: 15}("");
            require(sent, "Failed to send Ether to burner");
        }
    }

    /// @dev This function showcased, that there was a bug in the EVM implementation, that occurred when
    /// Cosmos state is modified in the same transaction as state information inside
    /// the EVM.
    /// @param _dest Address to send some funds from the contract
    /// @param _burner Address of the burner
    /// @param _amount Amount to burn
    /// @param _before Boolean to specify if funds should be transferred to burner before the precompile call
    /// @param _after Boolean to specify if funds should be transferred to burner after the precompile call
    function testMintHaqqWithTransfer(
        address payable _dest,
        address payable _burner,
        uint256 _amount,
        bool _before,
        bool _after
    ) public {
        if (_before) {
            counter++;
            (bool sent, ) = _dest.call{value: 15}("");
            require(sent, "Failed to send Ether to burner");
        }
        ethiq.ETHIQ_CONTRACT.mintHaqq(_burner, _burner, _amount);
        if (_after) {
            counter++;
            (bool sent, ) = _dest.call{value: 15}("");
            require(sent, "Failed to send Ether to burner");
        }
    }
}
