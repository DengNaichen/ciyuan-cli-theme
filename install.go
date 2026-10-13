package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

const blockStart = "# >>> ciyuan-cli-theme >>>"
const blockEnd = "# <<< ciyuan-cli-theme <<<"

func quote(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'" }
func removeBlock(s string) (string, error) {
	a := strings.Index(s, blockStart)
	if a < 0 {
		return s, nil
	}
	end := strings.Index(s[a:], blockEnd)
	if end < 0 {
		return "", fmt.Errorf("主题配置段未闭合，未改动 shell 文件")
	}
	end += a + len(blockEnd)
	if end < len(s) && s[end] == '\n' {
		end++
	}
	return s[:a] + s[end:], nil
}
func atomicWrite(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".ciyuan-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if err = f.Chmod(mode); err == nil {
		_, err = f.Write(data)
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(name, path)
}
func install(uninstall bool, args []string) error {
	selected := "codex"
	shell := filepath.Base(os.Getenv("SHELL"))
	if shell == "." || shell == "" {
		if runtime.GOOS == "darwin" {
			shell = "zsh"
		} else {
			shell = "bash"
		}
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--apps":
			i++
			if i >= len(args) {
				return fmt.Errorf("--apps 缺少值")
			}
			selected = args[i]
		case "--shell":
			i++
			if i >= len(args) {
				return fmt.Errorf("--shell 缺少值")
			}
			shell = args[i]
		default:
			return fmt.Errorf("未知安装参数 %s", args[i])
		}
	}
	if shell != "zsh" && shell != "bash" {
		return fmt.Errorf("安装器支持 zsh/bash；其他 shell 可直接运行 ciyuan-theme run codex")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := home
	if shell == "zsh" && os.Getenv("ZDOTDIR") != "" {
		dir = os.Getenv("ZDOTDIR")
	}
	rc := filepath.Join(dir, "."+shell+"rc")
	// Follow an existing symlink rather than replacing a dotfiles-manager symlink.
	if resolved, e := filepath.EvalSymlinks(rc); e == nil {
		rc = resolved
	}
	old, err := os.ReadFile(rc)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	clean, err := removeBlock(string(old))
	if err != nil {
		return err
	}
	apps := map[string]bool{}
	if !uninstall {
		for _, app := range strings.Split(selected, ",") {
			if app != "codex" && app != "claude" {
				return fmt.Errorf("不支持的程序 %s", app)
			}
			apps[app] = true
		}
		// Adding the other host must not remove the already-enabled host.
		if a := strings.Index(string(old), blockStart); a >= 0 {
			block := string(old)[a:]
			for _, app := range []string{"codex", "claude"} {
				if strings.Contains(block, app+"()") || strings.Contains(block, app+" ()") {
					apps[app] = true
				}
			}
		}
		for app := range apps {
			pattern := regexp.MustCompile(`(?m)^\s*alias\s+` + app + `=`)
			if pattern.MatchString(clean) {
				return fmt.Errorf("%s 存在自定义 alias；请保留其参数并手动接入启动器，未修改配置", app)
			}
		}
	}
	stamp := time.Now().Format("20060102-150405.000000000")
	backup := filepath.Join(home, ".config", "ciyuan-theme", "backups", stamp)
	binary := filepath.Join(home, ".local", "bin", "ciyuan-theme")
	if !uninstall {
		exe, e := os.Executable()
		if e != nil {
			return e
		}
		payload, e := os.ReadFile(exe)
		if e != nil {
			return e
		}
		if previous, e := os.ReadFile(binary); e == nil {
			if e = atomicWrite(filepath.Join(backup, "ciyuan-theme"), previous, 0755); e != nil {
				return e
			}
		} else if !os.IsNotExist(e) {
			return e
		}
		if e = atomicWrite(binary, payload, 0755); e != nil {
			return e
		}
		clean = strings.TrimRight(clean, "\n") + "\n\n" + blockStart + "\n# 词元无限 Go 单文件主题；PANGU=off 可单次跳过\n"
		for _, app := range []string{"codex", "claude"} {
			if apps[app] {
				clean += "unalias " + app + " 2>/dev/null\n" + app + "() { " + quote(binary) + " run " + app + " \"$@\"; }\n"
			}
		}
		clean += blockEnd + "\n"
	}
	if clean != string(old) {
		mode := os.FileMode(0644)
		if st, e := os.Stat(rc); e == nil {
			mode = st.Mode().Perm()
			if e = atomicWrite(filepath.Join(backup, filepath.Base(rc)), old, 0600); e != nil {
				return e
			}
		}
		if err = atomicWrite(rc, []byte(clean), mode); err != nil {
			return err
		}
	}
	if uninstall {
		fmt.Println("已移除主题启动函数。重新打开终端生效；二进制和备份仍保留。")
	} else {
		fmt.Printf("已安装 Go 单文件主题：%s\n已接入：%s\n重新打开终端，照常运行 codex 或 claude。\n", binary, rc)
	}
	return nil
}
