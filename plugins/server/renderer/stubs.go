// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

func (r *contractRenderer) RenderTransportHTTP() error    { return nil }
func (r *contractRenderer) RenderTransportContext() error { return nil }
func (r *contractRenderer) RenderTransportLogger() error  { return nil }
func (r *contractRenderer) RenderTransportFiber() error   { return nil }
func (r *contractRenderer) RenderTransportHeader() error  { return nil }
func (r *contractRenderer) RenderTransportErrors() error  { return nil }
func (r *contractRenderer) RenderTransportServer() error  { return nil }
func (r *contractRenderer) RenderTransportOptions() error { return nil }
func (r *contractRenderer) RenderTransportMetrics() error { return nil }
func (r *contractRenderer) RenderTransportVersion() error { return nil }
func (r *contractRenderer) RenderTransportJsonRPC() error { return nil }

func (r *transportRenderer) RenderHTTP() error       { return nil }
func (r *transportRenderer) RenderServer() error     { return nil }
func (r *transportRenderer) RenderExchange() error   { return nil }
func (r *transportRenderer) RenderMiddleware() error { return nil }
func (r *transportRenderer) RenderTrace() error      { return nil }
func (r *transportRenderer) RenderMetrics() error    { return nil }
func (r *transportRenderer) RenderLogger() error     { return nil }
func (r *transportRenderer) RenderJsonRPC() error    { return nil }
func (r *transportRenderer) RenderREST() error       { return nil }
