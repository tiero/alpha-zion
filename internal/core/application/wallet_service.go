package application

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/tdex-network/tdex-daemon/pkg/bufferutil"
	"github.com/tdex-network/tdex-daemon/pkg/explorer"
	"github.com/tdex-network/tdex-daemon/pkg/explorer/esplora"
	"github.com/tyler-smith/go-bip39"
	"github.com/vulpemventures/go-elements/address"
	"github.com/vulpemventures/go-elements/network"
	"github.com/vulpemventures/go-elements/payment"
	"github.com/vulpemventures/go-elements/pset"
	"github.com/vulpemventures/go-elements/slip77"
	"github.com/vulpemventures/go-elements/transaction"
)

type WalletService interface {
	Address() (addr *AddressAndBlinding)
	ChangeAddress() (addr *AddressAndBlinding)
	Balances() (map[string]BalanceInfo, error)
	CompleSwap(opts CompleteSwapOpts) (string, error)
	SignSwap(opts SignSwapOpts) (string, error)
	SendToMany(
		payouts map[string]AmountAndAsset,
	) ([]byte, error)
}

// CompleteSwapTxOpts is the struct given to UpdateTx method
type CompleteSwapOpts struct {
	PsetBase64   string
	InputAmount  uint64
	InputAsset   string
	OutputAmount uint64
	OutputAsset  string
	Network      *network.Network
}

type SignSwapOpts struct {
	PsetBase64                string
	InputBlindingKeyByScript  map[string][]byte
	OutputBlindingKeyByScript map[string][]byte
}

type AmountAndAsset struct {
	Amount uint64
	Asset  string
}

type AddressAndBlinding struct {
	ConfidentialAddress string
	BlindingPrivateKey  []byte
}

type Keys struct {
	signingPrivateKey  *btcec.PrivateKey
	blindingPrivateKey *btcec.PrivateKey
}

type walletService struct {
	receivingAddress string
	changeAddress    string

	receivingKeys *Keys
	changeKeys    *Keys

	// pkg/epxlorer compliant service
	esploraClient explorer.Service
	network       *network.Network

	nativeAsset string
}

func NewWalletService(
	mnemonic,
	nativeAsset,
	explorerEndpoint string,
	net *network.Network,
) (WalletService, error) {

	seed := bip39.NewSeed(mnemonic, "")

	hdNode, err := hdkeychain.NewMaster(seed, &chaincfg.MainNetParams)
	if err != nil {
		return nil, err
	}

	slip77Node, err := slip77.FromSeed(seed)
	if err != nil {
		return nil, err
	}

	purpose, err := hdNode.Derive(84 + hdkeychain.HardenedKeyStart)
	if err != nil {
		return nil, err
	}

	// m/84'/0'
	coinType, err := purpose.Derive(0 + hdkeychain.HardenedKeyStart)
	if err != nil {
		return nil, err
	}

	// m/84'/0'/0'
	acct0, err := coinType.Derive(0 + hdkeychain.HardenedKeyStart)
	if err != nil {
		return nil, err
	}

	// m/84'/0'/0'/0
	acct0External, err := acct0.Derive(0)
	if err != nil {
		return nil, err
	}

	// m/84'/0'/0'/1
	acct0Internal, err := acct0.Derive(1)
	if err != nil {
		return nil, err
	}

	receivingChild, err := acct0External.Derive(0)
	if err != nil {
		return nil, err
	}

	receivingKey, err := receivingChild.ECPrivKey()
	if err != nil {
		return nil, err
	}

	changeChild, err := acct0Internal.Derive(0)
	if err != nil {
		return nil, err
	}
	changeKey, err := changeChild.ECPrivKey()
	if err != nil {
		return nil, err
	}

	_, receivingPubKey := btcec.PrivKeyFromBytes(btcec.S256(), receivingKey.Serialize())
	_, changePubKey := btcec.PrivKeyFromBytes(btcec.S256(), changeKey.Serialize())

	receivePay := payment.FromPublicKey(receivingPubKey, net, nil)
	changePay := payment.FromPublicKey(changePubKey, net, nil)

	recBlindPrivKey, recBlindPubKey, err := slip77Node.DeriveKey(receivePay.WitnessScript)
	if err != nil {
		return nil, err
	}

	chaBlindPrivKey, chaBlindPubKey, err := slip77Node.DeriveKey(changePay.WitnessScript)
	if err != nil {
		return nil, err
	}

	receivePay.BlindingKey = recBlindPubKey
	changePay.BlindingKey = chaBlindPubKey

	receiveAddr, err := receivePay.ConfidentialWitnessPubKeyHash()
	if err != nil {
		return nil, err
	}

	changeAddr, err := changePay.ConfidentialWitnessPubKeyHash()
	if err != nil {
		return nil, err
	}

	esploraClient, err := esplora.NewService(explorerEndpoint, 5000)
	if err != nil {
		return nil, err
	}

	return newWalletService(receiveAddr, changeAddr, nativeAsset, receivingKey, changeKey, recBlindPrivKey, chaBlindPrivKey, esploraClient), nil
}

