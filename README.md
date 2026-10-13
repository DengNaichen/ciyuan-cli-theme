# 词元无限 · TOKENS INFINITY

Go 单文件 Codex / Claude Code 终端主题。二进制内嵌 Logo、蓝青渐变和英文标题字形：几何面从代码碎片中聚合，标题逐字显现，扫光后进入首页。

**运行无需 Python、Pillow、Node 或 Go 工具链，不读取外部素材。** 仍需安装原版 Codex 或 Claude Code。支持 macOS / Linux，ARM64 / x86_64。二进制使用 `CGO_ENABLED=0` 构建；Go 第三方库一并编入，不需要用户安装。

## 通过原生插件命令安装

这是一个临时安装插件：安装 → 执行一次 setup → 自动卸载插件及 skill。日常主题由独立二进制运行，开发会话不会持续加载安装 skill。

仓库目前为私有，需要当前 GitHub 账号有访问权限。下面使用 SSH；也可换成有权限的 Git URL。

### Codex

```sh
codex plugin marketplace add git@github.com:DengNaichen/ciyuan-cli-theme.git
codex plugin add ciyuan-theme@ciyuan-theme
```

在新 Codex 会话中选择本插件的 `setup` skill，或说「用词元无限插件安装 Codex 主题，成功后清理安装插件」。若也要 Claude Code，明确说「两个都安装」。安装向导完成后会执行 `codex plugin remove ciyuan-theme@ciyuan-theme`。

### Claude Code

```sh
claude plugin marketplace add git@github.com:DengNaichen/ciyuan-cli-theme.git
claude plugin install ciyuan-theme@ciyuan-theme --scope user
```

在 Claude Code 会话中运行 `/ciyuan-theme:setup`。完成后安装向导会执行 `claude plugin uninstall ciyuan-theme@ciyuan-theme --scope user`。

插件卸载后，原来已经打开的会话可能仍保留加载过的内容；新会话不再发现该 skill。Marketplace 目录登记会保留，方便下次更新，它本身不会给模型加载 skill。

## 不使用 skill

也可以直接运行当前平台的二进制：

```sh
# macOS Apple Silicon 示例，文件位于插件包的 bin/ 中
./ciyuan-theme-darwin-arm64 install --apps codex,claude
```

安装器将自身复制到 `~/.local/bin/ciyuan-theme`，并接入 zsh/bash。重新打开终端，照常运行 `codex` 或 `claude`。首次安装会替换此前的词元无限 Python 启动段；旧 Python 文件不再被启动，仍可用于恢复。

已有的本主题配置段支持重复安装并保留另一个宿主的启动函数；自定义 alias 会被提示人工接入，以免丢失用户参数。安装器不会修改原程序的配置或权限参数。历史的 Python `config.json` 不属于 Go 运行时，若曾自行配置额外启动参数，请加入 shell 函数调用。

## 使用、更新与移除

```sh
~/.local/bin/ciyuan-theme preview
PANGU=off codex
PANGU_MOTION=reduce claude
PANGU_THEME=light codex
~/.local/bin/ciyuan-theme uninstall
```

- `preview` 只播放动画，不启动 AI 会话。
- `PANGU=off` 单次跳过主题；`PANGU_MOTION=reduce` 改成一秒静态画面。
- `PANGU_THEME=light` 选择浅色配色；`PANGU_GLYPHS=octant` / `quadrant` 控制字符精度。默认 Ghostty 使用八分块，其他终端使用四分块。
- `--help`、`--version`、`exec` / `-p` 等命令直接交给原程序；恢复会话跳过开屏。
- `uninstall` 只移除当前 zsh/bash 的主题函数，保留二进制和备份。Shell 与旧二进制备份位于 `~/.config/ciyuan-theme/backups/`。
- 更新：用原生 marketplace 更新命令刷新目录，临时安装插件，重新执行 setup。最后仍然清理安装插件。

Go 版本先播放约三秒动画，再启动程序。窗口变化会更新 PTY 尺寸；首页只画在程序的已知空白区域，遇到不能追踪的终端操作会暂停覆盖直到清屏。不同版本的原程序可能改变界面布局；此覆盖层不是官方 UI 插件接口。新版不读取 Python 配置，也不使用 Python 版的 `PANGU_MODE` / `PANGU_HOME` 环境变量。

## 从源码构建

开发者需要 Go 1.24+；使用者只需要预编译二进制。

```sh
CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o ciyuan-theme .
./ciyuan-theme install --apps codex
```

`assets/art.json` 为已生成的矢量面、渐变和英文标题点阵，使用 `go:embed` 嵌入。原始 SVG 在 `assets/ciyuan-mark.svg`。Python 版本保留于 Git 历史的 `780ea35` 提交。

## 来源与许可

动画设计和终端覆盖机制源自 [Lightly20110815/pangu-cli-theme](https://github.com/Lightly20110815/pangu-cli-theme)，本版本用 Go 重写；保留原 MIT 许可。第三方 Go 库许可位于 `THIRD_PARTY/`。词元品牌 SVG 和标识权利归其权利人所有，不因代码 MIT 许可而自动授权。非 OpenAI / Anthropic 官方主题。
