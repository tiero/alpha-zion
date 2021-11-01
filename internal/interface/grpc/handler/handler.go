package grpchandler

import (
	"context"

	pbswap "github.com/tdex-network/tdex-protobuf/generated/go/swap"
	pb "github.com/tdex-network/tdex-protobuf/generated/go/trade"
	pbtypes "github.com/tdex-network/tdex-protobuf/generated/go/types"

	"github.com/tiero/zion/internal/core/application"
	"github.com/tiero/zion/internal/core/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type traderHandler struct {
	pb.UnimplementedTradeServer
	tradeSvc application.TradeService
}

func NewTraderHandler(tradeService application.TradeService) pb.TradeServer {

	return &traderHandler{tradeSvc: tradeService}
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
				Fixed: &pbtypes.Fixed{
					BaseFee:  v.FixedBaseFee,
					QuoteFee: v.FixedQuoteFee,
				},
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
			Fixed: &pbtypes.Fixed{
				BaseFee:  balance.FixedBaseFee,
				QuoteFee: balance.FixedQuoteFee,
			},
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

	return &pb.MarketPriceReply{
		Prices: []*pbtypes.PriceWithFee{
			{
				Price: &pbtypes.Price{
					BasePrice:  float32(0),
					QuotePrice: float32(0),
				},
				Fee: &pbtypes.Fee{
					BasisPoint: 0,
				},
				Amount: 0,
				Asset:  "",
			},
		},
	}, nil
}

func (t traderHandler) ProposeTrade(
	ctx context.Context,
	req *pb.ProposeTradeRequest,
) (*pb.ProposeTradeReply, error) {

	market := application.Market{
		BaseAsset:  req.GetMarket().GetBaseAsset(),
		QuoteAsset: req.GetMarket().GetQuoteAsset(),
	}
	tradeType := req.GetType()
	if err := validateTradeType(tradeType); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	swapRequest := req.GetSwapRequest()
	if err := validateSwapRequest(swapRequest); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	accept, fail, swapExpiryTime, err := t.tradeSvc.ProposeTrade(
		ctx, market, int(tradeType), domain.SwapRequest{
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
		return nil, err
	}

	var swapAccept *pbswap.SwapAccept
	var swapFail *pbswap.SwapFail

	if accept != nil {
		swapAccept = accept.ToProtobuf()
	}
	if fail != nil {
		swapFail = fail.ToProtobuf()
	}

	return &pb.ProposeTradeReply{
		SwapAccept:     swapAccept,
		SwapFail:       swapFail,
		ExpiryTimeUnix: swapExpiryTime,
	}, nil
}

func (t traderHandler) CompleteTrade(
	ctx context.Context,
	req *pb.CompleteTradeRequest,
) (*pb.CompleteTradeReply, error) {

	swapFail := req.GetSwapFail()
	if swapFail != nil {
		return &pb.CompleteTradeReply{
			SwapFail: swapFail,
		}, nil
	}

	swapComplete := req.GetSwapComplete()
	txID, fail, err := t.tradeSvc.CompleteTrade(
		ctx,
		domain.SwapComplete{
			ID:                swapComplete.GetId(),
			AcceptID:          swapComplete.GetAcceptId(),
			RawTxOrPsetBase64: swapComplete.GetTransaction(),
		},
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if fail != nil {
		return &pb.CompleteTradeReply{
			SwapFail: fail.ToProtobuf(),
		}, nil
	}

	return &pb.CompleteTradeReply{
		Txid:     txID,
		SwapFail: nil,
	}, nil
}
