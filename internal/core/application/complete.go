package application

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/tdex-network/tdex-daemon/pkg/circuitbreaker"
	"github.com/tiero/zion/internal/core/domain"
	"github.com/vulpemventures/go-elements/pset"
)

func (t *tradeService) CompleteTrade(
	ctx context.Context,
	swapComplete domain.SwapComplete,
) (txID string, swapFail *domain.SwapFail, err error) {
	tx := swapComplete.RawTxOrPsetBase64

	var txHex string
	if isHex(tx) {
		txHex = tx
	} else {
		txHex, _, err = finalizeAndExtractTransaction(tx)
		if err != nil {
			return "", nil, fmt.Errorf("invalid pset provided on complete: ", err)
		}
	}

	cb := circuitbreaker.NewCircuitBreaker()
	iTxid, err := cb.Execute(func() (interface{}, error) {
		return t.esploraClient.BroadcastTransaction(txHex)
	})
	if err != nil {
		log.WithError(err).WithField("hex", txHex).Warn("unable to broadcast trade tx")

		swapFailMsg := swapComplete.RejectWithReason("esplora broadcast error")
		return "", swapFailMsg, err
	}
	txID = iTxid.(string)

	log.Infof("trade with hash %s broadcasted", txID)

	return
}

func isHex(s string) bool {
	_, err := hex.DecodeString(s)
	return err == nil
}

func finalizeAndExtractTransaction(psetBase64 string) (string, string, error) {
	ptx, _ := pset.NewPsetFromBase64(psetBase64)

	ok, err := ptx.ValidateAllSignatures()
	if err != nil {
		return "", "", err
	}
	if !ok {
		return "", "", errors.New("invalid signatures")
	}

	if err := pset.FinalizeAll(ptx); err != nil {
		return "", "", err
	}

	tx, err := pset.Extract(ptx)
	if err != nil {
		return "", "", err
	}
	txHex, err := tx.ToHex()
	if err != nil {
		return "", "", err
	}

	return txHex, tx.TxHash().String(), nil
}
