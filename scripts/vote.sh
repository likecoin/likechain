#!/bin/bash

GAS="200000"
GAS_PRICE="1000"
FEE=$(expr $GAS "*" $GAS_PRICE)

set -e

LIKE_HOME="$(dirname "$0")/.."
pushd "$LIKE_HOME" > /dev/null
LIKE_HOME=$(pwd)
popd > /dev/null

CHAIN_ID=$(grep chain_id "$LIKE_HOME/.liked/config/genesis.json" | sed 's/ *"chain_id": *"\(.*\)"/\1/g' | sed 's/,$//g')
echo "Chain ID: $CHAIN_ID"


FEE="${FEE}nanolike"


read -p "Enter the ID of the proposal: " ID


read -p "Enter your vote for this proposal (Yes/No/NoWithVeto/Abstain): " DECISION


echo ""
echo "Now the script will generate and send the governance transaction, please confirm and enter your passphrase."

echo likecli --home /likechain/.likecli tx gov vote $ID $DECISION \
        --from validator \
        --chain-id "$CHAIN_ID" \
        --node tcp://liked:26657 \
        --fees "$FEE" \
        --gas "$GAS"

docker exec -it likechain_liked \
    likecli --home /likechain/.likecli tx gov vote $ID $DECISION \
        --from validator \
        --chain-id "$CHAIN_ID" \
        --node tcp://liked:26657 \
        --fees "$FEE" \
        --gas "$GAS"

echo "Governance transaction sent."
echo "You can use the transaction hash to query the transaction result on the block explorer."
