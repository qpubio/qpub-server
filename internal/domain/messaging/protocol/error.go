package protocol

// Code represents the internal error code
type Code int

const (
	// General errors
	ErrGeneral Code = 10000

	// Client errors
	ErrBadRequest         Code = 40000 // Bad request
	ErrInvalidMessage     Code = 40001 // Invalid message format/structure
	ErrInvalidAction      Code = 40002 // Invalid action requested
	ErrInvalidChannel     Code = 40003 // Invalid channel name
	ErrUnauthorized       Code = 40100 // Unauthorized
	ErrForbidden          Code = 40300 // Forbidden
	ErrNotFound           Code = 40400 // Not found
	ErrSubscriptionClosed Code = 40401 // Subscription is closed
	ErrRateLimited        Code = 42900 // Rate limit exceeded

	// Server errors
	ErrInternal          Code = 50000 // Internal server error
	ErrBrokerUnavailable Code = 50001 // Message broker is unavailable
	ErrPublishFailed     Code = 50002 // Failed to publish message
	ErrSubscribeFailed   Code = 50003 // Failed to subscribe to channel
	ErrUnsubscribeFailed Code = 50004 // Failed to unsubscribe from channel
	ErrConnectionClosed  Code = 50005 // Connection was closed
	ErrTimeout           Code = 50400 // Timeout
)

// Href represents the error shortlink path
type Href string

const (
	// General errors
	HrefGeneral Href = "https://qpb.li/err10000"

	// Client errors
	HrefBadRequest         Href = "https://qpb.li/err40000"
	HrefInvalidMessage     Href = "https://qpb.li/err40001"
	HrefInvalidAction      Href = "https://qpb.li/err40002"
	HrefInvalidChannel     Href = "https://qpb.li/err40003"
	HrefUnauthorized       Href = "https://qpb.li/err40100"
	HrefForbidden          Href = "https://qpb.li/err40300"
	HrefNotFound           Href = "https://qpb.li/err40400"
	HrefSubscriptionClosed Href = "https://qpb.li/err40401"
	HrefRateLimited        Href = "https://qpb.li/err42900"

	// Server errors
	HrefInternal          Href = "https://qpb.li/err50000"
	HrefBrokerUnavailable Href = "https://qpb.li/err50001"
	HrefPublishFailed     Href = "https://qpb.li/err50002"
	HrefSubscribeFailed   Href = "https://qpb.li/err50003"
	HrefUnsubscribeFailed Href = "https://qpb.li/err50004"
	HrefConnectionClosed  Href = "https://qpb.li/err50005"
	HrefTimeout           Href = "https://qpb.li/err50400"
)

// StatusCode represents the HTTP status code
// https://developer.mozilla.org/en-US/docs/Web/HTTP/Status
type StatusCode int

const (
	// 1xx Informational
	StatusCodeContinue           StatusCode = 100
	StatusCodeSwitchingProtocols StatusCode = 101
	StatusCodeProcessing         StatusCode = 102
	StatusCodeEarlyHints         StatusCode = 103

	// 2xx Successful
	StatusCodeOK                   StatusCode = 200
	StatusCodeCreated              StatusCode = 201
	StatusCodeAccepted             StatusCode = 202
	StatusCodeNonAuthoritativeInfo StatusCode = 203
	StatusCodeNoContent            StatusCode = 204
	StatusCodeResetContent         StatusCode = 205
	StatusCodePartialContent       StatusCode = 206

	// 3xx Redirection
	StatusCodeMultipleChoices   StatusCode = 300
	StatusCodeMovedPermanently  StatusCode = 301
	StatusCodeFound             StatusCode = 302
	StatusCodeSeeOther          StatusCode = 303
	StatusCodeNotModified       StatusCode = 304
	StatusCodeUseProxy          StatusCode = 305
	StatusCodeTemporaryRedirect StatusCode = 307
	StatusCodePermanentRedirect StatusCode = 308

	// 4xx Client errors
	StatusCodeBadRequest   StatusCode = 400
	StatusCodeUnauthorized StatusCode = 401
	StatusCodeForbidden    StatusCode = 403
	StatusCodeNotFound     StatusCode = 404
	StatusCodeTooManyRequests StatusCode = 429

	// 5xx Server errors
	StatusCodeInternal                      StatusCode = 500
	StatusCodeNotImplemented                StatusCode = 501
	StatusCodeBadGateway                    StatusCode = 502
	StatusCodeServiceUnavailable            StatusCode = 503
	StatusCodeGatewayTimeout                StatusCode = 504
	StatusCodeHTTPVersionNotSupported       StatusCode = 505
	StatusCodeVariantAlsoNegotiates         StatusCode = 506
	StatusCodeInsufficientStorage           StatusCode = 507
	StatusCodeLoopDetected                  StatusCode = 508
	StatusCodeNotExtended                   StatusCode = 510
	StatusCodeNetworkAuthenticationRequired StatusCode = 511
)
