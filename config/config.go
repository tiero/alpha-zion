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
)

var vip *viper.Viper

func init() {
	vip = viper.New()
	vip.SetEnvPrefix("ZION")
	vip.AutomaticEnv()

	vip.SetDefault(LogLevelKey, 5)
	vip.SetDefault(NetworkKey, "regtest")
	vip.SetDefault(TraderListeningPortKey, 9945)
	vip.SetDefault(ElementsRPCEndpointKey, "http://admin1:123@127.0.0.1:7041")

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
