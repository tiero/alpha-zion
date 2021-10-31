package application

import (
	"github.com/shopspring/decimal"
	"github.com/tiero/zion/internal/core/domain"
)

// trade type
const (
	TradeBuy = iota
	TradeSell
)

type Market struct {
	BaseAsset  string
	QuoteAsset string
}

type Balance struct {
	BaseAmount  uint64
	QuoteAmount uint64
}

// Fee is the market fee percentage in basis point:
// 	- 0,01% -> 1 bp
//	- 1,00% -> 100 bp
//	- 99,99% -> 9999 bp
type Fee struct {
	BasisPoint    int64
	FixedBaseFee  int64
	FixedQuoteFee int64
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
	Amount  uint64
	Asset   string
	Balance Balance
}

type BalanceInfo struct {
	TotalBalance       uint64
	ConfirmedBalance   uint64
	UnconfirmedBalance uint64
}

type TradeRequest domain.SwapRequest

type TradeAcceptOrFail struct {
	IsRejected bool
	Accept     *domain.SwapAccept
	Fail       *domain.SwapFail
	ExpiryTime uint64
}
