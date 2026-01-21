// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package server

const (
	// EnvVarOSType - переменная окружения для определения ОС.
	EnvVarOSType = "OSTYPE"

	// CommandOpen - команда для открытия браузера на macOS.
	CommandOpen = "open"

	// CommandXdgOpen - команда для открытия браузера на Linux.
	CommandXdgOpen = "xdg-open"

	// CommandCmd - команда для открытия браузера на Windows.
	CommandCmd = "cmd"

	// CommandUname - команда для определения ОС.
	CommandUname = "uname"

	// OSDarwin - название ОС Darwin (macOS).
	OSDarwin = "darwin"

	// OSLinux - название ОС Linux.
	OSLinux = "linux"
)
