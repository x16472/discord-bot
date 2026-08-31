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
5. 確認 `.env` 已設定 Discord Token 及天氣功能需要的設定：

```dotenv
DCToken=你的_Discord_Bot_Token
CWA_API=你的_中央氣象署_API_Key
CWA_LOCATION=臺北市
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
4. Bot 結束後，批次檔回傳相同的結束代碼。

停止機器人時，在執行中的終端機按下 `Ctrl+C`。

> 【注意】<br>
> 建議從 PowerShell 或命令提示字元執行，不要直接雙擊 `start.bat`。直接雙擊時，若程式發生錯誤，視窗可能立即關閉而看不到錯誤訊息。

> 【注意】<br>
> 如果出現「無法辨識 go」或「`go` is not recognized」訊息，代表 Go 尚未安裝，或 Go 的執行路徑尚未加入 Windows 的 `PATH`。

## Linux Debian 啟動流程：OMV systemd

以下流程以目前 OMV 的 `/root/discord-bot` 專案路徑為準。新版 `start.sh` 不再自行使用 `nohup`、`&`、PID 檔或背景程序；它會完成編譯後使用 `exec` 啟動 Bot，由 systemd 負責背景執行、停止、重啟及日誌。

### 1. 確認專案路徑與工具

```bash
cd /root/discord-bot
pwd
go version
git --version
```

`pwd` 預期顯示：

```text
/root/discord-bot
```

如果實際路徑不同，安裝 Service 前必須同步修改 `discord-bot.service` 的 `WorkingDirectory` 與 `ExecStart`。

### 2. 確認必要檔案

```bash
ls -l main.go weather.go talk.txt start.sh discord-bot.service .env
```

`.env` 至少需要：

```dotenv
DCToken=你的_Discord_Bot_Token
CWA_API=你的_中央氣象署_API_Key
CWA_LOCATION=臺北市
```

- `DCToken` 用於連線 Discord。
- `CWA_API` 用於中央氣象署 API。
- `CWA_LOCATION` 是文字指令 `天氣` 與 `下雨` 查詢的縣市。
- `.env` 不得提交至 Git，也不要將內容貼進公開日誌。

### 3. 檢查 Linux 換行與 Shell 語法

Windows 複製到 Debian 的 Shell 檔案可能含有 CRLF。先檢查：

```bash
file start.sh
bash -n start.sh
```

若看到 `/usr/bin/env: 'bash\r': No such file or directory` 或 `$'\r': command not found`，可轉換為 LF：

```bash
sed -i 's/\r$//' start.sh
```

### 4. 驗證 Go 程式與對話規則

```bash
go test ./...
go vet ./...
```

也可以直接以前景方式確認一次：

```bash
bash start.sh
```

前景測試成功後按下 `Ctrl+C`。`main.go` 會接收中止訊號並關閉 Discord Session。

> 【注意】<br>
> 如果 SSH 提示字元是 `root@openmediavault`，代表目前已經是 `root`，上述安裝指令可省略 `sudo`。但 Bot 正式長期運作時仍不建議使用 `root` 帳號。

> 【重要】<br>
> 新版 `start.sh` 是 systemd 的前景入口，不要再使用 `nohup bash start.sh &`。自行背景化會讓 systemd 無法正確追蹤真正的 Bot 程序。

### 5. 安裝 systemd Service

將專案內的 Service 複製到 systemd：

```bash
install -m 0644 discord-bot.service /etc/systemd/system/discord-bot.service
systemctl daemon-reload
systemctl enable --now discord-bot.service
```

這些動作會：

1. 安裝 Service 定義。
2. 重新載入 systemd 設定。
3. 設定開機自動啟動。
4. 立即啟動 Discord Bot。

`start.sh` 會將編譯檔放在 `/var/lib/discord-bot/discord-bot`。此目錄由 Service 的 `StateDirectory=discord-bot` 建立，不會污染 Git 工作樹。

### 6. 啟動、停止、重新啟動與狀態

啟動：

```bash
systemctl start discord-bot.service
```

停止：

```bash
systemctl stop discord-bot.service
```

重新編譯並啟動目前專案內容：

```bash
systemctl restart discord-bot.service
```

查看狀態：

```bash
systemctl status discord-bot.service --no-pager
```

取消開機自動啟動並立即停止：

```bash
systemctl disable --now discord-bot.service
```

systemd 停止時會傳送 `SIGTERM`。若 Bot 未在 `TimeoutStopSec=15` 內結束，systemd 才會進一步終止程序，因此不需要再手動管理 PID 檔或直接使用 `kill -9`。

> 【注意】<br>
> 不建議使用 `root` 帳號長期執行 Bot。正式運作時應建立權限受限的專用帳號，並只授予讀取專案與 `.env`、寫入必要目錄及連線網路所需的權限。

> 【注意】<br>
> 不同 OMV 版本所使用的 Debian 套件版本可能不同。如果 `apt` 安裝的 Go 低於 1.19，請使用 [Go 官方安裝方式](https://go.dev/doc/install)，不要忽略 `go.mod` 指定的版本需求。

### 7. 查閱日誌

顯示最近 100 行：

```bash
journalctl -u discord-bot.service -n 100 --no-pager
```

持續追蹤：

```bash
journalctl -u discord-bot.service -f
```

顯示本次開機後的日誌：

```bash
journalctl -u discord-bot.service -b --no-pager
```

程式的標準輸出與錯誤都交由 journal 保存，不再寫入 `~/.local/state/discord-bot/discord-bot.log`。

### 8. 查閱實際程序

systemd 提供的主程序資訊比 `ps | grep` 更精確：

```bash
systemctl show discord-bot.service --property=MainPID,ActiveState,SubState,ExecMainStatus
```

仍可使用下列指令交叉確認：

```bash
ps aux | grep discord-bot
```

`grep` 本身也可能出現在結果中，因此應以 `systemctl show` 的 `MainPID` 為主要依據。

### 9. 更新程式後重新部署

同步程式碼後先驗證，再重新啟動：

```bash
cd /root/discord-bot
go test ./...
bash -n start.sh
systemctl restart discord-bot.service
systemctl status discord-bot.service --no-pager
```

若修改過 `discord-bot.service`，需要重新安裝並載入：

```bash
install -m 0644 discord-bot.service /etc/systemd/system/discord-bot.service
systemctl daemon-reload
systemctl restart discord-bot.service
```

### 10. 常見錯誤

| 錯誤或現象 | 可能原因 | 處理方式 |
| --- | --- | --- |
| `separated by tabs` | Debian 上仍是舊版 `main.go` | 同步同一版本的 `main.go` 與 `talk.txt` |
| `unsupported action "local_time"` | 只更新 `talk.txt`，未更新 `main.go` | 同步新版 `main.go` 後重新啟動 |
| `CWA_API 不可為空` | `.env` 缺少中央氣象署 API Key | 補上 `CWA_API` |
| 找不到天氣資料 | `CWA_LOCATION` 不是 API 支援的縣市名稱 | 改用例如 `臺北市`、`臺中市` 的正式名稱 |
| `$'\r': command not found` | `start.sh` 使用 CRLF | 將腳本轉換成 LF |
| `status=203/EXEC` | Service 路徑錯誤或 Bash 不存在 | 檢查 `ExecStart` 與 `/usr/bin/env bash` |
| `go: command not found` | systemd 的 PATH 找不到 Go | 確認 Go 位於 Service 設定的 PATH，或調整 PATH |
| Service 不斷重新啟動 | Token、規則或編譯錯誤 | 使用 `journalctl -u discord-bot.service -n 100` 查閱原因 |
| 修改程式後行為沒變 | 尚未重新啟動 Service | 執行 `systemctl restart discord-bot.service` |

### systemd 注意事項

- 目前 Service 路徑依照 `/root/discord-bot` 設定，適用於現有 OMV 部署位置。
- Service 目前以系統管理者身分執行；若未來提高安全要求，建議建立專用的低權限帳號及專屬部署目錄。
- `Restart=on-failure` 只在異常結束時重新啟動；執行 `systemctl stop` 不會觸發自動重啟。
- `start.sh` 每次啟動都會重新編譯，能確保使用目前程式碼，但啟動主機必須保留 Go 工具鏈。
- `.env` 由程式在 `WorkingDirectory` 中透過 godotenv 載入，不需要再寫入 Service 檔。
- 不要同時使用舊版 `nohup` 程序與 systemd，否則可能出現兩個 Bot Session 同時登入。
