package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"
	"github.com/vulpemventures/go-elements/network"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	pbtrader "github.com/tdex-network/tdex-protobuf/generated/go/trade"
	"github.com/tiero/zion/internal/core/application"
	grpchandler "github.com/tiero/zion/internal/interface/grpc/handler"
	"google.golang.org/grpc"
)

const (
	base        = "144c654344aa716d6f3abcc1ca90e5641e4e2a7f633bc09fe3baf64585819a49"
	quote       = "bd44ed3b4b6eafe98de0726dce1f79ccbd9c457d9ddc1d44389e58f53b9607f8"
	privateKey  = "bfd87b3d29e1c0846ed293d4bdc7b78d62598a92d18ae69c153558906063df9b"
	explorerUrl = "https://blockstream.info/liquidtestnet/api"
)

func main() {

	// trade service
	tradeService, err := application.NewTradeService(
		privateKey,
		base,
		quote,
		explorerUrl,
		network.Regtest.AssetID,
		&network.Regtest,
	)
	if err != nil {
		log.WithError(err).Fatal()
	}

	// Port
	traderAddress := fmt.Sprintf(":%+v", 9945)

	// Grpc Server
	traderGrpcServer := grpc.NewServer()

	// Grpc Handler
	tradeHandler := grpchandler.NewTraderHandler(tradeService)

	// Register proto implementations on Trader interface
	pbtrader.RegisterTradeServer(traderGrpcServer, tradeHandler)

	log.Info("starting ziond")

	defer stop(traderGrpcServer)

	httpL, err := net.Listen("tcp", traderAddress)
	if err != nil {
		log.WithError(err).Fatal()
	}

	grpcWebServer := grpcweb.WrapServer(
		traderGrpcServer,
		grpcweb.WithCorsForRegisteredEndpointsOnly(false),
		grpcweb.WithOriginFunc(func(origin string) bool { return true }),
	)

	go http.Serve(httpL, http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if grpcWebServer.IsGrpcWebRequest(req) || grpcWebServer.IsAcceptableGrpcCorsRequest(req) {
			grpcWebServer.ServeHTTP(resp, req)
		}
	}))

	log.Info("trader interface is listening on " + traderAddress)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	<-sigChan

	log.Info("shutting down ziond")

}

func stop(
	traderServer *grpc.Server,
) {
	traderServer.Stop()
	log.Debug("disabled trader interface")
	log.Debug("exiting")
}