func newWalletService(receiveAddr, changeAddr, nativeAsset string, receivingKey, changeKey, recBlindPrivKey, chaBlindPrivKey *btcec.PrivateKey, esploraClient explorer.Service) *walletService {
	return &walletService{
		receivingAddress: receiveAddr,
		changeAddress:    changeAddr,
		receivingKeys: &Keys{
			signingPrivateKey:  receivingKey,
			blindingPrivateKey: recBlindPrivKey,
		},
		changeKeys: &Keys{
			signingPrivateKey:  changeKey,
			blindingPrivateKey: chaBlindPrivKey,
		},
		esploraClient: esploraClient,
		nativeAsset:   nativeAsset,
	}
}

func (w *walletService) Address() (addr *AddressAndBlinding) {
	return &AddressAndBlinding{
		ConfidentialAddress: w.receivingAddress,
		BlindingPrivateKey:  w.receivingKeys.blindingPrivateKey.Serialize(),
	}
}

func (w *walletService) ChangeAddress() (addr *AddressAndBlinding) {
	return &AddressAndBlinding{
		ConfidentialAddress: w.changeAddress,
		BlindingPrivateKey:  w.changeKeys.blindingPrivateKey.Serialize(),
	}
}

func (w *walletService) Balances() (balances map[string]BalanceInfo, err error) {
	unspents, err := w.esploraClient.GetUnspentsForAddresses(
		[]string{w.receivingAddress, w.changeAddress},
		[][]byte{w.receivingKeys.blindingPrivateKey.Serialize(), w.changeKeys.blindingPrivateKey.Serialize()},
	)
	if err != nil {
		return nil, err
	}

	return getBalancesByAsset(unspents), nil
}

