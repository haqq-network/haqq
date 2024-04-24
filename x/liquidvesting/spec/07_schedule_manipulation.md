# Schedule manipulation

This section describes in details how `x/liquidvesting` module handles operation with schedule mutation. Examples are provided.

## Liquidation

For example we have an account this account has 3 days of vesting so each day represented as a period and has amount which be unlocked once period is passed. 
Let's imagine every period has different amount 10,20 and 30 respectively
```
10,20,30
```

So total amount locked in this schedule is 60. We want to liquidate 20 tokens from this schedule.
We will subtract portion of this amount from every period proportionally to total sum.
For the first period :
- 10 - first period amount
- 20 - liquidation amount
- 60 - total amount

Formula is 10 - 10*20/60 -> 10 - 200/60 -> 10 - 3 = 7

Important note in above calculations. We have step 200/60 and this division has a remainder. We will track this remainder but won't use it to calculate new period.

If we perform the same operation for every period we will get:
```
7,14,20
```
The sum of new periods is 41 but expected sum is 40 because we were subtracting 20 from periods with sum of 60.
So calculate diff between sum of new periods and expected sum and it is 1. Now having the diff we subtract it from last period. So we get:
```
7,14,19
```
These are our new periods. These new periods will be the new schedule of vesting account targeted by liquidation.

Now we need to know periods for newly created liquid token. and this is simply a diff between original periods and decreased periods
```
10,20,30 - original amount in periods
7,14,19 - decreased amount in periods
3,6,11 - liquid token amount in periods
```
