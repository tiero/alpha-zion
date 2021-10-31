package application

import (
	"context"
	"regexp"

	"github.com/tdex-network/tdex-daemon/pkg/explorer"
	"github.com/tdex-network/tdex-daemon/pkg/explorer/esplora"
	"github.com/tiero/zion/internal/core/domain"
	"github.com/vulpemventures/go-elements/network"
)

type TradeService interface {
	GetTradableMarkets(ctx context.Context) ([]MarketWithFee, error)
	GetMarketBalance(ctx context.Context, market Market) (*BalanceWithFee, error)
	ProposeTrade(
		ctx context.Context,
		market Market,
		tradeType int,
		swapRequest domain.SwapRequest,
	) (*domain.SwapAccept, *domain.SwapFail, uint64, error)
}

type tradeService struct {
	esploraClient explorer.Service

	// wallet
	wallet WalletService

	// market
	baseAsset  string
	quoteAsset string

	// a fee on each trade expressed in basis point charged "on the way in"
	basisPointFee int64
	// a fixed fee charged in base asset
	baseFixedFee int64
	// a fixed fee charged in base asset
	quoteFixedFee int64
}

func NewTradeService(privkeyHex, baseAsset, quoteAsset, explorerEndpoint string) (TradeService, error) {
	return newTradeService(privkeyHex, baseAsset, quoteAsset, explorerEndpoint)
}

func newTradeService(privkeyHex, baseAsset, quoteAsset, explorerEndpoint string) (*tradeService, error) {
	esploraClient, err := esplora.NewService(explorerEndpoint, 8000)
	if err != nil {
		return nil, err
	}

	wallet, err := NewWalletService(privkeyHex, network.Regtest.AssetID, &network.Regtest, esploraClient)
	if err != nil {
		return nil, err
	}

	return &tradeService{
		esploraClient: esploraClient,
		wallet:        wallet,
		baseAsset:     baseAsset,
		quoteAsset:    quoteAsset,
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
