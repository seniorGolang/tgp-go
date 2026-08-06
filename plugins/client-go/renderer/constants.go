// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

const (
	PackageOS             = "os"
	PackageIO             = "io"
	PackageFmt            = "fmt"
	PackageTLS            = "crypto/tls"
	PackageTime           = "time"
	PackageHTTP           = "net/http"
	PackageURL            = "net/url"
	PackagePath           = "path"
	PackageContext        = "context"
	PackageStrconv        = "strconv"
	PackageBytes          = "bytes"
	PackageStrings        = "strings"
	PackageStdJSON        = "encoding/json"
	PackageErrors         = "errors"
	PackageReflect        = "reflect"
	PackageSort           = "sort"
	PackageUUID           = "github.com/google/uuid"
	PackageOAuth2         = "golang.org/x/oauth2"
	PackageFiber          = "github.com/gofiber/fiber/v2"
	PackageSlog           = "log/slog"
	PackagePrometheus     = "github.com/prometheus/client_golang/prometheus"
	PackagePrometheusAuto = "github.com/prometheus/client_golang/prometheus/promauto"
	tagPackageJSON        = "packageJSON"
	TagMetrics            = "metrics"
	tagSummary            = "summary"
	tagDesc               = "desc"
)

const (
	PackageMime          = "mime"
	PackageMimeMultipart = "mime/multipart"
	PackageNetTextproto  = "net/textproto"
	PackageSync          = "sync"
	PackageXML           = "encoding/xml"
	PackageMsgpack       = "github.com/vmihailenco/msgpack/v5"
	PackageCBOR          = "github.com/fxamacker/cbor/v2"
	PackageYAML          = "gopkg.in/yaml.v3"
)

const (
	TypeIDIOReader     = "io:Reader"
	TypeIDIOReadCloser = "io:ReadCloser"
)

const (
	jsonTagCode            = "code"
	jsonTagDataOmitempty   = "data,omitempty"
	jsonTagErrorOmitempty  = "error,omitempty"
	jsonTagMessage         = "message"
	jsonTagResultOmitempty = "result,omitempty"
	keyJSON                = "json"
	labelDescription       = "Описание"
	labelNo                = "Нет"
	metricsLabelNames      = "service, method, success, errCode, client_id"
	nameResult             = "result"
	nameTime               = "Time"
	pkgClient              = "client"
	pkgDTO                 = "dto"
	pkgJSONRPC             = "jsonrpc"
	tplContractName        = "ContractName"
	tplMethodCall          = "MethodCall"
	tplPkgName             = "PkgName"
	tplPkgPath             = "PkgPath"
	tplServiceVar          = "ServiceVar"
	typeAny                = "any"
	typeError              = "error"
	typeString             = "string"
)

const (
	labelField   = "Поле"
	labelName    = "Имя"
	labelType    = "Тип"
	labelTypeRef = "Ссылка на тип"
)
