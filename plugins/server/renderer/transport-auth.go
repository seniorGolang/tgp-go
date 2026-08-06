// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" //nolint:staticcheck,revive
)

func (r *transportRenderer) RenderTransportAuth() (err error) {

	if !r.hasBearerSecurity() {
		return
	}

	authPath := path.Join(r.outDir, "auth.go")
	bearerPkg := r.pkgPath(r.outDir) + "/bearer"

	srcFile := NewSrcFile(filepath.Base(r.outDir))
	srcFile.ImportName(PackageContext, "context")
	srcFile.ImportName(PackageErrors, "errors")
	srcFile.ImportName(PackageFmt, "fmt")
	srcFile.ImportName(PackageStrings, "strings")
	srcFile.ImportName(PackageStdJSON, "json")
	srcFile.ImportName(PackageHTTP, "http")
	srcFile.ImportName(PackageTLS, "tls")
	srcFile.ImportName(bearerPkg, "bearer")

	srcFile.Line().Var().Id("errBearerForbidden").Op("=").Qual(PackageErrors, "New").Call(Lit("bearer: forbidden"))

	srcFile.Line().Type().Id("AuthOption").Func().Params(Id("cfg").Op("*").Id("authConfig"))

	srcFile.Line().Type().Id("authConfig").Struct(
		Id("authorizationServer").String(),
		Id("resource").String(),
		Id("audience").String(),
		Id("client").Op("*").Qual(PackageHTTP, "Client"),
		Id("tlsConfig").Op("*").Qual(PackageTLS, "Config"),
		Id("bind").Func().Params(Id("ctx").Qual(PackageContext, "Context"), Id("claims").Map(String()).Any()).Params(Id("out").Qual(PackageContext, "Context")),
		Id("allow").Func().Params(Id("ctx").Qual(PackageContext, "Context"), Id("claims").Map(String()).Any()).Params(Id("err").Error()),
		Id("verify").Func().Params(Id("ctx").Qual(PackageContext, "Context"), Id("token").String()).Params(Id("subject").String(), Id("claims").Map(String()).Any(), Id("err").Error()),
	)

	srcFile.Line().Func().Id("AuthorizationServer").Params(Id("url").String()).Params(Id("opt").Id("AuthOption")).Block(
		Return(Func().Params(Id("cfg").Op("*").Id("authConfig")).Block(
			Id("cfg").Dot("authorizationServer").Op("=").Id("url"),
		)),
	)
	srcFile.Line().Func().Id("Resource").Params(Id("url").String()).Params(Id("opt").Id("AuthOption")).Block(
		Return(Func().Params(Id("cfg").Op("*").Id("authConfig")).Block(
			Id("cfg").Dot("resource").Op("=").Id("url"),
		)),
	)
	srcFile.Line().Func().Id("Audience").Params(Id("audience").String()).Params(Id("opt").Id("AuthOption")).Block(
		Return(Func().Params(Id("cfg").Op("*").Id("authConfig")).Block(
			Id("cfg").Dot("audience").Op("=").Id("audience"),
		)),
	)
	srcFile.Line().Func().Id("ClientHTTP").Params(Id("client").Op("*").Qual(PackageHTTP, "Client")).Params(Id("opt").Id("AuthOption")).Block(
		Return(Func().Params(Id("cfg").Op("*").Id("authConfig")).Block(
			If(Id("client").Op("!=").Nil()).Block(
				Id("cfg").Dot("client").Op("=").Id("client"),
			),
		)),
	)
	srcFile.Line().Func().Id("ConfigTLS").Params(Id("tlsConfig").Op("*").Qual(PackageTLS, "Config")).Params(Id("opt").Id("AuthOption")).Block(
		Return(Func().Params(Id("cfg").Op("*").Id("authConfig")).Block(
			Id("cfg").Dot("tlsConfig").Op("=").Id("tlsConfig"),
		)),
	)
	srcFile.Line().Func().Id("Bind").
		Params(Id("bind").Func().Params(Id("ctx").Qual(PackageContext, "Context"), Id("claims").Map(String()).Any()).Params(Id("out").Qual(PackageContext, "Context"))).
		Params(Id("opt").Id("AuthOption")).
		Block(
			Return(Func().Params(Id("cfg").Op("*").Id("authConfig")).Block(
				Id("cfg").Dot("bind").Op("=").Id("bind"),
			)),
		)
	srcFile.Line().Func().Id("Allow").
		Params(Id("allow").Func().Params(Id("ctx").Qual(PackageContext, "Context"), Id("claims").Map(String()).Any()).Params(Id("err").Error())).
		Params(Id("opt").Id("AuthOption")).
		Block(
			Return(Func().Params(Id("cfg").Op("*").Id("authConfig")).Block(
				Id("cfg").Dot("allow").Op("=").Id("allow"),
			)),
		)
	srcFile.Line().Func().Id("Verify").
		Params(Id("verify").Func().Params(Id("ctx").Qual(PackageContext, "Context"), Id("token").String()).Params(Id("subject").String(), Id("claims").Map(String()).Any(), Id("err").Error())).
		Params(Id("opt").Id("AuthOption")).
		Block(
			Return(Func().Params(Id("cfg").Op("*").Id("authConfig")).Block(
				Id("cfg").Dot("verify").Op("=").Id("verify"),
			)),
		)

	srcFile.Line().Type().Id("authenticator").Struct(
		Id("verifier").Op("*").Qual(bearerPkg, "Verifier"),
		Id("bind").Func().Params(Id("ctx").Qual(PackageContext, "Context"), Id("claims").Map(String()).Any()).Params(Id("out").Qual(PackageContext, "Context")),
		Id("allow").Func().Params(Id("ctx").Qual(PackageContext, "Context"), Id("claims").Map(String()).Any()).Params(Id("err").Error()),
		Id("verify").Func().Params(Id("ctx").Qual(PackageContext, "Context"), Id("token").String()).Params(Id("subject").String(), Id("claims").Map(String()).Any(), Id("err").Error()),
	)

	srcFile.Line().Func().Params(Id("auth").Op("*").Id("authenticator")).Id("Authenticate").
		Params(Id("ctx").Qual(PackageContext, "Context"), Id("token").String()).
		Params(Id("out").Qual(PackageContext, "Context"), Id("subject").String(), Err().Error()).
		BlockFunc(func(bg *Group) {
			bg.Var().Id("claims").Map(String()).Any()
			bg.If(Id("auth").Dot("verify").Op("!=").Nil()).Block(
				If(List(Id("subject"), Id("claims"), Err()).Op("=").Id("auth").Dot("verify").Call(Id("ctx"), Id("token")).Op(";").Err().Op("!=").Nil()).Block(Return()),
			).Else().Block(
				If(List(Id("subject"), Id("claims"), Err()).Op("=").Id("auth").Dot("verifier").Dot("Verify").Call(Id("ctx"), Id("token")).Op(";").Err().Op("!=").Nil()).Block(Return()),
			)
			bg.If(Id("auth").Dot("allow").Op("!=").Nil()).Block(
				If(Err().Op("=").Id("auth").Dot("allow").Call(Id("ctx"), Id("claims")).Op(";").Err().Op("!=").Nil()).Block(
					Err().Op("=").Qual(PackageFmt, "Errorf").Call(Lit("%w: %w"), Id("errBearerForbidden"), Err()),
					Return(),
				),
			)
			bg.Id("out").Op("=").Id("ctx")
			bg.If(Id("auth").Dot("bind").Op("!=").Nil()).Block(
				Id("out").Op("=").Id("auth").Dot("bind").Call(Id("ctx"), Id("claims")),
			)
			bg.Return()
		})

	srcFile.Line().Func().Id("Auth").Params(Id("opts").Op("...").Id("AuthOption")).Id("Option").Block(
		Return(Func().Params(Id("srv").Op("*").Id("Server")).BlockFunc(func(bg *Group) {
			bg.If(Id("srv").Dot("auth").Op("!=").Nil()).Block(Return())
			bg.Id("cfg").Op(":=").Id("authConfig").Values()
			bg.For(List(Id("_"), Id("opt")).Op(":=").Range().Id("opts")).Block(Id("opt").Call(Op("&").Id("cfg")))
			bg.Id("cfg").Dot("authorizationServer").Op("=").Qual(PackageStrings, "TrimSpace").Call(Id("cfg").Dot("authorizationServer"))
			bg.Id("cfg").Dot("resource").Op("=").Qual(PackageStrings, "TrimSpace").Call(Id("cfg").Dot("resource"))
			bg.Id("cfg").Dot("audience").Op("=").Qual(PackageStrings, "TrimSpace").Call(Id("cfg").Dot("audience"))
			bg.Id("auth").Op(":=").Op("&").Id("authenticator").Values(Dict{
				Id("allow"):  Id("cfg").Dot("allow"),
				Id("bind"):   Id("cfg").Dot("bind"),
				Id("verify"): Id("cfg").Dot("verify"),
			})
			bg.If(Id("cfg").Dot("verify").Op("==").Nil()).Block(
				Id("auth").Dot("verifier").Op("=").Qual(bearerPkg, "NewVerifier").Call(Qual(bearerPkg, "Config").Values(Dict{
					Id("Audience"):            Id("cfg").Dot("audience"),
					Id("AuthorizationServer"): Id("cfg").Dot("authorizationServer"),
					Id("Client"):              Id("cfg").Dot("client"),
					Id("TLS"):                 Id("cfg").Dot("tlsConfig"),
				})),
			)
			bg.Id("srv").Dot("auth").Op("=").Id("auth")
			bg.If(Id("cfg").Dot("resource").Op("!=").Lit("")).BlockFunc(func(rg *Group) {
				rg.Var().Id("servers").Index().String()
				rg.If(Id("cfg").Dot("authorizationServer").Op("!=").Lit("")).Block(
					Id("servers").Op("=").Index().String().Values(Id("cfg").Dot("authorizationServer")),
				)
				rg.Id("srv").Dot("authChallenge").Op("=").Lit(`Bearer resource_metadata="`).Op("+").Qual(PackageStrings, "TrimRight").Call(Id("cfg").Dot("resource"), Lit("/")).Op("+").Qual(bearerPkg, "WellKnownPath").Op("+").Lit(`"`)
				rg.List(Id("srv").Dot("authMetadata"), Id("_")).Op("=").Qual(PackageStdJSON, "Marshal").Call(Qual(bearerPkg, "Metadata").Values(Dict{
					Id("Audience"):               Id("cfg").Dot("audience"),
					Id("AuthorizationServers"):   Id("servers"),
					Id("BearerMethodsSupported"): Qual(bearerPkg, "MethodsHeader"),
					Id("Resource"):               Id("cfg").Dot("resource"),
				}))
			})
		})),
	)

	return srcFile.Save(authPath)
}
