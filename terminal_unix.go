//go:build darwin || linux

package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/creack/pty"
	"golang.org/x/term"
)

func terminalSize() (int, int) {
	w, h, e := term.GetSize(1)
	if e != nil || w < 1 || h < 1 {
		return 80, 24
	}
	return w, h
}
func write(s string) error { _, err := io.WriteString(os.Stdout, s); return err }
func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var e *exec.ExitError
	if errors.As(err, &e) {
		if s, ok := e.Sys().(syscall.WaitStatus); ok && s.Signaled() {
			return 128 + int(s.Signal())
		}
		return e.ExitCode()
	}
	return 1
}
func readChunks(file *os.File, done <-chan struct{}) <-chan []byte {
	ch := make(chan []byte, 8)
	go func() {
		defer close(ch)
		buf := make([]byte, 32768)
		for {
			n, e := file.Read(buf)
			if n > 0 {
				data := append([]byte(nil), buf[:n]...)
				select {
				case ch <- data:
				case <-done:
					return
				}
			}
			if e != nil {
				return
			}
		}
	}()
	return ch
}
func runTerminal(path string, args []string, intro, preview bool) int {
	if !term.IsTerminal(0) || !term.IsTerminal(1) {
		if preview {
			fmt.Fprintln(os.Stderr, "预览需要交互终端。")
			return 1
		}
		return direct(path, args)
	}
	state, err := term.MakeRaw(0)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	alt := false
	defer func() {
		_ = term.Restore(0, state)
		if alt {
			_ = write("\x1b[0m\x1b[<u\x1b[?2004l\x1b[?1004l\x1b[?1000l\x1b[?1002l\x1b[?1003l\x1b[?1006l\x1b[?25h\x1b[?1049l")
		}
	}()
	signals := make(chan os.Signal, 16)
	signal.Notify(signals, syscall.SIGWINCH, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTSTP)
	defer signal.Stop(signals)
	done := make(chan struct{})
	defer close(done)
	input := readChunks(os.Stdin, done)
	var queued []byte
	if intro {
		alt = true
		if write("\x1b[?1049h\x1b[?25l\x1b[2J") != nil {
			return 129
		}
		start := time.Now()
		duration := 2.95
		if os.Getenv("PANGU_MOTION") == "reduce" {
			duration = 1
		}
		tick := time.NewTicker(time.Second / 30)
		for time.Since(start).Seconds() < duration {
			select {
			case sig := <-signals:
				if sig != syscall.SIGWINCH {
					tick.Stop()
					return 128 + int(sig.(syscall.Signal))
				}
			case data, ok := <-input:
				if !ok {
					tick.Stop()
					return 129
				}
				if bytes.Contains(data, []byte{3}) || bytes.Contains(data, []byte("\x1b[99;5u")) {
					tick.Stop()
					return 130
				}
				if len(queued)+len(data) <= 65536 {
					queued = append(queued, data...)
				}
			case <-tick.C:
				w, h := terminalSize()
				l := choose(w, h-11)
				top := 5 + max(0, (h-11-l.height)/2)
				t := time.Since(start).Seconds()
				if duration == 1 {
					t = 3
				}
				if write("\x1b[?2026h\x1b[2J"+render(l.frame(top, t), w, h)+"\x1b[?2026l") != nil {
					tick.Stop()
					return 129
				}
			}
		}
		tick.Stop()
		if preview {
			return 0
		}
	}
	w, h := terminalSize()
	cmd := exec.Command(path, args...)
	master, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: uint16(h), Cols: uint16(w)})
	if err != nil {
		_ = term.Restore(0, state)
		if alt {
			_ = write("\x1b[0m\x1b[?25h\x1b[?1049l")
			alt = false
		}
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer master.Close()
	app := "codex"
	if filepath.Base(path) == "claude" {
		app = "claude"
	}
	ov := newOverlay(w, h, intro)
	ov.app = app
	out := readChunks(master, done)
	wait := make(chan error, 1)
	go func() { wait <- cmd.Wait() }()
	kill := func(sig syscall.Signal) { _ = syscall.Kill(-cmd.Process.Pid, sig) }
	completed := false
	defer func() {
		if !completed {
			kill(syscall.SIGHUP)
			select {
			case <-wait:
			case <-time.After(time.Second):
				kill(syscall.SIGKILL)
			}
		}
	}()
	if len(queued) > 0 {
		if _, err = master.Write(queued); err != nil {
			return 129
		}
	}
	var drain <-chan time.Time
	status := 0
	for {
		select {
		case data, ok := <-out:
			if !ok {
				out = nil
				if completed {
					alt = ov.alt
					return status
				}
				continue
			}
			if err = write(ov.feed(data)); err != nil {
				return 129
			}
			alt = ov.alt
		case data, ok := <-input:
			if !ok {
				input = nil
				kill(syscall.SIGHUP)
				continue
			}
			if bytes.Equal(data, []byte{26}) {
				kill(syscall.SIGSTOP)
				_ = term.Restore(0, state)
				_ = syscall.Kill(os.Getpid(), syscall.SIGSTOP)
				_, _ = term.MakeRaw(0)
				kill(syscall.SIGCONT)
				_ = pty.InheritSize(os.Stdout, master)
				w, h = terminalSize()
				ov.resize(w, h)
				continue
			}
			ov.noteInput(data)
			if _, err = master.Write(data); err != nil {
				return 129
			}
		case sig := <-signals:
			switch sig {
			case syscall.SIGWINCH:
				_ = pty.InheritSize(os.Stdout, master)
				w, h = terminalSize()
				ov.resize(w, h)
			case syscall.SIGTSTP:
				kill(syscall.SIGSTOP)
				_ = term.Restore(0, state)
				_ = syscall.Kill(os.Getpid(), syscall.SIGSTOP)
				_, _ = term.MakeRaw(0)
				kill(syscall.SIGCONT)
				_ = pty.InheritSize(os.Stdout, master)
			default:
				kill(sig.(syscall.Signal))
			}
		case err := <-wait:
			completed = true
			status = exitCode(err)
			wait = nil
			drain = time.After(200 * time.Millisecond)
			if out == nil {
				alt = ov.alt
				return status
			}
		case <-drain:
			if len(ov.carry) > 0 {
				_ = write(string(ov.carry))
			}
			alt = ov.alt
			return status
		}
	}
}
