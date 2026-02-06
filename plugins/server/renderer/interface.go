// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

type Renderer interface {
	// RenderHTTP генерирует HTTP обработчики для контракта.
	RenderHTTP() error

	// RenderServer генерирует серверную обертку для контракта.
	RenderServer() error

	// RenderExchange генерирует структуры обмена данными (request/response).
	RenderExchange() error

	// RenderMiddleware генерирует типы middleware.
	RenderMiddleware() error

	// RenderTrace генерирует middleware для трейсинга.
	RenderTrace() error

	// RenderMetrics генерирует middleware для метрик.
	RenderMetrics() error

	// RenderLogger генерирует middleware для логирования.
	RenderLogger() error

	// RenderJsonRPC генерирует JSON-RPC обработчики.
	RenderJsonRPC() error

	// RenderREST генерирует REST обработчики.
	RenderREST() error

	// Транспортные файлы (генерируются один раз для всех контрактов)
	RenderTransportHTTP() error
	RenderTransportContext() error
	RenderTransportLogger() error
	RenderTransportFiber() error
	RenderTransportHeader() error
	RenderTransportErrors() error
	RenderTransportServer() error
	RenderTransportOptions() error
	RenderTransportMetrics() error
	RenderTransportVersion() error
	RenderTransportJsonRPC() error
}
