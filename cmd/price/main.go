package main

import (
	"net/http"
	"strings"

	"github.com/bitfinexcom/bitfinex-api-go/v2/rest"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/shopspring/decimal"

	"github.com/tiero/zion/internal/core/ports"
)

func main() {
	// Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Routes
	e.GET("/:base/:quote", prices)

	// Start server
	e.Logger.Fatal(e.Start(":4040"))
}

// Handler ...
func prices(c echo.Context) error {

	// check the request
	pReq := new(ports.PriceRequest)
	if err := c.Bind(pReq); err != nil {
		return err
	}

	if strings.ToUpper(pReq.BaseAsset) != "BTC" || strings.ToUpper(pReq.QuoteAsset) != "USD" {
		return c.String(http.StatusBadRequest, "trading pair not supported")
	}

	//bitfinex
	bitfinexClient := rest.NewClient()

	tickerPrice, err := bitfinexClient.Tickers.Get("tBTCUSD")
	if err != nil {
		return err
	}

	basePrice := decimal.NewFromFloat(1 / tickerPrice.LastPrice)
	quotePrice := decimal.NewFromFloat(tickerPrice.LastPrice)

	pRes := &ports.PriceResponse{
		BasePrice:  basePrice.String(),
		QuotePrice: quotePrice.String(),
	}

	return c.JSON(http.StatusOK, pRes)
}
