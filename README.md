# 词元无限 · Codex CLI 主题

蓝青色几何标志从代码碎片中聚合成形，中文标题逐字显现，最后由一道光扫过并过渡到首页。使用提供的「词元」SVG，保留六个几何面的原始形状和渐变。

![开屏动画](docs/intro.gif)

## 安装

需要 Python 3.8+ 和已安装的 Codex CLI。运行时仅使用 Python 标准库。

```sh
gh repo clone DengNaichen/ciyuan-cli-theme
cd ciyuan-cli-theme
python3 install.py --apps codex --dry-run
python3 install.py --apps codex
```

重新打开终端，运行 `codex`。安装器会备份已有 shell 启动文件，安装到 `~/.ciyuan-cli-theme`，并添加带标记的启动函数。不会修改 Codex 可执行文件或用户的 Codex 配置文件。PTY 模式沿用上游的命令行参数，关闭 Codex 自带欢迎动画和提示文字，为自定义首页留出空间。

如果此前装过盘古主题，先运行 `python3 ~/.pangu-cli-theme/install.py --uninstall --keep-files`，再安装本主题，避免两个 shell 函数互相覆盖。旧主题文件保留，可重新执行它的安装器恢复。

## 使用

- `PANGU=off codex`：单次跳过主题。
- `PANGU_MOTION=reduce codex`：使用约一秒的静态画面。
- `PANGU_THEME=light codex`：浅色终端配色。
- `PANGU_MODE=simple codex`：先播放动画，再启动 Codex，不使用 PTY 首页。
- `python3 ~/.ciyuan-cli-theme/install.py --status`：查看安装状态。
- `python3 ~/.ciyuan-cli-theme/install.py --uninstall`：卸载。

为兼容上游，环境变量仍使用 `PANGU_*` 前缀。`--help`、`--version`、`exec` 等命令直接转交原程序；`resume` 和 `fork` 跳过开屏。macOS / Linux 支持动画和首页，Windows 沿用上游的独立开屏方式。终端使用字符块显示矢量图案，因此实际精细程度取决于终端支持；Ghostty 使用八分块，其他终端默认四分块。Codex 更新界面后，顶部文字替换或首页覆盖可能需要适配。

## 开发

```sh
python3 -m unittest discover -s tests
# 构建预览需要 Pillow；中文点阵生成还需要 macOS PingFang 字体。
python3 -m pip install Pillow
python3 tools/build_assets.py
python3 tools/render.py gif codex docs/intro.gif 120 38
python3 tools/render.py sheet codex docs/frames.png 120 38
```

原始 SVG 位于 `pangu/assets/ciyuan-mark.svg`。`art.json` 保存形状、每个面的渐变和预生成中文字形，因此用户运行时不需要安装字体或 Pillow。核心 Python 包名保留 `pangu`，便于对照上游。

## 来源与许可

基于 [Lightly20110815/pangu-cli-theme](https://github.com/Lightly20110815/pangu-cli-theme) 的终端渲染、PTY 转发、首页覆盖与安装器制作；保留原作者的 [MIT 许可](LICENSE)。本版本替换了品牌图形、标题、配色和动画运动方式。提供的「词元」品牌 SVG 及标识权利归其权利人所有，不因代码的 MIT 许可而自动授权。非 OpenAI 官方主题。
