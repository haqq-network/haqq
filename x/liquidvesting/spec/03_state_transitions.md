<!--
order: 3
-->

# State Transitions

## Liquidate

1. User submits `MsgLiquidate`
2. Checks if liquidation allowed for account and amount
   - tokens on target account are fully vested
   - specified amount is more than minimum liquidation amount param
   - specified amount is less or equal to locked token amount
3. Calculate new schedules for account and for liquid token
4. Update target account with new schedule
5. Escrow locked token to module account
6. Create new liquid token with previously calculated schedule and update token id counter
7. Send newly created liquid token to target account
8. Deploy ERC20 contract for liquid token and register token pair with \`x/erc20\` module
9. Convert all liquid tokens from cosmos to ERC20

## Redeem

1. User submits `MsgRedeem`
2. Checks if redeem possible
   - Specified liquid token does exist
   - Check user's account has sufficient amount of liquid token to redeem
3. Burn specified liquid token amount
4. Subtract burnt liquid token amount from liquid token schedule
5. Transfer ISLM to target account
6. Apply token unlock schedule to target account. If target account is not vesting account it will be converted to vesting one.