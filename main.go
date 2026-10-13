package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"golang.org/x/term"
)

const version = "0.2.0"

func main() { os.Exit(entry(os.Args[1:])) }
func entry(args []string) int {
	if len(args) == 0 {
		usage()
		return 0
	}
	switch args[0] {
	case "--version", "version":
		fmt.Println("ciyuan-theme", version, "(Go; embedded artwork)")
		return 0
	case "--help", "help":
		usage()
		return 0
	case "install", "uninstall":
		if err := install(args[0] == "uninstall", args[1:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	case "preview":
		if err := loadArt(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return runTerminal("", nil, true, true)
	case "run":
		args = args[1:]
	}
	app := "codex"
	if len(args) > 0 && (args[0] == "codex" || args[0] == "claude") {
		app = args[0]
		args = args[1:]
	}
	path, err := exec.LookPath(app)
	if err != nil {
		fmt.Fprintf(os.Stderr, "找不到 %s，请先安装原程序。\n", app)
		return 127
	}
	mode := classify(app, args)
	off := strings.ToLower(os.Getenv("PANGU"))
	if off == "off" || off == "0" || off == "no" || off == "false" || !term.IsTerminal(0) || !term.IsTerminal(1) || os.Getenv("TERM") == "dumb" {
		mode = "direct"
	}
	if mode == "direct" {
		return direct(path, args)
	}
	if app == "claude" {
		brandLabel = "Claude"
	}
	if err := loadArt(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if app == "codex" {
		args = append([]string{"-c", "tui.effects.welcome=false", "-c", "tui.show_tooltips=false"}, args...)
	}
	return runTerminal(path, args, mode == "intro", false)
}
func usage() {
	fmt.Print(`词元无限 · TOKENS INFINITY

ciyuan-theme run codex [args...]   启动 Codex
ciyuan-theme run claude [args...]  启动 Claude Code
ciyuan-theme preview              仅播放动画
ciyuan-theme install --apps codex 安装 shell 启动函数
ciyuan-theme uninstall            移除启动函数（保留二进制）
ciyuan-theme --version            版本

PANGU=off 跳过主题；PANGU_MOTION=reduce 静态开屏；PANGU_THEME=light 浅色。
`)
}
func direct(path string, args []string) int {
	if err := syscall.Exec(path, append([]string{path}, args...), os.Environ()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 126
	}
	return 0
}
func classify(app string, args []string) string {
	directFlags := map[string]bool{"--help": true, "-h": true, "--version": true, "-V": true, "--no-alt-screen": true}
	values := map[string]bool{}
	sub := map[string]bool{}
	resume := map[string]bool{}
	if app == "codex" {
		for _, s := range strings.Fields("-c --config --enable --disable --remote --remote-auth-token-env -i --image -m --model --local-provider -p --profile -s --sandbox -C --cd --add-dir -a --ask-for-approval") {
			values[s] = true
		}
		for _, s := range strings.Fields("agents exec e review login logout mcp plugin app-server remote-control app completion update doctor sandbox debug apply a queue archive delete migrate-rollouts unarchive cloud exec-server features help mcp-server proto responses-api-proxy stdio-to-uds generate-ts") {
			sub[s] = true
		}
		resume["resume"], resume["fork"] = true, true
	} else {
		for _, s := range strings.Fields("-p --print -v --safe-mode --bare --bg --background --desktop --cloud --ax-screen-reader --output-format --input-format") {
			directFlags[s] = true
		}
		for _, s := range strings.Fields("--model --settings --setting-sources --add-dir --plugin-dir --mcp-config --append-system-prompt --system-prompt --permission-mode --allowedTools --allowed-tools --disallowedTools --disallowed-tools --agents --agent --session-id --fallback-model --debug-file --betas --name -n --effort --tools --max-turns --worktree -w") {
			values[s] = true
		}
		for _, s := range strings.Fields("doctor update upgrade install auth mcp plugin plugins project agents attach logs stop kill rm respawn gateway import setup-token config migrate-installer api-key help") {
			sub[s] = true
		}
	}
	mode := "intro"
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			break
		}
		if strings.HasPrefix(a, "-") {
			flag := strings.SplitN(a, "=", 2)[0]
			if directFlags[flag] {
				return "direct"
			}
			if app == "claude" && (flag == "-c" || flag == "--continue" || flag == "-r" || flag == "--resume" || flag == "--from-pr" || flag == "--teleport") {
				mode = "overlay"
			}
			if values[flag] && !strings.Contains(a, "=") {
				i++
			}
			continue
		}
		if sub[a] {
			return "direct"
		}
		if resume[a] {
			return "overlay"
		}
		break
	}
	return mode
}
