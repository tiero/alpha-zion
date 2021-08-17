package main

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/caddyserver/certmagic"
	"github.com/go-acme/lego/challenge/tlsalpn01"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/soheilhy/cmux"
	"golang.org/x/net/http2"
	"google.golang.org/grpc"

	log "github.com/sirupsen/logrus"

	pbtrader "github.com/tdex-network/tdex-protobuf/generated/go/trade"
	grpchandler "github.com/tiero/zion/internal/interface/grpc/handler"

	"github.com/tiero/zion/internal/config"
	"github.com/tiero/zion/internal/core/application"

	"github.com/bitfinexcom/bitfinex-api-go/v2/rest"
)

func main() {

	// application layer

	//bitfinex
	bitfinexClient := rest.NewClient()

	// elements node
	elementService, err := application.NewElementsService(config.GetString(config.ElementsRPCEndpointKey))
	if err != nil {
		log.WithError(err).Panic("error starting on elements service")
	}

	// trade service
	tradeService, err := application.NewTradeServiceWithElements(
		bitfinexClient,
		elementService,
		config.GetNetwork(),
	)
	if err != nil {
		log.WithError(err).Panic("error starting on trade service")
	}

	// Ports
	traderAddress := fmt.Sprintf(":%+v", config.GetInt(config.TraderListeningPortKey))

	// Grpc Server
	traderGrpcServer := grpc.NewServer()
	traderHandler := grpchandler.NewTraderHandler(tradeService)

	// Register proto implementations on Trader interface
	pbtrader.RegisterTradeServer(traderGrpcServer, traderHandler)

	log.Info("starting ziond")

	defer stop(traderGrpcServer)

	// Serve grpc and grpc-web multiplexed on the same port
	if err := serveMux(traderAddress, true, traderGrpcServer); err != nil {
		log.WithError(err).Panic("error listening on trader interface")
	}

	log.Info("trader interface is listening on " + traderAddress)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	<-sigChan

	log.Info("shutting down ziond")

}

func serveMux(address string, withSsl bool, grpcServer *grpc.Server) error {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	// If the DOMAIN and EMAIL is given, we are going to spinup a basic http server to handle the http ACME challenge
	if domain := config.GetString(config.DomainKey); domain != "" && withSsl {

		magic := certmagic.NewDefault()
		myACME := certmagic.NewACMEManager(magic, certmagic.ACMEManager{
			Agreed: true,
			Email:  config.GetString(config.EmailKey),
		})
		magic.Issuer = myACME

		// we run this server only for handling HTTP ACME challange
		go http.ListenAndServe(":80", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if myACME.HandleHTTPChallenge(w, r) {
				return
			}
		}))

		err := magic.ManageAsync(context.Background(), []string{domain})
		if err != nil {
			return fmt.Errorf("manage sync: %w", err)
		}

		config := &tls.Config{
			NextProtos: []string{"http/1.1", http2.NextProtoTLS, "h2-14"}, // h2-14 is just for compatibility. will be eventually removed.
		}
		config.GetCertificate = magic.GetCertificate
		config.NextProtos = append(config.NextProtos, tlsalpn01.ACMETLS1Protocol)

		lis = tls.NewListener(lis, config)
	}

	if sslKey := config.GetString(config.SSLKeyPathKey); sslKey != "" && withSsl {
		certificate, err := tls.LoadX509KeyPair(config.GetString(config.SSLCertPathKey), sslKey)
		if err != nil {
			return err
		}

		const requiredCipher = tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
		config := &tls.Config{
			CipherSuites: []uint16{requiredCipher},
			NextProtos:   []string{"http/1.1", http2.NextProtoTLS, "h2-14"}, // h2-14 is just for compatibility. will be eventually removed.
			Certificates: []tls.Certificate{certificate},
		}
		config.Rand = rand.Reader

		lis = tls.NewListener(lis, config)
	}

	mux := cmux.New(lis)
	grpcL := mux.MatchWithWriters(cmux.HTTP2MatchHeaderFieldPrefixSendSettings("content-type", "application/grpc"))
	httpL := mux.Match(cmux.HTTP1Fast())

	grpcWebServer := grpcweb.WrapServer(
		grpcServer,
		grpcweb.WithCorsForRegisteredEndpointsOnly(false),
		grpcweb.WithOriginFunc(func(origin string) bool { return true }),
	)

	go grpcServer.Serve(grpcL)
	go http.Serve(httpL, http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if isValidRequest(req) {
			grpcWebServer.ServeHTTP(resp, req)
		}
	}))

	go mux.Serve()
	return nil
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

func stop(
	traderServer *grpc.Server,
) {
	traderServer.Stop()
	log.Debug("disabled trader interface")
	log.Debug("exiting")
}
