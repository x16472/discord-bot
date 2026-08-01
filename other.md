# 各系統環境的啟動流程

## Windows 啟動流程

### 啟動前檢查

1. 開啟命令提示字元或 PowerShell。
2. 確認已安裝 Go 1.19 以上版本：

```powershell
go version
```

3. 切換至專案目錄：

```powershell
cd E:\你的路徑\discord-bot
```

4. 確認專案根目錄已有 `.env` 與 `talk.txt`。
5. 確認 `.env` 已設定有效的 Discord Bot Token：

```dotenv
DCToken=你的_Discord_Bot_Token
```

### 使用 PowerShell 啟動

```powershell
.\start.bat
```

### 使用命令提示字元啟動

```bat
start.bat
```

Windows 啟動流程如下：

1. `start.bat` 自動切換至啟動檔所在的專案目錄。
2. 使用 `go run .` 編譯並啟動 Bot。
3. 使用 `start /b /wait` 等待 Bot 程序結束。
4. Bot 結束後，批次檔會回傳相同的結束代碼。

停止機器人時，在執行中的終端機按下 `Ctrl+C`。

> 【注意】<br>
> 建議從 PowerShell 或命令提示字元執行，不要直接雙擊 `start.bat`。直接雙擊時，若程式發生錯誤，視窗可能立即關閉而看不到錯誤訊息。

> 【注意】<br>
> 如果出現「無法辨識 go」或「`go` is not recognized」訊息，代表 Go 尚未安裝，或 Go 的執行路徑尚未加入 Windows 的 `PATH`。

## Linux Debian 啟動流程：這裡用 OMV（Debian）啟動

以下流程適用於透過 SSH 登入 OpenMediaVault 所使用的 Debian 系統。

### 1. 進入專案並確認必要工具

```bash
cd ~/discord-bot
go version
command -v bash awk mktemp go
```

Go 版本必須為 1.19 以上。若系統尚未安裝 Go，可先使用 Debian 套件管理工具安裝，再重新確認版本：

```bash
sudo apt update
sudo apt install golang-go
go version
```

> 【注意】<br>
> 如果 SSH 提示字元是 `root@openmediavault`，代表目前已經是 `root`，上述安裝指令可省略 `sudo`。但 Bot 正式長期運作時仍不建議使用 `root` 帳號。

