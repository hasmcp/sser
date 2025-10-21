package entity

import (
	"github.com/valyala/fasthttp"
)

type (
	Err struct {
		Code    int
		Message string
		Details map[string]any
	}
)

func (e Err) Error() string {
	return e.Message
}

const (
	ErrorCodeBadRequest                    = fasthttp.StatusBadRequest
	ErrorCodeUnauthorized                  = fasthttp.StatusUnauthorized
	ErrorCodePaymentRequired               = fasthttp.StatusPaymentRequired
	ErrorCodeForbidden                     = fasthttp.StatusForbidden
	ErrorCodeNotFound                      = fasthttp.StatusNotFound
	ErrorCodeMethodNotAllowed              = fasthttp.StatusMethodNotAllowed
	ErrorCodeNotAcceptable                 = fasthttp.StatusNotAcceptable
	ErrorCodeProxyAuthRequired             = fasthttp.StatusProxyAuthRequired
	ErrorCodeRequestTimeout                = fasthttp.StatusRequestTimeout
	ErrorCodeConflict                      = fasthttp.StatusConflict
	ErrorCodeGone                          = fasthttp.StatusGone
	ErrorCodeLengthRequired                = fasthttp.StatusLengthRequired
	ErrorCodePreconditionFailed            = fasthttp.StatusPreconditionFailed
	ErrorCodeRequestEntityTooLarge         = fasthttp.StatusRequestEntityTooLarge
	ErrorCodeRequestURITooLong             = fasthttp.StatusRequestURITooLong
	ErrorCodeUnsupportedMediaType          = fasthttp.StatusUnsupportedMediaType
	ErrorCodeRequestedRangeNotSatisfiable  = fasthttp.StatusRequestedRangeNotSatisfiable
	ErrorCodeExpectationFailed             = fasthttp.StatusExpectationFailed
	ErrorCodeMisdirectedRequest            = fasthttp.StatusMisdirectedRequest
	ErrorCodeUnprocessableEntity           = fasthttp.StatusUnprocessableEntity
	ErrorCodeLocked                        = fasthttp.StatusLocked
	ErrorCodeFailedDependency              = fasthttp.StatusFailedDependency
	ErrorCodeUpgradeRequired               = fasthttp.StatusUpgradeRequired
	ErrorCodePreconditionRequired          = fasthttp.StatusPreconditionRequired
	ErrorCodeTooManyRequests               = fasthttp.StatusTooManyRequests
	ErrorCodeRequestHeaderFieldsTooLarge   = fasthttp.StatusRequestHeaderFieldsTooLarge
	ErrorCodeUnavailableForLegalReasons    = fasthttp.StatusUnavailableForLegalReasons
	ErrorCodeInternalServerError           = fasthttp.StatusInternalServerError
	ErrorCodeNotImplemented                = fasthttp.StatusNotImplemented
	ErrorCodeBadGateway                    = fasthttp.StatusBadGateway
	ErrorCodeServiceUnavailable            = fasthttp.StatusServiceUnavailable
	ErrorCodeGatewayTimeout                = fasthttp.StatusGatewayTimeout
	ErrorCodeHTTPVersionNotSupported       = fasthttp.StatusHTTPVersionNotSupported
	ErrorCodeVariantAlsoNegotiates         = fasthttp.StatusVariantAlsoNegotiates
	ErrorCodeInsufficientStorage           = fasthttp.StatusInsufficientStorage
	ErrorCodeLoopDetected                  = fasthttp.StatusLoopDetected
	ErrorCodeNotExtended                   = fasthttp.StatusNotExtended
	ErrorCodeNetworkAuthenticationRequired = fasthttp.StatusNetworkAuthenticationRequired
)
