package application

import "github.com/shopspring/decimal"

type Market struct {
	BaseAsset  string
	QuoteAsset string
}

type Balance struct {
	BaseAmount  int64
	QuoteAmount int64
}

// Fee is the market fee percentage in basis point:
// 	- 0,01% -> 1 bp
//	- 1,00% -> 100 bp
//	- 99,99% -> 9999 bp
type Fee struct {
	BasisPoint int64
}

type BalanceWithFee struct {
	Balance
	Fee
}

type MarketWithFee struct {
	Market
	Fee
}

type MarketWithPrice struct {
	Market
	Price
}

type Price struct {
	BasePrice  decimal.Decimal
	QuotePrice decimal.Decimal
}

type PriceWithFee struct {
	Price
	Fee
	Amount uint64
	Asset  string
}
