package ports

// PriceRequest ...
type PriceRequest struct {
	BaseAsset  string `param:"base"`
	QuoteAsset string `param:"quote"`
}

// PriceResponse
type PriceResponse struct {
	BasePrice  string `json:"basePrice"`
	QuotePrice string `json:"quotePrice"`
}
