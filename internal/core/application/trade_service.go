package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/bitfinexcom/bitfinex-api-go/v2/rest"
	"github.com/shopspring/decimal"
	"github.com/tdex-network/tdex-daemon/pkg/explorer/esplora"
	"github.com/tiero/zion/internal/core/domain"
	"github.com/tiero/zion/internal/core/ports"
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
	baseFixedFee int64
	// a fixed fee charged in base asset
	quoteFixedFee int64
	// JSONRPC client for the Elements node
	client ElementsService
	// rest client to get the avergae price for a ticker via Bitfinex API
	bitfinexClient *rest.Client

	wallet WalletService

	priceEndpoint string

	baseAsset  string
	quoteAsset string

	useExplorer bool
}

func NewTradeServiceWithElements(bitfinexSvc *rest.Client, elementsSvc ElementsService, net *network.Network) (TradeService, error) {
	return &tradeService{
		bitfinexClient: bitfinexSvc,
		client:         elementsSvc,

		basisPointFee: 100,
		baseFixedFee:  650,
		quoteFixedFee: 4000,

		network:     net,
		useExplorer: false,
	}, nil
}

func NewTradeServiceWithExplorer(walletService WalletService, priceEndpoint, baseAsset, quoteAsset string) (TradeService, error) {

	return &tradeService{
		basisPointFee: 100,

		baseFixedFee:  650,
		quoteFixedFee: 4000,

		priceEndpoint: priceEndpoint,

		baseAsset:  baseAsset,
		quoteAsset: quoteAsset,

		wallet: walletService,

		useExplorer: true,
	}, nil
}

func (t *tradeService) GetTradableMarkets(ctx context.Context) ([]MarketWithFee, error) {

	markets := []Market{
		{
			BaseAsset:  t.baseAsset,
			QuoteAsset: t.quoteAsset,
		},
	}

	mkts := make([]MarketWithFee, 0, len(markets))
	for _, mkt := range markets {
		mkts = append(mkts, MarketWithFee{
			Market: Market{
				BaseAsset:  strings.TrimSpace(mkt.BaseAsset),
				QuoteAsset: strings.TrimSpace(mkt.QuoteAsset),
			},
			Fee: Fee{
				BasisPoint: t.basisPointFee,
			},
		})
	}

	return mkts, nil
}

func (t *tradeService) GetMarketBalance(ctx context.Context, market Market) (*BalanceWithFee, error) {
	if ok := validateAssetString(market.BaseAsset); !ok || t.baseAsset != market.BaseAsset {
		return nil, errors.New("invalid base asset")
	}
	if ok := validateAssetString(market.QuoteAsset); !ok || t.quoteAsset != market.QuoteAsset {
		return nil, errors.New("invalid quote asset")
	}

	baseAmount, quoteAmount, err := t.getBalances(market)
	if err != nil {
		return nil, err
	}

	return &BalanceWithFee{
		Balance: Balance{
			BaseAmount:  baseAmount,
			QuoteAmount: quoteAmount,
		},
		Fee: Fee{
			BasisPoint:    t.basisPointFee,
			FixedBaseFee:  t.baseFixedFee,
			FixedQuoteFee: t.quoteFixedFee,
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

	if ok := validateAssetString(market.BaseAsset); !ok || t.baseAsset != market.BaseAsset {
		return nil, errors.New("invalid base asset")
	}
	if ok := validateAssetString(market.QuoteAsset); !ok || t.quoteAsset != market.QuoteAsset {
		return nil, errors.New("invalid quote asset")
	}

	client := esplora.NewHTTPClient(5 * time.Second)
	status, response, err := client.NewHTTPRequest("GET", t.priceEndpoint, "", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, errors.New("price unavailable")
	}

	var priceResponse ports.PriceResponse
	err = json.Unmarshal([]byte(response), &priceResponse)
	if err != nil {
		return nil, err
	}

	basePrice, err := decimal.NewFromString(priceResponse.BasePrice)
	if err != nil {
		return nil, err
	}

	quotePrice, err := decimal.NewFromString(priceResponse.QuotePrice)
	if err != nil {
		return nil, err
	}

	decimalAmount := decimal.NewFromInt(int64(amount))

	assetToGive := market.QuoteAsset
	amountToGive := decimalAmount.Mul(quotePrice)
	if market.QuoteAsset == asset {
		assetToGive = market.BaseAsset
		amountToGive = decimalAmount.Mul(basePrice)
	}

	baseAmount, quoteAmount, err := t.getBalances(market)
	if err != nil {
		return nil, err
	}

	return &PriceWithFee{
		Price: Price{
			BasePrice:  basePrice,
			QuotePrice: quotePrice,
		},
		Fee: Fee{
			BasisPoint:    t.basisPointFee,
			FixedBaseFee:  t.baseFixedFee,
			FixedQuoteFee: t.quoteFixedFee,
		},
		Amount: amountToGive.BigInt().Uint64(),
		Asset:  assetToGive,
		Balance: Balance{
			BaseAmount:  baseAmount,
			QuoteAmount: quoteAmount,
		},
	}, nil
}

func (t *tradeService) TradePropose(
	ctx context.Context,
	market Market,
	tradeType int,
	swapRequest TradeRequest,
) (*TradeAcceptOrFail, error) {

	if ok := validateAssetString(market.BaseAsset); !ok || t.baseAsset != market.BaseAsset {
		return nil, errors.New("invalid base asset")
	}
	if ok := validateAssetString(market.QuoteAsset); !ok || t.quoteAsset != market.QuoteAsset {
		return nil, errors.New("invalid quote asset")
	}

	if tradeType < 0 || tradeType > 1 {
		return nil, errors.New("invalid quote asset")
	}

	request := domain.SwapRequest(swapRequest)

	// TODO get current price for the pair and chek if is ok
	// rejected := request.RejectWithReason()
	// return &TradeAcceptOrFail{ IsRejected: true, Fail: rejected }, nil

	//blind the transaction and sign inputs with SIGHASH_ALL
	updatedTx, err := t.wallet.CompleSwap(CompleteSwapOpts{
		PsetBase64:   request.PsetBase64,
		InputAmount:  request.AmountToReceive,
		InputAsset:   request.AssetToReceive,
		OutputAmount: request.AmountToBeSent,
		OutputAsset:  request.AssetToBeSent,
		Network:      t.network,
	})
	if err != nil {
		return nil, fmt.Errorf("complete swap: %w", err)
	}

	signedTx, err := t.wallet.SignSwap(SignSwapOpts{
		PsetBase64:                updatedTx,
		InputBlindingKeyByScript:  request.InputBlindingKeyByScript,
		OutputBlindingKeyByScript: request.OutputBlindingKeyByScript,
	})
	if err != nil {
		return nil, fmt.Errorf("sign swap: %w", err)
	}

	accepted := request.AcceptWithTransaction(signedTx, nil, nil)

	return &TradeAcceptOrFail{
		Accept:     accepted,
		ExpiryTime: uint64(time.Now().Add(time.Minute * 2).Unix()),
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

func (t *tradeService) getBalances(market Market) (uint64, uint64, error) {
	balances := make(map[string]BalanceInfo, 0)
	baseAmount := uint64(0)
	quoteAmount := uint64(0)

	balances, err := t.wallet.Balances()
	if err != nil {
		return 0, 0, err
	}

	if len(balances) > 0 {
		baseAmount = balances[market.BaseAsset].TotalBalance
		quoteAmount = balances[market.QuoteAsset].TotalBalance
	}

	return baseAmount, quoteAmount, nil
}
