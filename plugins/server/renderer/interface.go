// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

type ContractRenderer interface {
	RenderHTTP() (err error)
	RenderServer() (err error)
	RenderExchange() (err error)
	RenderMiddleware() (err error)
	RenderTrace() (err error)
	RenderMetrics() (err error)
	RenderLogger() (err error)
	RenderJSONRPC() (err error)
	RenderREST() (err error)
	RenderWebSocket() (err error)
	RenderSSE() (err error)
}

type TransportRenderer interface {
	RenderTransportContext() (err error)
	RenderTransportLimits() (err error)
	RenderTransportFiber() (err error)
	RenderTransportHeader() (err error)
	RenderTransportErrors() (err error)
	RenderTransportServer() (err error)
	RenderTransportOptions() (err error)
	RenderTransportAuth() (err error)
	RenderTransportSecurity() (err error)
	RenderTransportMetrics() (err error)
	RenderTransportVersion() (err error)
	RenderTransportJSONRPC() (err error)
	RenderLogViews() (err error)
}
