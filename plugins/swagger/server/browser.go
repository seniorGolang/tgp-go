// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package server

import (
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"strings"

	"tgp/core/exec"
	"tgp/core/i18n"
)

func DetectHostOS() (hostOS string) {

	cmd := exec.Command(CommandUname, "-s")
	if err := cmd.Start(); err == nil {
		var stdoutPipe io.ReadCloser
		if stdoutPipe, err = cmd.StdoutPipe(); err == nil {
			var stdoutBytes []byte
			var readErr error
			if stdoutBytes, readErr = io.ReadAll(stdoutPipe); readErr == nil {
				stdoutPipe.Close()

				var waitErr error
				if waitErr = cmd.Wait(); waitErr == nil {
					unameOutput := strings.TrimSpace(string(stdoutBytes))

					unameLower := strings.ToLower(unameOutput)
					if strings.Contains(unameLower, OSDarwin) {
						return OSDarwin
					}
					if strings.Contains(unameLower, OSLinux) {
						return OSLinux
					}
					return unameOutput
				}
			}
		}
	}

	var ostype string
	if ostype = os.Getenv(EnvVarOSType); ostype != "" {
		ostypeLower := strings.ToLower(ostype)
		if strings.Contains(ostypeLower, OSDarwin) {
			return OSDarwin
		}
		if strings.Contains(ostypeLower, OSLinux) {
			return OSLinux
		}
	}

	return ""
}

func OpenBrowser(browserURL string) (err error) {

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%s: %v", i18n.Msg("panic in openBrowser"), r)
			slog.Error(i18n.Msg("panic recovered in openBrowser"),
				slog.String("url", browserURL),
				slog.Any("panic", r),
			)
		}
	}()

	hostOS := DetectHostOS()

	var cmd *exec.Cmd
	switch hostOS {
	case OSDarwin:
		cmd = exec.Command(CommandOpen, browserURL)
	case OSLinux:
		cmd = exec.Command(CommandXdgOpen, browserURL)
	default:
		commands := []struct {
			name string
			args []string
		}{
			{CommandOpen, []string{browserURL}},
			{CommandXdgOpen, []string{browserURL}},
			{CommandCmd, []string{"/c", "start", browserURL}},
		}

		var lastErr error
		for _, cmdInfo := range commands {
			tryCmd := exec.Command(cmdInfo.name, cmdInfo.args...)

			if startErr := tryCmd.Start(); startErr != nil {
				lastErr = startErr
				continue
			}

			if waitErr := tryCmd.Wait(); waitErr != nil {
				lastErr = waitErr
				continue
			}

			return nil
		}

		slog.Error(i18n.Msg("failed to open browser"),
			slog.String("url", browserURL),
			slog.Any("error", lastErr),
		)
		if lastErr != nil {
			return fmt.Errorf("%s: %w", i18n.Msg("failed to open browser with any command"), lastErr)
		}
		return fmt.Errorf("%s", i18n.Msg("failed to open browser: no commands available"))
	}

	if err = cmd.Start(); err != nil {
		slog.Error(i18n.Msg("failed to open browser"),
			slog.String("url", browserURL),
			slog.Any("error", err),
		)
		return fmt.Errorf("%s: %w", i18n.Msg("failed to start browser command"), err)
	}

	if err = cmd.Wait(); err != nil {
		slog.Error(i18n.Msg("failed to open browser"),
			slog.String("url", browserURL),
			slog.Any("error", err),
		)
		return fmt.Errorf("%s: %w", i18n.Msg("browser command failed"), err)
	}

	return
}

func AddressToURL(addr string) (browserURL string) {

	// Если адрес уже содержит протокол, возвращаем как есть
	if strings.HasPrefix(addr, "http://") || strings.HasPrefix(addr, "https://") {
		return addr
	}

	// Если адрес начинается с ":", добавляем localhost
	if strings.HasPrefix(addr, ":") {
		addr = "localhost" + addr
	} else if !strings.Contains(addr, ":") {
		// Если адрес не содержит ":" и не начинается с ":", добавляем localhost: перед адресом
		addr = "localhost:" + addr
	}

	// Парсим адрес для проверки корректности
	parsedURL, err := url.Parse("http://" + addr)
	if err != nil {
		// Если не удалось распарсить, возвращаем простой вариант
		return "http://" + addr
	}

	browserURL = "http://" + parsedURL.Host
	if parsedURL.Path != "" {
		browserURL += parsedURL.Path
	}

	return browserURL
}
