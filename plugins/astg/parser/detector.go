// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package parser

import (
	"fmt"
	"go/types"
	"log/slog"

	"tgp/core/i18n"
	"tgp/internal/model"
)

// detectInterfaces определяет все интерфейсы, которые реализует тип.
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

	// Проверяем сам тип
	for ifaceID, iface := range allInterfaces {
		if types.Implements(typ, iface) {
			if !seenIDs[ifaceID] {
				implements = append(implements, ifaceID)
				seenIDs[ifaceID] = true
			}
		}
	}

	// Проверяем указатель на тип (для интерфейсов, которые требуют pointer receiver)
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

// getAllInterfacesFromLoader собирает интерфейсы из всех загруженных пакетов в loader.
// Явно загружает стандартные библиотеки с известными интерфейсами, если они еще не загружены.
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
		// Проверяем, не загружен ли уже пакет
		var ok bool
		if _, ok = loader.GetPackage(pkgPath); !ok {
			// Загружаем пакет (игнорируем ошибки, так как некоторые пакеты могут быть недоступны)
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

	// Получаем все загруженные пакеты (включая только что загруженные)
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

			// Формируем уникальный ID для интерфейса
			interfacePkgPath := typeName.Pkg().Path()
			interfaceID := fmt.Sprintf("%s:%s", interfacePkgPath, typeName.Name())

			// Проверяем, не добавляли ли мы уже этот интерфейс по ID
			// Если ID уже есть, это дубликат - пропускаем
			if seenInterfaces[interfaceID] {
				continue
			}

			// Добавляем интерфейс
			interfaces[interfaceID] = iface
			seenInterfaces[interfaceID] = true
		}
	}

	return
}