> 【注意】<br>
> 不同 OMV 版本所使用的 Debian 套件版本可能不同。如果 `apt` 安裝的 Go 低於 1.19，請使用 [Go 官方安裝方式](https://go.dev/doc/install)，不要忽略 `go.mod` 指定的版本需求。

### 2. 確認部署檔案版本一致

如果使用 Git 部署，先取得最新版本：

```bash
git pull
git status --short
```

確認 Linux 上的 `main.go` 已使用半形減號解析規則：

```bash
grep -n 'SplitN(line' main.go
```

正確結果應包含：

```go
strings.SplitN(line, "-", 3)
```

> 【注意】<br>
> `main.go`、`talk.txt` 與 `start.sh` 必須是同一版本。如果錯誤訊息仍出現 `separated by tabs`，代表 Debian 上的 `main.go` 仍是舊版；只更新 `talk.txt` 無法解決問題。

### 3. 檢查環境變數與權限

確認 `.env` 存在：

```bash
ls -la .env
```

限制 `.env` 僅能由檔案擁有者讀寫：

```bash
chmod 600 .env
```

> 【注意】<br>
> 不要把 `.env`、Bot Token 或 Token 畫面提交至 Git。若 Token 曾經公開，應立即前往 Discord Developer Portal 重新產生 Token。

### 4. 檢查 Linux 換行格式

從 Windows 複製到 Debian 的腳本可能帶有 CRLF 換行。可先檢查：

```bash
file start.sh talk.txt main.go
```

若輸出包含 `CRLF`，可使用 `sed` 轉換成 Linux 的 LF：

```bash
sed -i 's/\r$//' start.sh talk.txt main.go
```

> 【注意】<br>
> 如果看到 `/usr/bin/env: 'bash\r': No such file or directory`、`$'\r': command not found` 或類似錯誤，通常就是 `start.sh` 使用了 CRLF 換行。

### 5. 檢查 talk.txt 規則

每一筆非註解規則都必須使用兩個半形減號：

```text
比對方式-觸發文字-回覆內容
```

正確範例：

```text
exact-午安-午安！記得吃午餐。
action-瑪麗亞凱莉解凍-christmas_countdown
```

> 【注意】<br>
> 不可使用 Tab、空格、全形破折號 `—` 或全形減號取代半形減號 `-`。觸發文字本身不可包含半形減號；第三欄的回覆內容可以包含減號。

### 6. 確認 OMV 時區

`現在時間`、耶誕節倒數及農曆新年倒數都使用 Debian 主機的 `time.Now()`，因此應先確認系統時區：

```bash
timedatectl status
date
```

若需要使用台灣時間，可由具有管理權限的人員評估後設定：

```bash
sudo timedatectl set-timezone Asia/Taipei
```

> 【注意】<br>
> `timedatectl set-timezone` 會變更整台 OMV 主機的系統時區，不只影響 Discord Bot。主機若還有其他服務，請先確認影響再執行。

### 7. 啟動 Bot

不需要先設定執行權限，可直接透過 Bash 啟動：

```bash
bash start.sh
```

如果想直接執行腳本，需先設定權限：

```bash
chmod +x start.sh
./start.sh
```

Debian 啟動流程如下：

1. `start.sh` 自動切換至專案目錄。
2. 使用 `go build` 將執行檔建立於 `~/.local/state/discord-bot/`。
3. 使用 `nohup` 在背景啟動 Bot，並將標準輸入導向 `/dev/null`。
4. 將 Bot 的 PID 寫入 `discord-bot.pid`，輸出寫入 `discord-bot.log`。
5. 等待一秒確認 Bot 沒有因設定錯誤立即結束。
6. 啟動成功後立即返回命令列，Bot 不會繼續占用 SSH 終端。

> 【注意】<br>
> 不建議使用 `root` 帳號長期執行 Bot。正式運作時應建立權限受限的專用帳號，並只授予讀取專案與 `.env`、寫入必要目錄及連線網路所需的權限。

> 【注意】<br>
> `start.sh` 使用 `nohup` 讓 Bot 脫離 SSH 終端，關閉 SSH 後仍可繼續執行。若需要開機自動啟動、失敗後自動重啟及集中管理日誌，仍建議改用 `systemd`。

### OMV 常見錯誤

| 錯誤或現象 | 可能原因 | 處理方式 |
| --- | --- | --- |
| `separated by tabs` | Debian 上仍是舊版 `main.go` | 同步最新版 `main.go`、`talk.txt`、`start.sh` 後重新建置 |
| `必須使用半形減號分隔` | `talk.txt` 某一行格式錯誤 | 依錯誤行號改成 `類型-觸發文字-回覆內容` |
| `$'\r': command not found` | `start.sh` 使用 CRLF 換行 | 將腳本轉換成 LF |
| `go: command not found` | Go 未安裝或不在 `PATH` | 安裝 Go 並重新登入 SSH 工作階段 |
| `permission denied` 且路徑位於 `.local/state` | 執行帳號無法寫入自己的狀態目錄 | 檢查家目錄、`.local` 與其子目錄的擁有者及權限 |
| `401 Unauthorized` | `.env` Token 錯誤、過期或已重設 | 更新 `DCToken` 後重新啟動 |
| Bot 在線但不回覆 | Discord 權限、訊息內容設定或頻道權限不足 | 檢查 Developer Portal 與伺服器頻道權限 |
| 回覆時間不正確 | OMV 主機時區不正確 | 使用 `timedatectl status` 檢查時區 |
| SSH 關閉後 Bot 停止 | 使用到舊版啟動檔，或不是透過 `nohup` 啟動 | 同步新版 `start.sh` 後重新啟動 |

目前背景執行所使用的檔案預設位於：

```text
~/.local/state/discord-bot/discord-bot
~/.local/state/discord-bot/discord-bot.pid
~/.local/state/discord-bot/discord-bot.log
```

### 非同步啟動與等待原則

Go 語言沒有 `async` 與 `await` 關鍵字。Discord 的訊息事件由事件處理機制執行，而各平台的啟動方式如下：

- Windows 使用 `start /b /wait` 啟動並等待 `go run .`。
- Debian 使用 `nohup` 與背景程序 `&` 啟動已編譯的執行檔，記錄 PID 後立即返回命令列。

Debian 背景程序不再依賴原本的 SSH 工作階段，可透過 PID 檔及日誌持續查閱與管理。

### 背景程序查閱與注意事項

目前 `start.sh` 建立的執行檔名稱是 `discord-bot`，可使用以下指令搜尋背景程序：

```bash
ps aux | grep discord-bot
```

`ps aux` 會列出系統程序，`grep discord-bot` 則篩選出命令列包含 `discord-bot` 的項目。主要欄位包含執行帳號、PID、CPU、記憶體、啟動時間、累計執行時間及完整指令；查閱時應確認程序的執行帳號、PID 與命令路徑是否符合預期。

> 【注意】<br>
> `grep discord-bot` 本身通常也會出現在搜尋結果中，因此只看到含有 `grep` 的那一行，不代表 Bot 正在執行。可以改用下列寫法，避免列出 `grep` 自己：

```bash
ps aux | grep '[d]iscord-bot'
```

若要精確查閱由 `start.sh` 記錄的程序，建議直接讀取 PID 檔：

```bash
ps -p "$(cat ~/.local/state/discord-bot/discord-bot.pid)" \
  -o pid,ppid,user,stat,lstart,etime,%cpu,%mem,cmd
```

也可以使用 `pgrep` 依名稱及完整命令列查詢：

```bash
pgrep -af 'discord-bot'
```

查看即時日誌：

```bash
tail -f ~/.local/state/discord-bot/discord-bot.log
```

確認 PID 是否仍在執行：

```bash
kill -0 "$(cat ~/.local/state/discord-bot/discord-bot.pid)"
echo $?
```

結束代碼為 `0` 代表該 PID 存在且目前帳號有權限傳送訊號；非 `0` 可能表示程序已結束、PID 檔過期，或目前帳號權限不足。

正常停止 Bot：

```bash
kill "$(cat ~/.local/state/discord-bot/discord-bot.pid)"
```

> 【注意】<br>
> 執行 `kill` 前應先使用 `ps` 核對 PID、執行帳號與命令路徑，避免 PID 檔過期後誤停止其他程序。停止後再次執行 `start.sh` 時，腳本會自動清除無效的舊 PID 檔。

> 【注意】<br>
> 不要直接使用 `kill -9` 作為一般停止方式。`SIGKILL` 不允許程式執行正常的連線關閉與清理流程，只應在一般 `kill` 無法結束程序時作為最後手段。

> 【注意】<br>
> `nohup` 只能讓程序脫離終端，無法提供開機自動啟動、異常自動重啟、日誌輪替或服務相依管理。OMV 長期正式運作仍以 `systemd` 服務較合適。
