<!--
order: 1
-->
# Concepts

## Liquidation

Users with vesting accounts can make their locked ISLM tokens liquid. If vesting account contains only locked tokens user can use `Liquidate` transaction and next things will happen:

1. Specified amount amount of locked ISLM token will be transfered from a user vesting account to `x/liquidvesting` module account
2. `x/liquidvesting` module will mint a liquid token which won't be locked and could be freely used in any way. Its amount will be equal to specified amount of locked ISLM token transfered to module account.
3. ERC20 contract of newly created liquid token will be deployed on evm layer and token pair for it will be created with `x/erc20` module

### Liquid token 

Liquid token represents arbitrary amount of ISLM token locked in vesting. For each liquidate transaction new unique liquid token will be created.
Liquid token has vesting unlock schedule, it derives from original vesting account schedule which liquid token created from.

## Redeem
Once user has any liquid token on its account, it can be redeemed to locked ISLM token. Once user uses `Redeem` transaction next things will happen:

1. Liquid token amount specified for redeem will be burnt
2. ISLM token will be transfered to user's account from `x/liquidvesting` module
3. Liquid token unlock schedule will be applied to user's account. If user has a regular account it converts to vesting account. If user already has vesting account liquid token schedule will be merged with already existing schedule.