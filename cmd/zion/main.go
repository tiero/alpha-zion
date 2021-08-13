package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	log "github.com/sirupsen/logrus"
	"github.com/soheilhy/cmux"
	"github.com/tiero/zion/internal/config"
	"github.com/tiero/zion/internal/core/application"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"

	pbtrader "github.com/tdex-network/tdex-protobuf/generated/go/trade"
	grpchandler "github.com/tiero/zion/internal/interface/grpc/handler"
)

func main() {
	log.SetLevel(log.Level(config.GetInt(config.LogLevelKey)))

	// wallet service
	walletService, err := application.NewWalletService(
		config.GetString(config.MnemonicKey),
		config.GetString(config.ExplorerEndpointKey),
		config.GetNetwork(),
	)
	if err != nil {
		log.WithError(err).Panic("error starting on wallet service")
	}

	log.Info("deposit address: ", walletService.Address().ConfidentialAddress)

	// trade service
	tradeService, err := application.NewTradeServiceWithExplorer(
		walletService,
		config.GetString(config.PriceEndpointKey),
		config.GetString(config.BaseAssetKey),
		config.GetString(config.QuoteAssetKey),
	)
	if err != nil {
		log.WithError(err).Panic("error starting on trade service")
	}

	// Ports
	traderAddress := fmt.Sprintf(":%+v", config.GetInt(config.TraderListeningPortKey))
	// Grpc Server
	traderGrpcServer := grpc.NewServer()
	//traderHandler := &pbtrader.UnimplementedTradeServer{}
	traderHandler := grpchandler.NewTraderHandler(tradeService)

	// Register proto implementations on Trader interface
	pbtrader.RegisterTradeServer(traderGrpcServer, traderHandler)

	srv := &Server{
		address: traderAddress,
		grpc:    traderGrpcServer,
	}

	log.Info("starting zion")

	defer srv.stop()

	// Serve grpc and grpc-web multiplexed on the same port
	srv.serveMux()

	log.Info("trader interface is listening on " + srv.address)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	<-sigChan

	log.Info("shutting down zion")
}

type Server struct {
	address string
	grpc    *grpc.Server
}

func (s *Server) stop() {
	s.grpc.GracefulStop()
	//s.grpc.Stop()
	log.Debug("disabled trader interface")
	log.Debug("exiting")
}

func (s *Server) serveMux() {
	lis, err := net.Listen("tcp", s.address)
	if err != nil {
		log.Fatal("error starting http server: ", err)
	}

	cMux := cmux.New(lis)
	grpcListener := cMux.MatchWithWriters(cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"))
	grpcWebHTTP2Listener := cMux.Match(cmux.HTTP2())
	grpcWebHTTP1Listener := cMux.Match(cmux.HTTP1())

	wrappedGrpc := grpcweb.WrapServer(
		s.grpc,
		grpcweb.WithCorsForRegisteredEndpointsOnly(false),
		grpcweb.WithOriginFunc(func(origin string) bool { return true }),
	)
	handleGrpcWebReq := func(w http.ResponseWriter, req *http.Request) {
		if isValidRequest(req) {
			wrappedGrpc.ServeHTTP(w, req)
			return
		}

		msg := "received a request that could not be matched to grpc or grpc-web"
		log.Error(msg)
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(msg))
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleGrpcWebReq)

	httpSrv := &http.Server{
		Addr:    s.address,
		Handler: mux,
	}

	http2Srv := &http.Server{
		Addr:    s.address,
		Handler: h2c.NewHandler(http.HandlerFunc(handleGrpcWebReq), &http2.Server{}),
	}

	serveGRPC(s.grpc, grpcListener)
	serveHTTP(http2Srv, grpcWebHTTP2Listener)
	serveHTTP(httpSrv, grpcWebHTTP1Listener)

	go cMux.Serve()
}

func serveHTTP(s *http.Server, l net.Listener) {
	go func() {
		if err := s.Serve(l); err != nil {
			log.Fatal("error starting http server: ", err)
		}
	}()
}

func serveGRPC(s *grpc.Server, l net.Listener) {
	go func() {
		if err := s.Serve(l); err != nil {
			log.Fatal("error starting grpc server: ", err)
		}
	}()
}

func isValidRequest(req *http.Request) bool {
	return isValidGrpcWebOptionRequest(req) || isValidGrpcWebRequest(req)
}

func isValidGrpcWebRequest(req *http.Request) bool {
	return req.Method == http.MethodPost && isValidGrpcContentTypeHeader(req.Header.Get("content-type"))
}

func isValidGrpcContentTypeHeader(contentType string) bool {
	return strings.HasPrefix(contentType, "application/grpc-web-text") ||
		strings.HasPrefix(contentType, "application/grpc-web")
}

func isValidGrpcWebOptionRequest(req *http.Request) bool {
	accessControlHeader := req.Header.Get("Access-Control-Request-Headers")
	return req.Method == http.MethodOptions &&
		strings.Contains(accessControlHeader, "x-grpc-web") &&
		strings.Contains(accessControlHeader, "content-type")
}
