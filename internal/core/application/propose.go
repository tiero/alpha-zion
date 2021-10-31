package application

import (
	"context"
	"fmt"
	"time"

	"github.com/tiero/zion/internal/core/domain"
	"github.com/vulpemventures/go-elements/network"
)

func (t *tradeService) ProposeTrade(
	ctx context.Context,
	market Market,
	tradeType int,
	swapRequest domain.SwapRequest,
) (*domain.SwapAccept, *domain.SwapFail, uint64, error) {

	//validate market
	if ok := validateAssetString(market.BaseAsset); !ok || t.baseAsset != market.BaseAsset {
		return nil, nil, 0, fmt.Errorf("base asset must be %s", t.baseAsset)
	}
	if ok := validateAssetString(market.QuoteAsset); !ok || t.quoteAsset != market.QuoteAsset {
		return nil, nil, 0, fmt.Errorf("quote asset must be %s", t.quoteAsset)
	}

	request := domain.SwapRequest(swapRequest)
	// validate

	// rejected := request.RejectWithReason()
	// return &TradeAcceptOrFail{ IsRejected: true, Fail: rejected }, nil

	updatedTx, err := t.wallet.CompleSwap(CompleteSwapOpts{
		PsetBase64:   request.PsetBase64,
		InputAmount:  request.AmountToReceive,
		InputAsset:   request.AssetToReceive,
		OutputAmount: request.AmountToBeSent,
		OutputAsset:  request.AssetToBeSent,
		Network:      &network.Regtest,
	})
	if err != nil {
		return nil, nil, 0, fmt.Errorf("complete swap: %w", err)
	}

	signedTx, err := t.wallet.SignSwap(SignSwapOpts{
		PsetBase64: updatedTx,
	})
	if err != nil {
		return nil, nil, 0, fmt.Errorf("sign swap: %w", err)
	}

	accepted := request.AcceptWithTransaction(signedTx, nil, nil)

	return accepted, nil, uint64(time.Now().Add(time.Minute * 2).Unix()), nil
}
