// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.17;

import "../EthiqI.sol" as ethiq;

/// @title EthiqCaller
/// @author Haqq Network Core Team
/// @dev This contract is used to test external contract calls to the ethiq precompile.
contract EthiqCaller {
    /// counter is used to test the state persistence bug, when EVM and Cosmos state were both
    /// changed in the same function.
    uint256 public counter;
    string[] private mintHaqqMethod = [ethiq.MSG_MINT_HAQQ];

    /// @dev This function calls the ethiq precompile's approve method.
    /// @param _addr The address to approve.
    /// @param _methods The methods to approve.
    /// @param _amount The amount to grant.
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

    /// @dev This function calls the ethiq precompile's revoke method.
    /// @param _grantee The address that was approved to spend the funds.
    /// @param _methods The methods to revoke.
    function testRevoke(
        address _grantee,
        string[] calldata _methods
    ) public {
        bool success = ethiq.ETHIQ_CONTRACT.revoke(_grantee, _methods);
        require(success, "Failed to revoke approval for ethiq methods");
    }

    /// @dev This function calls the ethiq precompile's mintHaqq method.
    /// @param _from_addr The address to approve.
    /// @param _to_addr The mint receiver address.
    /// @param _amount The amount to burn.
    function testMintHaqq(
        address _from_addr,
        address _to_addr,
        uint256 _amount
    ) public {
        counter++;
        ethiq.ETHIQ_CONTRACT.mintHaqq(_from_addr, _to_addr, _amount);
        counter--;
    }

    /// @dev This function calls the ethiq precompile's allowance query method.
    /// @param _grantee The address that received the grant.
    /// @param method The method to query.
    /// @return allowance The allowance.
    function getAllowance(
        address _grantee,
        string memory method
    ) public view returns (uint256 allowance) {
        return ethiq.ETHIQ_CONTRACT.allowance(_grantee, msg.sender, method);
    }

    /// @dev This function calls the ethiq precompile's approve method to grant approval for a burn/mint.
    /// Next, the mintHaqq method is called to execute an burn/mint.
    /// @param _addr The address to approve.
    /// @param _approveAmount The amount to approve.
    /// @param _burnAmount The amount to burn.
    /// @param _to_addr The address to mint coins to.
    function testApproveAndThenMintHaqq(
        address _addr,
        uint256 _approveAmount,
        uint256 _burnAmount,
        address _to_addr
    ) public {
        string[] memory approvedMethods = new string[](1);
        approvedMethods[0] = ethiq.MSG_MINT_HAQQ;
        bool success = ethiq.ETHIQ_CONTRACT.approve(
            _addr,
            _approveAmount,
            approvedMethods
        );
        require(success, "failed to approve mintHaqq method");
        ethiq.ETHIQ_CONTRACT.mintHaqq(
            tx.origin,
            _to_addr,
            _burnAmount
        );
    }

    /// @dev This function is used to test the behaviour when executing transactions using special
    /// function calling opcodes,
    /// like call, delegatecall, staticcall, and callcode.
    /// @param _addr The address to approve.
    /// @param _to_addr The address to mint coins to.
    /// @param _burnAmount The amount to burn.
    /// @param _calltype The opcode to use.
    function testCallMintHaqq(
        address _addr,
        address _to_addr,
        uint256 _burnAmount,
        string memory _calltype
    ) public returns (uint256 minted) {
        address calledContractAddress = ethiq.ETHIQ_PRECOMPILE_ADDRESS;
        bytes memory payload = abi.encodeWithSignature(
            "mintHaqq(address,address,uint256)",
            _addr,
            _to_addr,
            _burnAmount
        );
        bytes32 calltypeHash = keccak256(abi.encodePacked(_calltype));

        if (calltypeHash == keccak256(abi.encodePacked("delegatecall"))) {
            (bool success, bytes memory data) = calledContractAddress
                .delegatecall(payload);
            require(success, "failed delegatecall to precompile");
            minted = abi.decode(data, (uint256));
        } else if (calltypeHash == keccak256(abi.encodePacked("staticcall"))) {
            (bool success, bytes memory data) = calledContractAddress
                .staticcall(payload);
            require(success, "failed staticcall to precompile");
            minted = abi.decode(data, (uint256));
        } else if (calltypeHash == keccak256(abi.encodePacked("call"))) {
            (bool success, bytes memory data) = calledContractAddress.call(
                payload
            );
            require(success, "failed call to precompile");
            minted = abi.decode(data, (uint256));
        } else if (calltypeHash == keccak256(abi.encodePacked("callcode"))) {
            //Function signature
            bytes4 sig = bytes4(keccak256(bytes("mintHaqq(address,address,uint256)")));
            // Length of the input data is 164 bytes on 32bytes chunks:
            //                          Memory location
            // 0 - 4 byte signature     x
            // 1 - 0x0000..address		x + 0x04
            // 2 - 0x0000..to_address	x + 0x24
            // 3 - 0x0000..burnAmt		x + 0x44
            uint256 len = 100;
            // Coin type includes denom & amount
            // need to get these separately from the bytes response

            // NOTE: callcode is deprecated and now only available via inline assembly
            assembly {
                // Load the function signature and argument data onto the stack
                let x := mload(0x40) // Find empty storage location using "free memory pointer"
                mstore(x, sig) // Place function signature at beginning of empty storage
                mstore(add(x, 0x04), _addr)    // Place the address (input param) after the function sig
                mstore(add(x, 0x24), _to_addr) // Place the receiver's address (input param)
                mstore(add(x, 0x44), _burnAmount) // Place the amount to burn (input param)

                // Invoke the contract at calledContractAddress in the context of the current contract
                // using CALLCODE opcode and the loaded function signature and argument data
                let success := callcode(
                    gas(),
                    calledContractAddress, // to addr
                    0, // no value
                    x, // inputs are stored at location x
                    len, // inputs length
                    x, //store output over input (saves space)
                    0x20 // output length for this call
                )

                // output length for this call is 32 bytes
                minted := mload(x) // Assign minted amount output value - 32 bytes long

                // Check if the call was successful and revert the transaction if it failed
                if iszero(success) {
                    revert(0, 0)
                }
            }
        } else {
            revert("invalid calltype");
        }

        return minted;
    }

    /// @dev This function showcased, that there was a bug in the EVM implementation, that occurred when
    /// Cosmos state is modified in the same transaction as state information inside
    /// the EVM.
    /// @param _to_addr The address to mint coins to.
    /// @param _burnAmount The amount to burn.
    function testMintHaqqIncrementCounter(
        address _to_addr,
        uint256 _burnAmount
    ) public {
        bool successApprove = ethiq.ETHIQ_CONTRACT.approve(
            address(this),
            _burnAmount,
            mintHaqqMethod
        );
        require(successApprove, "Mint Approve failed");
        ethiq.ETHIQ_CONTRACT.mintHaqq(
            address(this),
            _to_addr,
            _burnAmount
        );
        counter += 1;
    }

    /// @dev This function showcases the possibility to deposit into the contract
    /// and immediately burn/mint to a receiver address using the same balance in the same transaction.
    function approveDepositAndMintHaqq(
        address _to_addr
    ) public payable {
        bool successTx = ethiq.ETHIQ_CONTRACT.approve(
            address(this),
            msg.value,
            mintHaqqMethod
        );
        require(successTx, "Mint Approve failed");
        ethiq.ETHIQ_CONTRACT.mintHaqq(
            address(this),
            _to_addr,
            msg.value
        );
    }

    /// @dev This function is suppose to fail because the amount to burn is
    /// higher than the amount approved.
    function approveDepositAndMintHaqqExceedingAllowance(
        address _to_addr
    ) public payable {
        bool successTx = ethiq.ETHIQ_CONTRACT.approve(
            tx.origin,
            msg.value,
            mintHaqqMethod
        );
        require(successTx, "Mint Approve failed");
        ethiq.ETHIQ_CONTRACT.mintHaqq(
            address(this),
            _to_addr,
            msg.value + 1
        );
    }

    /// @dev This function is suppose to fail because the amount to burn is
    /// higher than the amount approved.
    function approveDepositMintHaqqAndFailCustomLogic(
        address _to_addr
    ) public payable {
        bool successTx = ethiq.ETHIQ_CONTRACT.approve(
            tx.origin,
            msg.value,
            mintHaqqMethod
        );
        require(successTx, "Mint Approve failed");
        ethiq.ETHIQ_CONTRACT.mintHaqq(
            address(this),
            _to_addr,
            msg.value
        );
        // This should fail since the balance is already spent in the previous call
        payable(msg.sender).transfer(msg.value);
    }
}
