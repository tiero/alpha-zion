package grpchandler

import (
	"errors"

	pbswap "github.com/tdex-network/tdex-protobuf/generated/go/swap"
	pb "github.com/tdex-network/tdex-protobuf/generated/go/trade"
)

func validateSwapRequest(swapRequest *pbswap.SwapRequest) error {
	if swapRequest == nil {
		return errors.New("swap request is null")
	}
	if swapRequest.GetAmountP() <= 0 ||
		len(swapRequest.GetAssetP()) <= 0 ||
		swapRequest.GetAmountR() <= 0 ||
		len(swapRequest.GetAssetR()) <= 0 ||
		len(swapRequest.GetTransaction()) <= 0 {
		return errors.New("swap request is malformed")
	}
	return nil
}

func validateTradeType(tType pb.TradeType) error {
	if int(tType) < 0 || int(tType) > 1 {
		return errors.New("trade type is unknown")
	}
	return nil
}
