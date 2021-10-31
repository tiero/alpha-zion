package application

import (
	"context"
	"fmt"

	"github.com/tdex-network/tdex-daemon/pkg/explorer"
)

func (t *tradeService) GetMarketBalance(ctx context.Context, market Market) (*BalanceWithFee, error) {

	if ok := validateAssetString(market.BaseAsset); !ok || t.baseAsset != market.BaseAsset {
		return nil, fmt.Errorf("base asset must be %s", t.baseAsset)
	}
	if ok := validateAssetString(market.QuoteAsset); !ok || t.quoteAsset != market.QuoteAsset {
		return nil, fmt.Errorf("quote asset must be %s", t.quoteAsset)
	}

	balances, err := getBalancesWithEsplora(t.wallet.Address(), t.esploraClient)
	if err != nil {
		return nil, err
	}

	baseAmount := balances[market.BaseAsset].TotalBalance
	quoteAmount := balances[market.QuoteAsset].TotalBalance

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

func getBalancesWithEsplora(address string, esploraClient explorer.Service) (balances map[string]BalanceInfo, err error) {
	unspents, err := esploraClient.GetUnspentsForAddresses(
		[]string{address},
		nil,
	)
	if err != nil {
		return nil, err
	}

	return getBalancesByAsset(unspents), nil
}

func getBalancesByAsset(unspents []explorer.Utxo) map[string]BalanceInfo {
	balances := make(map[string]BalanceInfo, 0)

	if len(unspents) == 0 {
		return balances
	}

	for _, unspent := range unspents {
		if _, ok := balances[unspent.Asset()]; !ok {
			balances[unspent.Asset()] = BalanceInfo{}
		}

		balance := balances[unspent.Asset()]
		balance.TotalBalance += unspent.Value()
		if unspent.IsConfirmed() {
			balance.ConfirmedBalance += unspent.Value()
		} else {
			balance.UnconfirmedBalance += unspent.Value()
		}
		balances[unspent.Asset()] = balance
	}
	return balances
}
