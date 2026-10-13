---
name: setup
description: 安装或更新词元无限 Go 终端主题，并在成功后卸载本安装插件及其 skill。仅用于用户请求启用该主题，不用于普通开发任务。
---

这是一次性安装向导。日常主题运行不依赖插件、模型、Python、Node 或 Go 工具链。

1. 从包含此 SKILL.md 的 skill 目录，以 `../..` 定位插件根目录；路径包含 `scripts/setup.sh` 和 `bin/`。根据用户要求选择 `codex`、`claude` 或 `codex,claude`，未指定时仅安装当前宿主。不要重新询问已经明确的选择。
2. 告知用户：将把独立二进制复制到 `~/.local/bin/ciyuan-theme`，备份并更新当前 zsh/bash 启动文件。旧的本主题 Python 启动段会替换为 Go 启动段；Codex/Claude 的程序与配置不修改。
3. 执行 `sh "<插件根目录>/scripts/setup.sh" "<所选程序>"`。以安装器实际退出状态为准，不下载依赖、不现场编译、不运行交互会话或测试。失败时保留插件，说明具体错误，不宣称完成。
4. 安装成功后，移除当前宿主中的临时安装插件，以清除本 skill：
   - Codex：`codex plugin remove ciyuan-theme@ciyuan-theme`
   - Claude Code：`claude plugin uninstall ciyuan-theme@ciyuan-theme --scope user`
   只清理本插件，不移除其他插件或 marketplace，不调用主题二进制的 uninstall。独立二进制已经复制到插件目录之外，所以删除插件缓存不会破坏主题。若当前是仓库源文件而非已安装插件，说明不存在需卸载的实例；不要删除源码里的 skill。
5. 清理失败时明确报告“主题已安装，但安装插件未移除”并给出上面的对应清理命令。全部成功后告知新开终端或重新加载 shell；当前会话中已加载的 skill 内容不会追溯消失，新会话不再加载。

更新仍使用同一流程：原生命令临时安装插件 → 运行 setup → 卸载插件。永久关闭主题是另一项用户操作：`~/.local/bin/ciyuan-theme uninstall`。不要将主题卸载与安装向导清理混淆。
