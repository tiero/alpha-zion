package application

import (
	"context"
	"strings"
)

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
