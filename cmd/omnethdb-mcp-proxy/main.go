package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func main() {
	logPath, target, targetArgs, err := parseProxyArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if target == "" {
		fmt.Fprintln(os.Stderr, "error: --target is required")
		os.Exit(1)
	}
	if logPath == "" {
		logPath = filepath.Join(os.TempDir(), "omnethdb-mcp-proxy.log")
	}

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	defer logFile.Close()
	logger := &proxyLogger{file: logFile}

	cmd := exec.Command(target, targetArgs...)
	childIn, err := cmd.StdinPipe()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	childOut, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	childErr, err := cmd.StderrPipe()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	if err := cmd.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	logger.write("proxy_start", fmt.Sprintf("target=%s args=%q pid=%d", target, targetArgs, cmd.Process.Pid))

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		_, _ = io.Copy(childIn, io.TeeReader(os.Stdin, logger.stream("stdin")))
		_ = childIn.Close()
	}()

	go func() {
		defer wg.Done()
		_, _ = io.Copy(io.MultiWriter(os.Stdout, logger.stream("stdout")), childOut)
	}()

	go func() {
		defer wg.Done()
		_, _ = io.Copy(io.MultiWriter(os.Stderr, logger.stream("stderr")), childErr)
	}()

	err = cmd.Wait()
	wg.Wait()
	if err != nil {
		logger.write("proxy_exit", fmt.Sprintf("error=%v", err))
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	logger.write("proxy_exit", "ok")
}

func parseProxyArgs(args []string) (string, string, []string, error) {
	separator := len(args)
	for i, arg := range args {
		if arg == "--" {
			separator = i
			break
		}
	}

	known := args[:separator]
	trailing := []string{}
	if separator < len(args) {
		trailing = args[separator+1:]
	}

	fs := flag.NewFlagSet("omnethdb-mcp-proxy", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	logPath := fs.String("log", "", "path to proxy log file")
	target := fs.String("target", "", "path to target MCP binary")
	if err := fs.Parse(known); err != nil {
		return "", "", nil, err
	}

	targetArgs := append([]string{}, fs.Args()...)
	targetArgs = append(targetArgs, trailing...)
	return strings.TrimSpace(*logPath), strings.TrimSpace(*target), targetArgs, nil
}

type proxyLogger struct {
	file *os.File
	mu   sync.Mutex
}

type prefixedLogWriter struct {
	logger *proxyLogger
	stream string
}

func (l *proxyLogger) stream(stream string) io.Writer {
	return &prefixedLogWriter{logger: l, stream: stream}
}

func (l *proxyLogger) write(stream string, message string) {
	_, _ = l.stream(stream).Write([]byte(message))
}

func (w *prefixedLogWriter) Write(p []byte) (int, error) {
	w.logger.mu.Lock()
	defer w.logger.mu.Unlock()
	prefix := []byte("[" + time.Now().UTC().Format(time.RFC3339Nano) + "] " + w.stream + ": ")
	if _, err := w.logger.file.Write(prefix); err != nil {
		return 0, err
	}
	if _, err := w.logger.file.Write(p); err != nil {
		return 0, err
	}
	if len(p) == 0 || p[len(p)-1] != '\n' {
		if _, err := w.logger.file.Write([]byte("\n")); err != nil {
			return 0, err
		}
	}
	return len(p), nil
}
