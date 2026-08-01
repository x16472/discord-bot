#!/usr/bin/env bash

set -uo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
cd "$script_dir" || exit 1

go run . &
bot_pid=$!

forward_signal() {
    kill -TERM "$bot_pid" 2>/dev/null || true
}

trap forward_signal INT TERM
wait "$bot_pid"
exit_code=$?

trap - INT TERM
exit "$exit_code"
