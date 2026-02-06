// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package model

import (
	"tgp/internal/tags"
)

type TypeKind string

const (
	TypeKindString  TypeKind = "string"
	TypeKindInt     TypeKind = "int"
	TypeKindInt8    TypeKind = "int8"
	TypeKindInt16   TypeKind = "int16"
	TypeKindInt32   TypeKind = "int32"
	TypeKindInt64   TypeKind = "int64"
	TypeKindUint    TypeKind = "uint"
	TypeKindUint8   TypeKind = "uint8"
	TypeKindUint16  TypeKind = "uint16"
	TypeKindUint32  TypeKind = "uint32"
	TypeKindUint64  TypeKind = "uint64"
	TypeKindFloat32 TypeKind = "float32"
	TypeKindFloat64 TypeKind = "float64"
	TypeKindBool    TypeKind = "bool"
	TypeKindByte    TypeKind = "byte"
	TypeKindRune    TypeKind = "rune"
	TypeKindError   TypeKind = "error"
	TypeKindAny     TypeKind = "any"

	TypeKindArray     TypeKind = "array"
	TypeKindMap       TypeKind = "map"
	TypeKindChan      TypeKind = "chan"
	TypeKindStruct    TypeKind = "struct"
	TypeKindInterface TypeKind = "interface"
	TypeKindFunction  TypeKind = "function"
	TypeKindAlias     TypeKind = "alias"
)

type Project struct {
	Version      string `json:"version"`
	ModulePath   string `json:"modulePath"`
	ContractsDir string `json:"contractsDir"`

	Git *GitInfo `json:"git,omitempty"`

	Annotations tags.DocTags `json:"annotations,omitempty"`

	Services  []*Service       `json:"services,omitempty"`
	Contracts []*Contract      `json:"contracts,omitempty"`
	Types     map[string]*Type `json:"types,omitempty"`

	ExcludeDirs []string `json:"excludeDirs,omitempty"`

	ProjectID string `json:"projectId,omitempty"` // Base58 UUIDv5 идентификатор проекта
	Marker    string `json:"marker,omitempty"`    // SHA256 маркер состояния проекта
}

type GitInfo struct {
	Commit    string `json:"commit"`
	Branch    string `json:"branch"`
	Tag       string `json:"tag,omitempty"`
	Dirty     bool   `json:"dirty"`
	User      string `json:"user,omitempty"`
	Email     string `json:"email,omitempty"`
	RemoteURL string `json:"remoteUrl,omitempty"`
}

type Service struct {
	Name        string   `json:"name"`
	MainPath    string   `json:"mainPath"`
	ContractIDs []string `json:"contractIds,omitempty"`
}

type Contract struct {
	Name            string                `json:"name"`
	PkgPath         string                `json:"pkgPath"`
	FilePath        string                `json:"filePath"`
	ID              string                `json:"id"`
	Docs            []string              `json:"docs,omitempty"`
	Annotations     tags.DocTags          `json:"annotations,omitempty"`
	Methods         []*Method             `json:"methods,omitempty"`
	Implementations []*ImplementationInfo `json:"implementations,omitempty"`
}

type Method struct {
	Name        string       `json:"name"`
	ContractID  string       `json:"contractID"`
	Args        []*Variable  `json:"args,omitempty"`
	Results     []*Variable  `json:"results,omitempty"`
	Docs        []string     `json:"docs,omitempty"`
	Annotations tags.DocTags `json:"annotations,omitempty"`
	Errors      []*ErrorInfo `json:"errors,omitempty"`
	Handler     *HandlerInfo `json:"handler,omitempty"`
}

// TypeRef описывает использование типа в конкретном месте: какой тип, указатели, слайс/массив/map.
// Рекурсивно используется в MapKey/MapValue для вложенных map. Не содержит имени, тегов и аннотаций.
type TypeRef struct {
	TypeID           string   `json:"typeID,omitempty"`
	NumberOfPointers int      `json:"numberOfPointers,omitempty"`
	IsSlice          bool     `json:"isSlice,omitempty"`
	ArrayLen         int      `json:"arrayLen,omitempty"`
	IsEllipsis       bool     `json:"isEllipsis,omitempty"`
	ElementPointers  int      `json:"elementPointers,omitempty"`
	MapKey           *TypeRef `json:"mapKey,omitempty"`
	MapValue         *TypeRef `json:"mapValue,omitempty"`
}

