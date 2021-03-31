# Zion
## 🏃‍♀💨 🚓🚓🚓 

![markus-spiske-iar-afB0QQw-unsplash](https://user-images.githubusercontent.com/3596602/113153447-65943380-9237-11eb-8a3a-c6767b030d4f.jpg)

Photo by <a href="https://unsplash.com/@markusspiske?utm_source=unsplash&utm_medium=referral&utm_content=creditCopyText">Markus Spiske</a> on <a href="https://unsplash.com/s/photos/matrix?utm_source=unsplash&utm_medium=referral&utm_content=creditCopyText">Unsplash</a>
  

## Usage

### Run

```sh
$ export ELEMENTS_RPC_ENDPOINT=http://user:pass@127.0.0.1:7041
$ ziond 
```

### Open a market

You need to create a wallet in the elements node for each market, using the asset hash of the quote asset, as the name of the wallet.


```sh
# Save the USDt asset hash in a variable
USDT="ce091c998b83c78bb71a632313ba3760f1763d9cfcffae02258ffa9865a37bd2"
# Open a L-BTC-USDt market
$ elements-cli createwallet $USDT
# Get an native segwit address to deposit funds
$ elements-cli -rpcwallet=$USDT getnewaddress "" "bech32"
```

Now you can depoist funds into it, you should have both L-BTC and USDt in it to serve trades properly. Be sure to put more L-BTC to subsidize for fees.


### Make a trade

Traders could connect to your public trader interface (by default on :9945) and can make trades now.

```sh
## assuming you have already installed, with a wallet created
$ tdex-cli connect localhost:9945
$ tdex-cli market LBTC-USDt
$ tdex-cli trade 
# Follow the instructions prompted
```