func (w *walletService) CompleSwap(opts CompleteSwapOpts) (string, error) {
	ptx, err := pset.NewPsetFromBase64(opts.PsetBase64)
	if err != nil {
		return "", err
	}

	unspents, err := w.esploraClient.GetUnspentsForAddresses(
		[]string{w.receivingAddress, w.changeAddress},
		[][]byte{w.receivingKeys.blindingPrivateKey.Serialize(), w.changeKeys.blindingPrivateKey.Serialize()},
	)
	if err != nil {
		return "", err
	}

	for _, v := range unspents {
		println(v.Asset())
		println(v.Value())
		println(v.Script())

		println(v.RangeProof())
		println(v.Nonce())
		println(v.SurjectionProof())
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

	script, _ := address.ToOutputScript(w.receivingAddress)
	output, err := newTxOutput(opts.OutputAsset, opts.OutputAmount, script)
	if err != nil {
		return "", err
	}

	inputsToAdd := []explorer.Utxo{}
	inputsToAdd = append(inputsToAdd, selectedUnspents...)
	inputsToAdd = append(inputsToAdd, feeUnspents...)
	outputsToAdd := []*transaction.TxOutput{output}

	if change > 0 {
		script, _ := address.ToOutputScript(w.changeAddress)

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

func (w *walletService) SignSwap(opts SignSwapOpts) (string, error) {
	ptx, err := pset.NewPsetFromBase64(opts.PsetBase64)
	if err != nil {
		return "", fmt.Errorf("decode pset: %w", err)
	}

	receiveScript, _ := address.ToOutputScript(w.receivingAddress)
	changeScript, _ := address.ToOutputScript(w.receivingAddress)

	// add our own blinding keys
	inBlindingKeys := opts.InputBlindingKeyByScript
	inBlindingKeys[hex.EncodeToString(receiveScript)] = w.receivingKeys.blindingPrivateKey.Serialize()
	inBlindingKeys[hex.EncodeToString(changeScript)] = w.changeKeys.blindingPrivateKey.Serialize()

	outBlindingKeys := opts.OutputBlindingKeyByScript
	outBlindingKeys[hex.EncodeToString(receiveScript)] = w.receivingKeys.blindingPrivateKey.Serialize()
	outBlindingKeys[hex.EncodeToString(changeScript)] = w.changeKeys.blindingPrivateKey.Serialize()

	toBeSignedTx, err := blindTransaction(ptx, inBlindingKeys, outBlindingKeys)
	if err != nil {
		return "", fmt.Errorf("blind: %w", err)
	}

	for i, in := range toBeSignedTx.Inputs {
		var privKey *btcec.PrivateKey

		isReceive := bytes.Compare(receiveScript, in.WitnessUtxo.Script) == 0
		isChange := bytes.Compare(changeScript, in.WitnessUtxo.Script) == 0

		if isReceive {
			privKey = w.receivingKeys.signingPrivateKey
		}

		if isChange {
			privKey = w.changeKeys.signingPrivateKey
		}

		if !isReceive && !isChange {
			continue
		}

		err := signInput(toBeSignedTx, i, privKey)
		if err != nil {
			return "", fmt.Errorf("sign input: %w", err)
		}
	}

	return toBeSignedTx.ToBase64()
}

func (w *walletService) SendToMany(payouts map[string]AmountAndAsset) (txID []byte, err error) {
	return
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

func blindTransaction(
	ptx *pset.Pset,
	InputBlindingKeys map[string][]byte,
	OutputBlindingKeys map[string][]byte,
) (*pset.Pset, error) {

	for _, in := range ptx.Inputs {
		if in.WitnessUtxo == nil {
			return nil, errors.New("input witness utxo must not be null")
		}
	}

	println(len(InputBlindingKeys), len(ptx.Outputs), len(OutputBlindingKeys))

	inKeysLen := len(ptx.Inputs)
	inBlindingKeys := make([]pset.BlindingDataLike, 0, inKeysLen)
	for _, in := range ptx.Inputs {
		script := hex.EncodeToString(in.WitnessUtxo.Script)
		inBlindingKeys = append(
			inBlindingKeys,
			pset.PrivateBlindingKey(InputBlindingKeys[script]),
		)
	}

	outBlindingKeys := make(map[int][]byte)
	for i, out := range ptx.UnsignedTx.Outputs {
		script := hex.EncodeToString(out.Script)
		_, pubkey := btcec.PrivKeyFromBytes(btcec.S256(), OutputBlindingKeys[script])
		outBlindingKeys[i] = pubkey.SerializeCompressed()
	}

	blinder, err := pset.NewBlinder(
		ptx,
		inBlindingKeys,
		outBlindingKeys,
		nil,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("new blinder: %w", err)
	}

	if err := blinder.Blind(); err != nil {
		return nil, fmt.Errorf("blind: %w", err)
	}

	return ptx, nil
}

func signInput(ptx *pset.Pset, inIndex int, prvkey *btcec.PrivateKey) error {
	updater, err := pset.NewUpdater(ptx)
	if err != nil {
		return err
	}

	pay, err := payment.FromScript(ptx.Inputs[inIndex].WitnessUtxo.Script, nil, nil)
	if err != nil {
		return err
	}

	script := pay.Script
	hashForSignature := ptx.UnsignedTx.HashForWitnessV0(
		inIndex,
		script,
		ptx.Inputs[inIndex].WitnessUtxo.Value,
		txscript.SigHashAll,
	)

	signature, err := prvkey.Sign(hashForSignature[:])
	if err != nil {
		return err
	}

	if !signature.Verify(hashForSignature[:], prvkey.PubKey()) {
		return fmt.Errorf(
			"signature verification failed for input %d",
			inIndex,
		)
	}

	sigWithSigHashType := append(signature.Serialize(), byte(txscript.SigHashAll))
	_, err = updater.Sign(
		inIndex,
		sigWithSigHashType,
		prvkey.PubKey().SerializeCompressed(),
		nil,
		nil,
	)
	if err != nil {
		return err
	}
	return nil
}
