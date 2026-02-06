// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package parser

import (
	"fmt"
	"go/types"
	"log/slog"

	"tgp/core/i18n"
	"tgp/internal/model"
)

func detectInterfaces(typ types.Type, coreType *model.Type, project *model.Project, loader *AutonomousPackageLoader) {
	if coreType.ImportPkgPath == "" || coreType.TypeName == "" {
		return
	}

	// Если интерфейсы уже определены, не анализируем повторно
	if len(coreType.ImplementsInterfaces) > 0 {
		return
	}

	allInterfaces := getAllInterfacesFromLoader(loader)
	if len(allInterfaces) == 0 {
		return
	}

	implements := make([]string, 0)
	seenIDs := make(map[string]bool)

	for ifaceID, iface := range allInterfaces {
		if types.Implements(typ, iface) {
			if !seenIDs[ifaceID] {
				implements = append(implements, ifaceID)
				seenIDs[ifaceID] = true
			}
		}
	}

	// Это нужно для типов, которые реализуют интерфейсы только через указатель
	pointerType := types.NewPointer(typ)
	for ifaceID, iface := range allInterfaces {
		if types.Implements(pointerType, iface) {
			if !seenIDs[ifaceID] {
				implements = append(implements, ifaceID)
				seenIDs[ifaceID] = true
			}
		}
	}

	coreType.ImplementsInterfaces = implements

	// Обновляем тип в project.Types, если он уже там есть, чтобы сохранить интерфейсы
	if coreType.ImportPkgPath != "" && coreType.TypeName != "" {
		typeID := fmt.Sprintf("%s:%s", coreType.ImportPkgPath, coreType.TypeName)
		if _, exists := project.Types[typeID]; exists {
			project.Types[typeID] = coreType
		}
	}
}

func getAllInterfacesFromLoader(loader *AutonomousPackageLoader) (interfaces map[string]*types.Interface) {

	// Список стандартных библиотек с важными интерфейсами
	// Эти пакеты нужно загрузить явно, чтобы найти интерфейсы
	stdlibPackagesWithInterfaces := []string{
		"encoding",            // TextMarshaler, TextUnmarshaler, BinaryMarshaler, BinaryUnmarshaler
		"encoding/json",       // Marshaler, Unmarshaler, Token, isZeroer
		"encoding/xml",        // Marshaler, Unmarshaler, Token
		"fmt",                 // Stringer, GoStringer
		"io",                  // Reader, Writer, Closer, ReadWriter, etc.
		"context",             // Context
		"errors",              // Unwrap, Is, As (Go 1.13+)
		"sort",                // Interface
		"database/sql",        // Scanner, Valuer
		"database/sql/driver", // Value, Valuer, etc.
		"flag",                // Value
		"net",                 // Addr, Conn, Listener, etc.
		"reflect",             // Type, Value
		"strings",             // Builder
		"sync",                // Locker
		"time",                // (для проверки типов, которые реализуют интерфейсы)
		"cmp",                 // Ordered (Go 1.21+)
	}

	// Явно загружаем стандартные библиотеки с интерфейсами
	for _, pkgPath := range stdlibPackagesWithInterfaces {
		var ok bool
		if _, ok = loader.GetPackage(pkgPath); !ok {
			var err error
			if _, err = loader.LoadPackageLazy(pkgPath); err != nil {
				slog.Debug(i18n.Msg("Failed to load standard library package for interface detection"),
					slog.String("package", pkgPath),
					slog.Any("error", err))
			}
		}
	}

	interfaces = make(map[string]*types.Interface)
	seenInterfaces := make(map[string]bool)

	allPackages := loader.GetAllPackages()

	for _, pkgInfo := range allPackages {
		if pkgInfo.Types == nil {
			continue
		}

		scope := pkgInfo.Types.Scope()
		for _, name := range scope.Names() {
			obj := scope.Lookup(name)
			if obj == nil {
				continue
			}

			typeName, ok := obj.(*types.TypeName)
			if !ok {
				continue
			}

			typ := typeName.Type()
			iface, ok := typ.Underlying().(*types.Interface)
			if !ok {
				continue
			}

			interfacePkgPath := typeName.Pkg().Path()
			interfaceID := fmt.Sprintf("%s:%s", interfacePkgPath, typeName.Name())

			// Если ID уже есть, это дубликат - пропускаем
			if seenInterfaces[interfaceID] {
				continue
			}

			interfaces[interfaceID] = iface
			seenInterfaces[interfaceID] = true
		}
	}

	return
}
