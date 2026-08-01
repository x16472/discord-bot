#!/usr/bin/env bash

set -Eeuo pipefail

# 固定從啟動檔所在目錄執行，確保能找到 talk.txt 與 go.mod。
script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
cd "$script_dir"

talk_file="$script_dir/talk.txt"
app_path=""
bot_pid=""

# 在編譯前驗證每筆對話規則是否使用兩個半形減號分隔三個欄位。
validate_talk_file() {
    if [[ ! -f "$talk_file" ]]; then
        printf '錯誤：找不到對話規則檔案 %s\n' "$talk_file" >&2
        return 1
    fi

    awk '
        /^[[:space:]]*$/ || /^[[:space:]]*#/ { next }
        $0 !~ /^[^-]+-[^-]+-.+$/ {
            printf "錯誤：talk.txt 第 %d 行必須使用半形減號分隔比對方式、觸發文字與回覆內容\n", NR > "/dev/stderr"
            invalid = 1
        }
        END { exit invalid }
    ' "$talk_file"
}

# 結束腳本時只清理由本次執行建立的暫存檔案。
cleanup() {
    if [[ -n "$app_path" && -f "$app_path" ]]; then
        rm -f -- "$app_path"
    fi
}

# 收到停止訊號時轉送給背景執行的 Discord Bot。
forward_signal() {
    if [[ -n "$bot_pid" ]]; then
        kill -TERM "$bot_pid" 2>/dev/null || true
    fi
}

trap cleanup EXIT
trap forward_signal INT TERM

validate_talk_file

# 使用唯一的暫存路徑，避免覆寫專案內既有的 app 檔案。
app_path="$(mktemp "${TMPDIR:-/tmp}/discord-bot.XXXXXX")"
go build -o "$app_path" .

# 在背景啟動機器人，再等待程序結束並保留其結束代碼。
"$app_path" &
bot_pid=$!
if wait "$bot_pid"; then
    exit_code=0
else
    exit_code=$?
fi

trap - INT TERM
exit "$exit_code"
