package grpchandler

import (
	"context"

	pb "github.com/tdex-network/tdex-protobuf/generated/go/trade"
	pbtypes "github.com/tdex-network/tdex-protobuf/generated/go/types"

	"github.com/tiero/zion/internal/core/application"
)

type traderHandler struct {
	pb.UnimplementedTradeServer
	tradeSvc application.TradeService
}

func NewTraderHandler(
	tradeService application.TradeService,
) pb.TradeServer {

	return &traderHandler{
		tradeSvc: tradeService,
	}
}

func (t traderHandler) Markets(
	ctx context.Context,
	req *pb.MarketsRequest,
) (*pb.MarketsReply, error) {

	mkts, err := t.tradeSvc.GetTradableMarkets(ctx)
	if err != nil {
		return nil, err
	}

	marketsWithFee := make([]*pbtypes.MarketWithFee, 0, len(mkts))
	for _, v := range mkts {
		m := &pbtypes.MarketWithFee{
			Market: &pbtypes.Market{
				BaseAsset:  v.BaseAsset,
				QuoteAsset: v.QuoteAsset,
			},
			Fee: &pbtypes.Fee{
				BasisPoint: v.BasisPoint,
			},
		}
		marketsWithFee = append(marketsWithFee, m)
	}

	return &pb.MarketsReply{Markets: marketsWithFee}, nil
}

func (t traderHandler) Balances(
	ctx context.Context,
	req *pb.BalancesRequest,
) (*pb.BalancesReply, error) {

	balance, err := t.tradeSvc.GetMarketBalance(ctx, application.Market{
		BaseAsset:  req.GetMarket().GetBaseAsset(),
		QuoteAsset: req.GetMarket().GetQuoteAsset(),
	})
	if err != nil {
		return nil, err
	}

	balancesWithFee := make([]*pbtypes.BalanceWithFee, 0)
	balancesWithFee = append(balancesWithFee, &pbtypes.BalanceWithFee{
		Balance: &pbtypes.Balance{
			BaseAmount:  balance.BaseAmount,
			QuoteAmount: balance.QuoteAmount,
		},
		Fee: &pbtypes.Fee{
			BasisPoint: balance.BasisPoint,
		},
	})

	return &pb.BalancesReply{
		Balances: balancesWithFee,
	}, nil
}

func (t traderHandler) MarketPrice(
	ctx context.Context,
	req *pb.MarketPriceRequest,
) (*pb.MarketPriceReply, error) {

	price, err := t.tradeSvc.GetMarketPrice(
		ctx,
		application.Market{
			BaseAsset:  req.GetMarket().GetBaseAsset(),
			QuoteAsset: req.GetMarket().GetQuoteAsset(),
		},
		int(req.GetType()),
		0,
		"",
	)
	if err != nil {
		return nil, err
	}

	basePrice, _ := price.BasePrice.Float64()
	quotePrice, _ := price.QuotePrice.Float64()

	return &pb.MarketPriceReply{
		Prices: []*pbtypes.PriceWithFee{
			{
				Price: &pbtypes.Price{
					BasePrice:  float32(basePrice),
					QuotePrice: float32(quotePrice),
				},
				Fee: &pbtypes.Fee{
					BasisPoint: price.BasisPoint,
				},
				Amount: price.Amount,
				Asset:  price.Asset,
			},
		},
	}, nil
}

func (t traderHandler) TradePropose(
	req *pb.TradeProposeRequest,
	stream pb.Trade_TradeProposeServer,
) error {
	return nil
}

func (t traderHandler) TradeComplete(
	req *pb.TradeCompleteRequest,
	stream pb.Trade_TradeCompleteServer,
) error {
	return nil
}
