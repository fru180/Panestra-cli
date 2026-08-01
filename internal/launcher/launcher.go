package launcher

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/fru180/Panestra-cli/internal/config"
	"github.com/fru180/Panestra-cli/internal/prompt"
	ptmux "github.com/fru180/Panestra-cli/internal/tmux"
	"golang.org/x/term"
)

const dedicatedConfig = `set -g status off
set -g pane-border-status top
set -g pane-border-format '#{?#{@panestra_cli_prompt}, #{@panestra_cli_prefix}#{@panestra_cli_prompt} ,#{?#{@panestra_cli_show_waiting}, Panestra CLI — Waiting for prompt ,}}'
set -g mouse on
bind-key -n WheelUpPane copy-mode -e \; send-keys -X -N 5 scroll-up
set -g allow-rename off
set -g set-titles off
set -g exit-empty on
`

type transcriptResult struct {
	data []byte
	err  error
}

func Launch(agent string, args []string) int {
	depth, _ := strconv.Atoi(os.Getenv("PANESTRA_CLI_LAUNCH_DEPTH"))
	if depth >= 2 {
		fmt.Fprintln(os.Stderr, "panestra: recursive launch detected")
		return 126
	}
	real, err := Resolve(agent)
	if err != nil {
		fmt.Fprintln(os.Stderr, "panestra:", err)
		return 127
	}
	cfg := config.Load()
	interactive := Interactive(agent, args, term.IsTerminal(int(os.Stdin.Fd())), term.IsTerminal(int(os.Stdout.Fd())))
	if !cfg.Enabled || !cfg.AutoTmux || !interactive {
		return run(real, args, depth+1)
	}
	pane := os.Getenv("TMUX_PANE")
	if pane == "" && os.Getenv("TMUX") != "" {
		pane, _ = ptmux.New().CurrentPane()
	}
	if pane != "" {
		t := ptmux.New()
		if err := t.Acquire(pane, prompt.DisplayPrefix(cfg.Prefix), cfg.ShowWaitingMessage); err != nil {
			_ = t.Release(pane)
			fmt.Fprintln(os.Stderr, "panestra: tmux display unavailable; continuing normally:", err)
			return run(real, args, depth+1)
		}
		defer func() {
			if err := t.Release(pane); err != nil {
				fmt.Fprintln(os.Stderr, "panestra: could not restore tmux settings:", err)
			}
		}()
		return run(real, args, depth+1)
	}
	if os.Getenv("TMUX") != "" {
		fmt.Fprintln(os.Stderr, "panestra: current tmux pane could not be identified; continuing normally")
		return run(real, args, depth+1)
	}
	if _, err := exec.LookPath("tmux"); err != nil {
		fmt.Fprintln(os.Stderr, "panestra: tmux was not found; continuing normally")
		return run(real, args, depth+1)
	}
	return runDedicated(real, args)
}

func run(path string, args []string, depth int) int {
	cmd := exec.Command(path, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Env = withEnv(os.Environ(), "PANESTRA_CLI_LAUNCH_DEPTH", strconv.Itoa(depth))
	if err := cmd.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "panestra:", err)
		return 126
	}
	sigs := make(chan os.Signal, 4)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case s := <-sigs:
				_ = cmd.Process.Signal(s)
			case <-done:
				return
			}
		}
	}()
	err := cmd.Wait()
	close(done)
	signal.Stop(sigs)
	if err == nil {
		return 0
	}
	if ee, ok := err.(*exec.ExitError); ok {
		return ee.ExitCode()
	}
	return 126
}

