// SPDX-License-Identifier: MIT
pragma solidity 0.8.18;

/*
* @title HaqqTesting
* This smart contract allows making deposits and withdrawals for testing purposes.
*/
contract HaqqTesting {
    uint256 public constant MAX_DEPOSITS = 1;

    mapping(address => uint256) public depositsCounter;

    struct Deposit {
        uint256 sumDeposited;
        uint256 sumPaidAlready;
    }

    /// @dev beneficiary address => deposit
    mapping(address => Deposit) public deposits;

    /// @dev Event to be emitted, when deposit was made.
    event DepositMade (
        address indexed beneficiaryAddress,
        uint256 indexed depositId,
        uint256 sumDeposited,
        address depositedBy
    );

    /// @dev Event that will be emitted, when withdrawal was made
    event WithdrawalMade (
        address indexed beneficiary,
        uint256 sumWithdrawn
    );

    /// @dev Function to make a new deposit.
    /// @param _beneficiaryAddress address that will receive payments from this deposit
    function deposit(address _beneficiaryAddress) external payable returns (bool success) {
        require(_beneficiaryAddress != address(0), "HaqqVesting: beneficiary address is zero");
        require(msg.value > 0, "HaqqVesting: deposit sum is zero");
        // new deposit id for this deposit
        uint256 depositId = ++depositsCounter[_beneficiaryAddress];
        require(depositId <= MAX_DEPOSITS, "Max deposit number for this address reached");

        // make records to this deposit:
        deposits[_beneficiaryAddress] = Deposit({sumDeposited: msg.value, sumPaidAlready: 0});

        // emit event:
        emit DepositMade(
            _beneficiaryAddress,
            depositId,
            msg.value,
            msg.sender
        );

        return true;
    }

    function withdraw(uint256 sumToWithdraw) external returns (bool success) {
        uint256 allowedToWithdraw;
        require(depositsCounter[msg.sender] > 0, "No deposits for this address");
        require(sumToWithdraw > 0, "Sum to withdraw should be > 0");

        allowedToWithdraw = deposits[msg.sender].sumDeposited - deposits[msg.sender].sumPaidAlready;
        require(allowedToWithdraw > 0, "Nothing to withdraw");
        require(sumToWithdraw <= allowedToWithdraw, "Not enough funds to withdraw");

        deposits[msg.sender].sumPaidAlready = deposits[msg.sender].sumPaidAlready + sumToWithdraw;

        // payable(_beneficiaryAddress).transfer(sumToWithdraw);
        // changed to .call
        // see https://solidity-by-example.org/sending-ether/
        (bool sent,) = payable(msg.sender).call{value: sumToWithdraw}("");
        require(sent, "Failed to send tokens");

        emit WithdrawalMade(
            msg.sender,
            sumToWithdraw
        );

        return true;
    }
}