// Variable — аргумент/результат метода или элемент описания (map key/value и т.д.). Содержит TypeRef + имя и метаданные.
type Variable struct {
	TypeRef     `json:",inline"`
	Name        string       `json:"name"`
	Docs        []string     `json:"docs,omitempty"`
	Annotations tags.DocTags `json:"annotations,omitempty"`
}

type HandlerInfo struct {
	PkgPath string `json:"pkgPath"`
	Name    string `json:"name"`
}

type ImplementationInfo struct {
	PkgPath    string                           `json:"pkgPath"`
	StructName string                           `json:"structName"`
	MethodsMap map[string]*ImplementationMethod `json:"methods,omitempty"`
}

type ImplementationMethod struct {
	Name       string                `json:"name,omitempty"`
	FilePath   string                `json:"filePath"`
	ErrorTypes []*ErrorTypeReference `json:"errorTypes,omitempty"`
}

type ErrorInfo struct {
	PkgPath      string `json:"pkgPath"`
	TypeName     string `json:"typeName"`
	FullName     string `json:"fullName"`
	HTTPCode     int    `json:"httpCode,omitempty"`
	HTTPCodeText string `json:"httpCodeText,omitempty"`
	TypeID       string `json:"typeID,omitempty"`
}

type ErrorTypeReference struct {
	PkgPath  string `json:"pkgPath"`
	TypeName string `json:"typeName"`
	FullName string `json:"fullName"`
}

type Type struct {
	Kind TypeKind `json:"kind,omitempty"`

	TypeName      string `json:"typeName,omitempty"`
	ImportAlias   string `json:"importAlias,omitempty"`
	ImportPkgPath string `json:"importPkgPath,omitempty"`
	PkgName       string `json:"pkgName,omitempty"` // Реальное имя пакета из package декларации

	AliasOf string `json:"aliasOf,omitempty"`

	ArrayLen        int    `json:"arrayLen,omitempty"`
	IsSlice         bool   `json:"isSlice,omitempty"`
	IsEllipsis      bool   `json:"isEllipsis,omitempty"`
	ArrayOfID       string `json:"arrayOfID,omitempty"`
	ElementPointers int    `json:"elementPointers,omitempty"` // Для элементов массивов/слайсов и значений map

	MapKey   *TypeRef `json:"mapKey,omitempty"`
	MapValue *TypeRef `json:"mapValue,omitempty"`

	ChanDirection int    `json:"chanDirection,omitempty"`
	ChanOfID      string `json:"chanOfID,omitempty"`

	StructFields []*StructField `json:"structFields,omitempty"`

	InterfaceMethods   []*Function `json:"interfaceMethods,omitempty"`
	EmbeddedInterfaces []*Variable `json:"embeddedInterfaces,omitempty"`

	FunctionArgs    []*Variable `json:"functionArgs,omitempty"`
	FunctionResults []*Variable `json:"functionResults,omitempty"`

	UnderlyingTypeID string   `json:"underlyingTypeID,omitempty"`
	UnderlyingKind   TypeKind `json:"underlyingKind,omitempty"`

	ImplementsInterfaces []string `json:"implementsInterfaces,omitempty"`
}

// StructField — поле структуры. Содержит TypeRef + имя поля, теги и документацию.
type StructField struct {
	TypeRef `json:",inline"`
	Name    string              `json:"name"`
	Tags    map[string][]string `json:"tags,omitempty"`
	Docs    []string            `json:"docs,omitempty"`
}

type Function struct {
	Name    string      `json:"name"`
	Args    []*Variable `json:"args,omitempty"`
	Results []*Variable `json:"results,omitempty"`
	Docs    []string    `json:"docs,omitempty"`
}
