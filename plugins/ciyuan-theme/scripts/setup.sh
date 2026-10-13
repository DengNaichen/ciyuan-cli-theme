#!/bin/sh
set -eu
ciyuan_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
case $(uname -s) in Darwin) ciyuan_os=darwin ;; Linux) ciyuan_os=linux ;; *) echo '当前二进制支持 macOS / Linux。' >&2; exit 1 ;; esac
case $(uname -m) in arm64|aarch64) ciyuan_arch=arm64 ;; x86_64|amd64) ciyuan_arch=amd64 ;; *) echo '当前二进制支持 ARM64 / x86_64。' >&2; exit 1 ;; esac
ciyuan_apps=${1:-codex}
exec "$ciyuan_root/bin/ciyuan-theme-$ciyuan_os-$ciyuan_arch" install --apps "$ciyuan_apps"
