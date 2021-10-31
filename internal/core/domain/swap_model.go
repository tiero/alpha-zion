package domain

import (
	"github.com/tdex-network/tdex-protobuf/generated/go/swap"
	"github.com/thanhpk/randstr"
)

type SwapRequest struct {
	// Random unique identifier for the current message
	ID string
	// The proposer's asset hash
	AssetToBeSent string
	// The proposer's quantity
	AmountToBeSent uint64
	// The responder's asset hash
	AssetToReceive string
	// The responder's quantity
	AmountToReceive uint64
	// The proposer's unsigned transaction in PSET format (base64 string)
	PsetBase64 string
	// In case of a confidential transaction the blinding key of each confidential
	// input is included. Each blinding key is identified by the prevout script
	// hex encoded.
	InputBlindingKeyByScript map[string][]byte
	// In case of a confidential transaction the blinding key of each confidential
	// output is included. Each blinding key is identified by the output script
	// hex encoded.
	OutputBlindingKeyByScript map[string][]byte
}

type SwapAccept struct {
	// Random unique identifier for the current message
	ID string
	// indetifier of the SwapRequest message
	RequestID string
	// The partial signed transaction base64 encoded containing the Responder's
	// signed inputs in a PSBT format
	PsetBase64 string
	// In case of a confidential transaction the blinding key of each confidential
	// input is included. Each blinding key is identified by the prevout script
	// hex encoded.
	InputBlindingKeyByScript map[string][]byte
	// In case of a confidential transaction the blinding key of each confidential
	// output is included. Each blinding key is identified by the output script
	// hex encoded.
	OutputBlindingKeyByScript map[string][]byte
}

type SwapFail struct {
	// Random unique identifier for the current message
	ID string
	// indetifier of either SwapRequest or SwapAccept message. It can be empty
	MessageID string
	// The failure code. It can be empty
	Code uint32
	// The failure reason messaged
	FailureMessage string
}

func (r *SwapRequest) AcceptWithTransaction(transaction string, inputsBlindKeys, outsBlindKeys map[string][]byte) *SwapAccept {
	randomID := randstr.Hex(8)

	// TODO validate the given transaction agains SwapRequest terms (?)

	return &SwapAccept{
		ID:         randomID,
		RequestID:  r.ID,
		PsetBase64: transaction,
		// this eventually could be extracted via rangeproof sidechannel
		InputBlindingKeyByScript: inputsBlindKeys,
		// this should be extracted via PSET2 spec
		OutputBlindingKeyByScript: outsBlindKeys,
	}
}

func (r *SwapRequest) RejectWithReason(message string) *SwapFail {
	randomID := randstr.Hex(8)

	return &SwapFail{
		ID:             randomID,
		MessageID:      r.ID,
		FailureMessage: message,
	}
}

func (a *SwapAccept) ToProtobuf() *swap.SwapAccept {
	return &swap.SwapAccept{
		Id:                a.ID,
		RequestId:         a.RequestID,
		Transaction:       a.PsetBase64,
		InputBlindingKey:  a.InputBlindingKeyByScript,
		OutputBlindingKey: a.OutputBlindingKeyByScript,
	}
}

func (f *SwapFail) ToProtobuf() *swap.SwapFail {
	return &swap.SwapFail{
		Id:             f.ID,
		MessageId:      f.MessageID,
		FailureMessage: f.FailureMessage,
	}
}
