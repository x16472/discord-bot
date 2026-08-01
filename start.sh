#!/usr/bin/env bash

set -Eeuo pipefail
umask 077

# 固定從啟動檔所在目錄執行，確保程式能讀取 .env 與 talk.txt。
script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
cd "$script_dir"

# 將 systemd 執行的編譯檔存放在狀態目錄，避免影響 Git 工作樹。
runtime_dir="${DISCORD_BOT_STATE_DIR:-/var/lib/discord-bot}"
app_path="$runtime_dir/discord-bot"
mkdir -p "$runtime_dir"

# 啟動服務前先完成編譯，讓編譯錯誤直接由 systemd 判定為啟動失敗。
go build -o "$app_path" .

# 使用 exec 讓 Bot 取代 Shell，交由 systemd 直接追蹤、停止與重新啟動。
exec "$app_path"
