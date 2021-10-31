package application

import (
	"encoding/hex"
	"fmt"

	"github.com/btcsuite/btcd/btcec"
	"github.com/tdex-network/tdex-daemon/pkg/bufferutil"
	"github.com/tdex-network/tdex-daemon/pkg/explorer"
	"github.com/vulpemventures/go-elements/address"
	"github.com/vulpemventures/go-elements/network"
	"github.com/vulpemventures/go-elements/payment"
	"github.com/vulpemventures/go-elements/pset"
	"github.com/vulpemventures/go-elements/transaction"
)

// CompleteSwapTxOpts is the struct given to compleSwap function
type CompleteSwapOpts struct {
	PsetBase64   string
	InputAmount  uint64
	InputAsset   string
	OutputAmount uint64
	OutputAsset  string
	Network      *network.Network
}

type KeyPair struct {
	PrivateKey *btcec.PrivateKey
	PublicKey  *btcec.PublicKey
}

type WalletService interface {
	Address() string
	Script() []byte
	CompleSwap(opts CompleteSwapOpts) (string, error)
}

type walletService struct {
	address string
	script  []byte
	keyPair *KeyPair

	// pkg/epxlorer compliant service
	esploraClient explorer.Service
	network       *network.Network

	nativeAsset string
}

func NewWalletService(privateKeyHex, nativeAsset string, net *network.Network, esploraClient explorer.Service) (WalletService, error) {

	privBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return nil, err
	}

	priv, pubKey := btcec.PrivKeyFromBytes(btcec.S256(), privBytes)

	receivePay := payment.FromPublicKey(pubKey, net, nil)

	receiveAddr, err := receivePay.WitnessPubKeyHash()
	if err != nil {
		return nil, err
	}

	println("address", receiveAddr)

	return &walletService{
		address: receiveAddr,
		script:  receivePay.Script,
		keyPair: &KeyPair{
			PublicKey:  pubKey,
			PrivateKey: priv,
		},
		esploraClient: esploraClient,
		nativeAsset:   nativeAsset,
		network:       net,
	}, nil
}

func (w *walletService) Address() string {
	return w.address
}

func (w *walletService) Script() []byte {
	return w.script
}

func (w *walletService) CompleSwap(opts CompleteSwapOpts) (string, error) {
	ptx, err := pset.NewPsetFromBase64(opts.PsetBase64)
	if err != nil {
		return "", fmt.Errorf("decode base: %w", err)
	}

	unspents, err := w.esploraClient.GetUnspentsForAddresses(
		[]string{w.address},
		nil,
	)
	if err != nil {
		return "", err
	}

	selectedUnspents, change, err := explorer.SelectUnspents(
		unspents,
		opts.InputAmount,
		opts.InputAsset,
	)
	if err != nil {
		return "", err
	}

	feeUnspents, feeChange, err := explorer.SelectUnspents(
		unspents,
		650,
		w.nativeAsset,
	)
	if err != nil {
		return "", err
	}

	script, _ := address.ToOutputScript(w.address)
	output, err := newTxOutput(opts.OutputAsset, opts.OutputAmount, script)
	if err != nil {
		return "", err
	}

	inputsToAdd := []explorer.Utxo{}
	inputsToAdd = append(inputsToAdd, selectedUnspents...)
	inputsToAdd = append(inputsToAdd, feeUnspents...)
	outputsToAdd := []*transaction.TxOutput{output}

	if change > 0 {
		script, _ := address.ToOutputScript(w.address)

		changeOutput, err := newTxOutput(opts.InputAsset, change, script)
		if err != nil {
			return "", err
		}
		outputsToAdd = append(outputsToAdd, changeOutput)
	}

	if feeChange > 0 {
		feeChangeOutput, err := newTxOutput(w.nativeAsset, feeChange, script)
		if err != nil {
			return "", err
		}

		outputsToAdd = append(outputsToAdd, feeChangeOutput)
	}

	psetBase64, err := addInsAndOutsToPset(ptx, selectedUnspents, outputsToAdd)
	if err != nil {
		return "", err
	}

	return psetBase64, nil
}

func newTxOutput(asset string, value uint64, script []byte) (*transaction.TxOutput, error) {
	changeAsset, err := bufferutil.AssetHashToBytes(asset)
	if err != nil {
		return nil, err
	}
	changeValue, err := bufferutil.ValueToBytes(value)
	if err != nil {
		return nil, err
	}
	return transaction.NewTxOutput(changeAsset, changeValue, script), nil
}

func addInsAndOutsToPset(
	ptx *pset.Pset,
	inputsToAdd []explorer.Utxo,
	outputsToAdd []*transaction.TxOutput,
) (string, error) {
	updater, err := pset.NewUpdater(ptx)
	if err != nil {
		return "", err
	}

	for _, in := range inputsToAdd {
		input, witnessUtxo, _ := in.Parse()
		updater.AddInput(input)
		err := updater.AddInWitnessUtxo(witnessUtxo, len(ptx.Inputs)-1)
		if err != nil {
			return "", err
		}
	}

	for _, out := range outputsToAdd {
		updater.AddOutput(out)
	}

	return ptx.ToBase64()
}
