# reti-algodrop

A daemon that rewards, every round, some Reti pool account with testnet Algo. 

## Eligible accounts

Account has to:
* be a Reti pool
* be online
* be not suspended
* have at least one staker

## Prize

The default prize is 100% of previous block fees
If the pool happens to be a proposer then extra 2A algo is added to the prize

## Winner selection

Prize is awarded to the eligible proposer or random eligible pool if proposing account does not meet the requirements (ie not a Reti pool)


