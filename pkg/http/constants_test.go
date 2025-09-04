package http

type customKeyType struct{}

var customKey = customKeyType{}

const (
	contentTypeJSON = "application/json"
	queryParamValue = "value"
	exampleURL      = "https://example.com"
	hello           = "Hello, World!"
	appOctetStream  = "application/octet-stream"
	textPlain       = "text/plain"
)
