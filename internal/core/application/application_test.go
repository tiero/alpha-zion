package application_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/tdex-network/tdex-daemon/pkg/explorer"
	"github.com/tdex-network/tdex-daemon/pkg/explorer/esplora"
	"github.com/tdex-network/tdex-daemon/pkg/trade"
	pbswap "github.com/tdex-network/tdex-protobuf/generated/go/swap"
	"github.com/tiero/zion/internal/core/application"
	"github.com/tiero/zion/internal/core/domain"
	"github.com/vulpemventures/go-elements/network"
)

const (
	base        = "5ac9f65c0efcc4775e0baec4ec03abdde22473cd3cf33c0419ca290e0751b225"
	quote       = "2dcf5a8834645654911964ec3602426fd3b9b4017554d3f9c19403e7fc1411d3"
	privateKey  = "bfd87b3d29e1c0846ed293d4bdc7b78d62598a92d18ae69c153558906063df9b"
	explorerUrl = "http://localhost:3001"
)

var ctx = context.Background()

func TestProposeTrade(t *testing.T) {
	t.Run("try to Trade buy", func(t *testing.T) {
		tradeSvc, err := application.NewTradeService(
			privateKey, base, quote, explorerUrl,
			network.Regtest.AssetID,
			&network.Regtest,
		)
		require.NoError(t, err)

		markets, err := tradeSvc.GetTradableMarkets(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, markets)

		market := markets[0].Market
		balances, err := tradeSvc.GetMarketBalance(ctx, market)
		require.NoError(t, err)
		require.NotNil(t, balances)
		require.True(t, balances.Balance.BaseAmount > 0)
		require.True(t, balances.Balance.QuoteAmount > 0)

		buyBaseAsset(t, tradeSvc, market)
	})
}

func buyBaseAsset(
	t *testing.T,
	tradeSvc application.TradeService,
	market application.Market,
) {
	amount := uint64(0.0001 * math.Pow10(8))
	/*
		preview, err := tradeSvc.GetMarketPrice(ctx, market, tradeType, amount, asset)
		require.NoError(t, err)
		require.NotNil(t, preview) */

	wallet, err := trade.NewRandomWallet(&network.Regtest)
	require.NoError(t, err)
	require.NotNil(t, wallet)
	_, script := wallet.Script()

	assetToSend := quote
	amountToSend := amount
	assetToReceive := base
	amountToReceive := uint64(0.5 * math.Pow10(8))

	unspents := []explorer.Utxo{
		esplora.NewUnconfidentialWitnessUtxo(
			randomHex(32),
			uint32(randomIntInRange(0, 15)),
			amountToSend,
			assetToSend,
			script,
		),
	}

	psetBase64, _ := trade.NewSwapTx(
		unspents,
		assetToSend,
		amountToSend,
		assetToReceive,
		amountToReceive,
		script,
	)

	println("psbt", psetBase64)

	swapRequest := &pbswap.SwapRequest{
		Id:                randomId(),
		AssetP:            assetToSend,
		AmountP:           amountToSend,
		AssetR:            assetToReceive,
		AmountR:           amountToReceive,
		Transaction:       psetBase64,
		InputBlindingKey:  map[string][]byte{},
		OutputBlindingKey: map[string][]byte{},
	}

	swapAccept, swapFail, expiryTime, err := tradeSvc.ProposeTrade(
		ctx, market, application.TradeBuy,
		domain.SwapRequest{
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
	require.NoError(t, err)
	require.Nil(t, swapAccept)
	require.NotNil(t, swapFail)
	require.True(t, time.Now().Before(time.Unix(int64(expiryTime), 0)))

}

func randomId() string {
	return uuid.New().String()
}

func randomHex(len int) string {
	return hex.EncodeToString(randomBytes(len))
}

func randomBytes(len int) []byte {
	b := make([]byte, len)
	rand.Read(b)
	return b
}

func randomIntInRange(min, max int) int {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(max)))
	return int(int(n.Int64())) + min
}

func randomValueCommitment() string {
	c := randomBytes(32)
	c[0] = 9
	return hex.EncodeToString(c)
}

func randomAssetCommitment() string {
	c := randomBytes(32)
	c[0] = 10
	return hex.EncodeToString(c)
}
