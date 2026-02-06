package generator

import (
	"io"
	"os"

	"tgp/core/exec"
)

func runGoTidyCMD(outDir string) (err error) {

	return runCmdWithStdio(exec.Command("go", "mod", "tidy"), outDir)
}

func runGoGenerateCMD(outDir string) (err error) {

	return runCmdWithStdio(exec.Command("go", "generate", "./..."), outDir)
}

// runCmdWithStdio запускает команду и перенаправляет её stdout/stderr в os.Stdout/os.Stderr.
// Для WASM exec: сначала Start(), затем StdoutPipe/StderrPipe (pipe'ы берутся из ответа запущенной команды).
func runCmdWithStdio(cmd *exec.Cmd, outDir string) (err error) {

	cmd = cmd.Dir(outDir)

	if err = cmd.Start(); err != nil {
		return
	}

	var stdoutPipe io.ReadCloser
	if stdoutPipe, err = cmd.StdoutPipe(); err != nil {
		return
	}

	var stderrPipe io.ReadCloser
	if stderrPipe, err = cmd.StderrPipe(); err != nil {
		return
	}

	go func() { _, _ = io.Copy(os.Stdout, stdoutPipe) }()
	go func() { _, _ = io.Copy(os.Stderr, stderrPipe) }()

	return cmd.Wait()
}
