package application

import (
	"context"
	"errors"
	"math"
	"regexp"
	"strings"

	"github.com/bitfinexcom/bitfinex-api-go/v2/rest"
	"github.com/shopspring/decimal"
	"github.com/tiero/zion/internal/core/domain"
	"github.com/vulpemventures/go-elements/network"
)

type TradeService interface {
	GetTradableMarkets(ctx context.Context) ([]MarketWithFee, error)
	GetMarketBalance(ctx context.Context, market Market) (*BalanceWithFee, error)
	GetMarketPrice(
		ctx context.Context,
		market Market,
		tradeType int,
		amount uint64,
		asset string,
	) (*PriceWithFee, error)
	TradePropose(
		ctx context.Context,
		market Market,
		tradeType int,
		swapRequest TradeRequest,
	) (*TradeAcceptOrFail, error)
}

type tradeService struct {
	// network holds the current network we are working on
	network *network.Network
	// a fee on each trade expressed in basis point charged "on the way in"
	basisPointFee int64
	// a fixed fee charged in base asset
	//baseFixedFee int64
	// a fixed fee charged in base asset
	//quoteFixedFee int64
	// JSONRPC client for the Elements node
	client ElementsService

	// rest client to get the avergae price for a ticker via Bitfinex API
	bitfinexClient *rest.Client
}

func NewTradeService(bitfinexSvc *rest.Client, elementsSvc ElementsService, net *network.Network) (TradeService, error) {

	return &tradeService{
		bitfinexClient: bitfinexSvc,
		client:         elementsSvc,
		basisPointFee:  100,
		network:        net,
	}, nil
}

func (t *tradeService) GetTradableMarkets(ctx context.Context) ([]MarketWithFee, error) {

	wallets, err := t.client.ListWallets()
	if err != nil {
		return nil, err
	}

	mkts := make([]MarketWithFee, 0, len(wallets))
	for _, walletName := range wallets {
		if ok := validateAssetString(walletName); ok {
			mkts = append(mkts, MarketWithFee{
				Market: Market{
					BaseAsset:  strings.TrimSpace(t.network.AssetID),
					QuoteAsset: strings.TrimSpace(walletName),
				},
				Fee: Fee{
					BasisPoint: t.basisPointFee,
				},
			})
		}
	}

	return mkts, nil
}

func (t *tradeService) GetMarketBalance(ctx context.Context, market Market) (*BalanceWithFee, error) {
	if ok := validateAssetString(market.BaseAsset); !ok {
		return nil, errors.New("invalid base asset")
	}
	if ok := validateAssetString(market.QuoteAsset); !ok {
		return nil, errors.New("invalid quote asset")
	}

	balances, err := t.client.GetBalance(strings.TrimSpace(market.QuoteAsset))
	if err != nil {
		return nil, err
	}

	if _, ok := balances["bitcoin"]; !ok {
		return nil, errors.New("no balance for base asset. Has the market been funded?")
	}

	if _, ok := balances[market.QuoteAsset]; !ok {
		return nil, errors.New("no balance for quote asset. Has the market been funded?")
	}

	baseAmount := balances["bitcoin"] * float64(math.Pow10(8))
	quoteAmount := balances[market.QuoteAsset] * float64(math.Pow10(8))

	return &BalanceWithFee{
		Balance: Balance{
			BaseAmount:  int64(baseAmount),
			QuoteAmount: int64(quoteAmount),
		},
		Fee: Fee{
			BasisPoint: t.basisPointFee,
		},
	}, nil

}

func (t *tradeService) GetMarketPrice(
	ctx context.Context,
	market Market,
	tradeType int,
	amount uint64,
	asset string,
) (*PriceWithFee, error) {

	if ok := validateAssetString(market.BaseAsset); !ok {
		return nil, errors.New("invalid base asset")
	}
	if ok := validateAssetString(market.QuoteAsset); !ok {
		return nil, errors.New("invalid quote asset")
	}

	tickerPrice, err := t.bitfinexClient.Tickers.Get("tBTCUSD")
	if err != nil {
		return nil, err
	}

	basePrice := decimal.NewFromFloat(1 / tickerPrice.LastPrice)
	quotePrice := decimal.NewFromFloat(tickerPrice.LastPrice)
	decimalAmount := decimal.NewFromInt(int64(amount))

	assetToGive := market.QuoteAsset
	amountToGive := decimalAmount.Mul(quotePrice)
	if market.QuoteAsset == asset {
		assetToGive = market.BaseAsset
		amountToGive = decimalAmount.Mul(basePrice)
	}

	return &PriceWithFee{
		Price: Price{
			BasePrice:  basePrice,
			QuotePrice: quotePrice,
		},
		Fee: Fee{
			BasisPoint: t.basisPointFee,
		},
		Amount: amountToGive.BigInt().Uint64(),
		Asset:  assetToGive,
	}, nil
}

func (t *tradeService) TradePropose(
	ctx context.Context,
	market Market,
	tradeType int,
	swapRequest TradeRequest,
) (*TradeAcceptOrFail, error) {

	if ok := validateAssetString(market.BaseAsset); !ok {
		return nil, errors.New("invalid base asset")
	}

	if ok := validateAssetString(market.QuoteAsset); !ok {
		return nil, errors.New("invalid quote asset")
	}

	if tradeType < 0 || tradeType > 1 {
		return nil, errors.New("invalid quote asset")
	}

	// TODO get current price for the pair and chek if is ok

	// TODO blind the transaction and sign ziond's inputs with SIGHASH_ALL

	request := domain.SwapRequest(swapRequest)
	accepted := request.AcceptWithTransaction("fooo bar", nil, nil)

	return &TradeAcceptOrFail{
		Accept: accepted,
	}, nil
}

func validateAssetString(asset string) bool {
	const regularExpression = `[0-9A-Fa-f]{64}`

	matched, err := regexp.Match(regularExpression, []byte(asset))
	if err != nil {
		return false
	}

	if !matched {
		return false
	}

	return true
}
