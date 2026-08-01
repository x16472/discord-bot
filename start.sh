#!/usr/bin/env bash

set -Eeuo pipefail
umask 077

# 固定從啟動檔所在目錄執行，確保程式能讀取 .env 與 talk.txt。
script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
cd "$script_dir"

# 將執行檔、PID 與日誌存放在專案目錄之外，避免影響 Git 狀態。
runtime_dir="${XDG_STATE_HOME:-$HOME/.local/state}/discord-bot"
app_path="$runtime_dir/discord-bot"
pid_file="$runtime_dir/discord-bot.pid"
log_file="$runtime_dir/discord-bot.log"
mkdir -p "$runtime_dir"

# 避免重複啟動仍在執行的 Discord Bot。
if [[ -f "$pid_file" ]]; then
    existing_pid=""
    read -r existing_pid < "$pid_file" || true
    if [[ "$existing_pid" =~ ^[0-9]+$ ]] && kill -0 "$existing_pid" 2>/dev/null; then
        running_app="$(readlink -f "/proc/$existing_pid/exe" 2>/dev/null || true)"
        if [[ "$running_app" == "$app_path" ]]; then
            printf 'Discord Bot 已在背景執行，PID：%s\n' "$existing_pid"
            printf '日誌：%s\n' "$log_file"
            exit 0
        fi
    fi
    rm -f -- "$pid_file"
fi

# 先完成編譯，避免背景程序啟動後才回報編譯錯誤。
go build -o "$app_path" .

# 使用 nohup 脫離 SSH 終端，並將輸出集中寫入日誌。
nohup "$app_path" >> "$log_file" 2>&1 < /dev/null &
bot_pid=$!
printf '%s\n' "$bot_pid" > "$pid_file"

# 確認 Bot 沒有在啟動後立即因設定錯誤而結束。
sleep 1
if ! kill -0 "$bot_pid" 2>/dev/null; then
    rm -f -- "$pid_file"
    printf 'Discord Bot 啟動失敗，請檢查日誌：%s\n' "$log_file" >&2
    tail -n 20 "$log_file" >&2 || true
    exit 1
fi

printf 'Discord Bot 已在背景啟動，PID：%s\n' "$bot_pid"
printf '日誌：%s\n' "$log_file"
printf '停止指令：kill %s\n' "$bot_pid"
