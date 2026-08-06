// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	. "github.com/dave/jennifer/jen" //nolint:staticcheck,revive
)

func (r *transportRenderer) registerStreamFunc() Code {

	return Func().Params(Id("srv").Op("*").Id("Server")).
		Id("registerStream").
		Params(Id("stream").Id("streamCloser")).
		Params(Id("id").Uint64()).
		Block(
			Line(),
			If(Id("srv").Op("==").Nil().Op("||").Id("stream").Op("==").Nil()).Block(
				Return(),
			),
			Id("srv").Dot("streamsMu").Dot("Lock").Call(),
			Defer().Id("srv").Dot("streamsMu").Dot("Unlock").Call(),
			If(Id("srv").Dot("rootCtx").Op("!=").Nil().Op("&&").Id("srv").Dot("rootCtx").Dot("Err").Call().Op("!=").Nil()).Block(
				Id("stream").Dot("Close").Call(),
				Return(),
			),
			Id("srv").Dot("streamsSeq").Op("++"),
			Id("id").Op("=").Id("srv").Dot("streamsSeq"),
			Id("srv").Dot("streams").Index(Id("id")).Op("=").Id("stream"),
			Id("srv").Dot("streamsWG").Dot("Add").Call(Lit(1)),
			Return(),
		)
}

func (r *transportRenderer) unregisterStreamFunc() Code {

	return Func().Params(Id("srv").Op("*").Id("Server")).
		Id("unregisterStream").
		Params(Id("id").Uint64()).
		Block(
			Line(),
			If(Id("srv").Op("==").Nil().Op("||").Id("id").Op("==").Lit(0)).Block(
				Return(),
			),
			Id("srv").Dot("streamsMu").Dot("Lock").Call(),
			Delete(Id("srv").Dot("streams"), Id("id")),
			Id("srv").Dot("streamsMu").Dot("Unlock").Call(),
			Id("srv").Dot("streamsWG").Dot("Done").Call(),
		)
}

func (r *transportRenderer) cancelCloserType() Code {

	return Type().Id("cancelCloser").Struct(
		Id("cancel").Qual(PackageContext, "CancelFunc"),
	).Line().
		Func().Params(Id("closer").Id("cancelCloser")).
		Id("Close").
		Params().
		Block(
			Line(),
			Id("closer").Dot("cancel").Call(),
		)
}

func (r *transportRenderer) acquireWSFunc() Code {

	return Func().Params(Id("srv").Op("*").Id("Server")).
		Id("acquireWS").
		Params().
		Params(Id("ok").Bool()).
		Block(
			Line(),
			If(Id("srv").Dot("maxWSConnections").Op("<=").Lit(0)).Block(
				Id("ok").Op("=").True(),
				Return(),
			),
			For().Block(
				Id("current").Op(":=").Id("srv").Dot("wsConnections").Dot("Load").Call(),
				If(Id("current").Op(">=").Int64().Call(Id("srv").Dot("maxWSConnections"))).Block(
					Return(),
				),
				If(Id("srv").Dot("wsConnections").Dot("CompareAndSwap").Call(Id("current"), Id("current").Op("+").Lit(1))).Block(
					Id("ok").Op("=").True(),
					Return(),
				),
			),
		)
}

func (r *transportRenderer) releaseWSFunc() Code {

	return Func().Params(Id("srv").Op("*").Id("Server")).
		Id("releaseWS").
		Params().
		Block(
			Line(),
			If(Id("srv").Dot("maxWSConnections").Op("<=").Lit(0)).Block(
				Return(),
			),
			Id("srv").Dot("wsConnections").Dot("Add").Call(Lit(-1)),
		)
}

func (r *transportRenderer) acquireSSEFunc() Code {

	return Func().Params(Id("srv").Op("*").Id("Server")).
		Id("acquireSSE").
		Params().
		Params(Id("ok").Bool()).
		Block(
			Line(),
			If(Id("srv").Dot("maxSSEConnections").Op("<=").Lit(0)).Block(
				Id("ok").Op("=").True(),
				Return(),
			),
			For().Block(
				Id("current").Op(":=").Id("srv").Dot("sseConnections").Dot("Load").Call(),
				If(Id("current").Op(">=").Int64().Call(Id("srv").Dot("maxSSEConnections"))).Block(
					Return(),
				),
				If(Id("srv").Dot("sseConnections").Dot("CompareAndSwap").Call(Id("current"), Id("current").Op("+").Lit(1))).Block(
					Id("ok").Op("=").True(),
					Return(),
				),
			),
		)
}

func (r *transportRenderer) releaseSSEFunc() Code {

	return Func().Params(Id("srv").Op("*").Id("Server")).
		Id("releaseSSE").
		Params().
		Block(
			Line(),
			If(Id("srv").Dot("maxSSEConnections").Op("<=").Lit(0)).Block(
				Return(),
			),
			Id("srv").Dot("sseConnections").Dot("Add").Call(Lit(-1)),
		)
}

func (r *transportRenderer) effectiveSSEHeartbeatFunc() Code {

	streamPkg := r.pkgPath(r.outDir) + "/stream"
	return Func().Params(Id("srv").Op("*").Id("Server")).
		Id("effectiveSSEHeartbeat").
		Params().
		Params(Id("heartbeat").Qual(PackageTime, "Duration")).
		Block(
			Line(),
			Id("heartbeat").Op("=").Id("srv").Dot("sseHeartbeat"),
			If(Id("heartbeat").Op(">").Lit(0)).Block(
				Return(),
			),
			If(Id("srv").Dot("streamWriteTimeout").Op("<=").Lit(0)).Block(
				Return(),
			),
			Id("heartbeat").Op("=").Id("srv").Dot("streamWriteTimeout").Op("/").Lit(2),
			If(Id("heartbeat").Op(">").Qual(streamPkg, "DefaultSSEHeartbeat")).Block(
				Id("heartbeat").Op("=").Qual(streamPkg, "DefaultSSEHeartbeat"),
			),
			Return(),
		)
}
