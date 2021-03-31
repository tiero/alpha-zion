package grpchandler

import (
	"context"

	log "github.com/sirupsen/logrus"
	pb "github.com/tdex-network/tdex-protobuf/generated/go/trade"
	pbtypes "github.com/tdex-network/tdex-protobuf/generated/go/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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
		req.GetAmount(),
		req.GetAsset(),
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
	market := application.Market{
		BaseAsset:  req.GetMarket().GetBaseAsset(),
		QuoteAsset: req.GetMarket().GetQuoteAsset(),
	}
	tradeType := req.GetType()
	if err := validateTradeType(tradeType); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	swapRequest := req.GetSwapRequest()
	if err := validateSwapRequest(swapRequest); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	reply, err := t.tradeSvc.TradePropose(
		stream.Context(),
		market,
		int(tradeType),
		application.TradeRequest{
			ID:                        swapRequest.GetId(),
			AmountToBeSent:            swapRequest.GetAmountP(),
			AmountToReceive:           swapRequest.GetAmountR(),
			AssetToBeSent:             swapRequest.GetAssetP(),
			AssetToReceive:            swapRequest.GetAssetR(),
			PsetBase64:                swapRequest.GetTransaction(),
			InputBlindingKeyByScript:  swapRequest.GetInputBlindingKey(),
			OutputBlindingKeyByScript: swapRequest.GetOutputBlindingKey(),
		},
	)
	if err != nil {
		log.Debug("trying to process trade proposal: ", err)
		return status.Error(codes.Internal, "cannot serve request please retry")
	}

	// if the proposal could be rejected for bad pricing
	if reply.IsRejected {
		if err := stream.Send(&pb.TradeProposeReply{
			SwapFail: reply.Fail.ToProtobuf(),
		}); err != nil {
			return status.Error(codes.Internal, err.Error())
		}
	}

	// if accepted close the stream right away
	if err := stream.Send(&pb.TradeProposeReply{
		SwapAccept:     reply.Accept.ToProtobuf(),
		ExpiryTimeUnix: reply.ExpiryTime,
	}); err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	return nil
}

func (t traderHandler) TradeComplete(
	req *pb.TradeCompleteRequest,
	stream pb.Trade_TradeCompleteServer,
) error {
	return nil
}
