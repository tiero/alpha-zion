package application

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/tiero/zion/pkg/jsonrpc"
)

const DefaultWallet string = ""

type ElementsService interface {
	ListWallets() ([]string, error)
	GetBalance(wallet string) (map[string]float64, error)
}

type elementsService struct {
	// rpcClient holds the JSONRPC client to connect to the elements node
	rpcClient *jsonrpc.RPCClient
}

func NewElementsService(endpoint string) (ElementsService, error) {
	parsedEndpoint, _ := url.Parse(endpoint)
	host := parsedEndpoint.Hostname()
	port, _ := strconv.Atoi(parsedEndpoint.Port())
	user := parsedEndpoint.User.Username()
	password, _ := parsedEndpoint.User.Password()

	client, err := jsonrpc.NewClient(host, port, user, password, false, 30)
	if err != nil {
		return nil, err
	}

	// health check
	r, err := client.Call("getblockcount", nil)
	if err = jsonrpc.HandleError(err, &r); err != nil {
		return nil, err
	}

	return &elementsService{
		rpcClient: client,
	}, nil
}

func (e *elementsService) ListWallets() ([]string, error) {

	resp, err := e.rpcClient.Call("listwallets", nil)
	if err = jsonrpc.HandleError(err, &resp); err != nil {
		return nil, err
	}

	var wallets []string
	if err := json.Unmarshal(resp.Result, &wallets); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	return wallets, nil
}

func (e *elementsService) GetBalance(wallet string) (map[string]float64, error) {

	resp, err := e.rpcClient.CallWithPath(useWallet(wallet), "getbalance", nil)
	if err = jsonrpc.HandleError(err, &resp); err != nil {
		return nil, err
	}

	var balances map[string]float64
	if err := json.Unmarshal(resp.Result, &balances); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	return balances, nil
}

func useWallet(name string) string {
	return "/wallet/" + name
}