func runDedicated(real string, args []string) int {
	self, err := os.Executable()
	if err != nil {
		return run(real, args, 1)
	}
	tmp, err := os.MkdirTemp("", "panestra-cli-")
	if err != nil {
		return run(real, args, 1)
	}
	defer os.RemoveAll(tmp)
	conf := filepath.Join(tmp, "tmux.conf")
	status := filepath.Join(tmp, "status")
	transcript := filepath.Join(tmp, "history.sock")
	if err := os.WriteFile(conf, []byte(dedicatedConfig), 0600); err != nil {
		return run(real, args, 1)
	}
	listener, transcriptCh, err := listenTranscript(transcript)
	if err != nil {
		transcript = ""
	} else {
		defer listener.Close()
	}
	payload, _ := json.Marshal(append([]string{real}, args...))
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	name := fmt.Sprintf("panestra-cli-%d-%d", os.Getpid(), time.Now().UnixNano())
	command := shellQuote(self) + " _exec-status " + shellQuote(status) + " " + shellQuote(encoded) + " " + shellQuote(transcript)
	cmd := exec.Command("tmux", "-L", name, "-f", conf, "new-session", "-s", name, command)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Env = withEnv(os.Environ(), "PANESTRA_CLI_LAUNCH_DEPTH", "1")
	if err := cmd.Start(); err != nil {
		if listener != nil {
			_ = listener.Close()
		}
		fmt.Fprintln(os.Stderr, "panestra: dedicated tmux failed; continuing normally:", err)
		return run(real, args, 1)
	}
	sigs := make(chan os.Signal, 2)
	done := make(chan struct{})
	interrupted := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		select {
		case sig := <-sigs:
			interrupted <- sig
			_ = exec.Command("tmux", "-L", name, "kill-server").Run()
		case <-done:
		}
	}()
	waitErr := cmd.Wait()
	close(done)
	signal.Stop(sigs)
	b, err := os.ReadFile(status)
	if err == nil {
		if listener != nil {
			select {
			case result := <-transcriptCh:
				if result.err == nil {
					writeTranscript(result.data)
				}
			case <-time.After(time.Second):
			}
		}
		code, _ := strconv.Atoi(string(b))
		return code
	}
	select {
	case sig := <-interrupted:
		if s, ok := sig.(syscall.Signal); ok {
			return 128 + int(s)
		}
		return 130
	default:
	}
	if waitErr != nil {
		fmt.Fprintln(os.Stderr, "panestra: dedicated tmux exited unexpectedly:", waitErr)
		return 126
	}
	return 0
}

func ExecStatus(status, encoded, transcript string) int {
	b, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return 126
	}
	var argv []string
	if json.Unmarshal(b, &argv) != nil || len(argv) == 0 {
		return 126
	}
	pane := os.Getenv("TMUX_PANE")
	if pane == "" && os.Getenv("TMUX") != "" {
		pane, _ = ptmux.New().CurrentPane()
		if pane != "" {
			_ = os.Setenv("TMUX_PANE", pane)
		}
	}
	if pane != "" {
		cfg := config.Load()
		client := ptmux.New()
		_ = client.SetPrefix(pane, prompt.DisplayPrefix(cfg.Prefix))
		_ = client.SetWaiting(pane, cfg.ShowWaitingMessage)
	}
	code := run(argv[0], argv[1:], 1)
	_ = os.WriteFile(status, []byte(strconv.Itoa(code)), 0600)
	if transcript != "" {
		var history []byte
		if pane != "" {
			if captured, captureErr := ptmux.New().CaptureHistory(pane); captureErr == nil {
				history = captured
			}
		}
		_ = sendTranscript(transcript, history)
	}
	return code
}

func listenTranscript(path string) (net.Listener, <-chan transcriptResult, error) {
	listener, err := net.Listen("unix", path)
	if err != nil {
		return nil, nil, err
	}
	if err := os.Chmod(path, 0600); err != nil {
		_ = listener.Close()
		return nil, nil, err
	}
	result := make(chan transcriptResult, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			result <- transcriptResult{err: err}
			return
		}
		defer conn.Close()
		data, err := io.ReadAll(conn)
		result <- transcriptResult{data: data, err: err}
	}()
	return listener, result, nil
}

func sendTranscript(path string, data []byte) error {
	conn, err := net.DialTimeout("unix", path, time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()
	_ = conn.SetWriteDeadline(time.Now().Add(time.Second))
	_, err = io.Copy(conn, bytes.NewReader(data))
	return err
}

func writeTranscript(data []byte) {
	data = transcriptForTerminal(data)
	if len(data) != 0 {
		_, _ = os.Stdout.Write(data)
	}
}

func transcriptForTerminal(data []byte) []byte {
	data = bytes.TrimRight(data, "\r\n")
	if len(data) == 0 {
		return nil
	}
	out := make([]byte, 0, len(data)+9)
	out = append(out, "\x1b[0m"...)
	out = append(out, data...)
	out = append(out, '\n')
	out = append(out, "\x1b[0m"...)
	return out
}

func withEnv(env []string, key, value string) []string {
	prefix := key + "="
	out := make([]string, 0, len(env)+1)
	for _, item := range env {
		if len(item) < len(prefix) || item[:len(prefix)] != prefix {
			out = append(out, item)
		}
	}
	return append(out, prefix+value)
}

func shellQuote(s string) string {
	result := "'"
	for _, r := range s {
		if r == '\'' {
			result += "'\\''"
		} else {
			result += string(r)
		}
	}
	return result + "'"
}
