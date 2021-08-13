package config

import (
	"github.com/spf13/viper"
	"github.com/vulpemventures/go-elements/network"
)

const (
	// LogLevelKey are the different logging levels. For reference on the values https://godoc.org/github.com/sirupsen/logrus#Level
	LogLevelKey = "LOG_LEVEL"
	// NetworkKey defines the network
	NetworkKey = "NETWORK"
	// SSLCertPathKey is the path to the SSL certificate
	SSLCertPathKey = "SSL_CERT"
	// SSLKeyPathKey is the path to the SSL private key
	SSLKeyPathKey = "SSL_KEY"
	// DomainKey is TLD domain to be used for obtaining and renewing the TLS certificate
	DomainKey = "DOMAIN"
	// EmailKey is the email used for TLS certificates communications. Mandatory if TAXI_DOMAIN is given
	EmailKey = "EMAIL"
	// TraderListeningPortKey is the port where the gRPC Trader interface will listen on
	TraderListeningPortKey = "TRADER_LISTENING_PORT"
	// ElementsRPCEndpointKey is the url for the RPC interface of the Elements
	// node in the form protocol://user:password@host:port
	ElementsRPCEndpointKey = "ELEMENTS_RPC_ENDPOINT"

	// PriceEndpointKey is the GET HTTP endpoint to call to fetch price for the given trading pair
	// example response: { basePrice: "50000", quotePrice: "0.000023"}
	PriceEndpointKey = "PRICE_ENDPOINT"
	// EpxlorerEndpointKey is the Electrs-compatible Liquid explorer base URL to source blockchain data
	ExplorerEndpointKey = "EXPLORER_ENDPOINT"

	// BaseAssetKey is the asset hash of the base asset for the single market
	BaseAssetKey = "BASE_ASSET_ID"
	// BaseAssetKey is the asset hash of the base asset for the single market
	QuoteAssetKey = "QUOTE_ASSET_ID"

	// MnemonicKey are the keys used to hold funds
	MnemonicKey = "MNEMONIC"
)

var vip *viper.Viper

func init() {
	vip = viper.New()
	vip.SetEnvPrefix("ZION")
	vip.AutomaticEnv()

	vip.SetDefault(LogLevelKey, 5)
	vip.SetDefault(NetworkKey, "regtest")
	vip.SetDefault(TraderListeningPortKey, 9945)

	vip.SetDefault(ExplorerEndpointKey, "http://localhost:3001")
	vip.SetDefault(PriceEndpointKey, "http://localhost:4040/btc/usd")

	vip.SetDefault(BaseAssetKey, network.Regtest.AssetID)

	vip.SetDefault(MnemonicKey, "still double lounge behind shield idle pistol west dismiss hen august tray")

}

//GetString ...
func GetString(key string) string {
	return vip.GetString(key)
}

//GetInt ...
func GetInt(key string) int {
	return vip.GetInt(key)
}

//GetNetwork ...
func GetNetwork() *network.Network {
	if vip.GetString(NetworkKey) == network.Regtest.Name {
		return &network.Regtest
	}
	return &network.Liquid
}
