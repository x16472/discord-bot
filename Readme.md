# Discord Bot

## 專案簡介

這是一個採用 Go/Python 混合式架構的 Discord 文字訊息機器人。

程式採用事件驅動架構，啟動後會監聽伺服器中的文字訊息，並依照訊息內容回覆或執行對應功能。

Go 是常駐主程式，負責 Discord 連線、訊息事件、文字規則、日期、天氣與股票功能控制；Python 只在收到股票查詢時執行一次，向臺灣證券交易所取得資料並以 UTF-8 JSON 回傳給 Go。股票查詢完成後 Python 程序會立即結束，不需要常駐服務，也不使用任何 TCP Port。

小助手是Codex。

## 主要功能

- 由 `talk.txt` 管理完全比對、部分比對與動態 action。
- 計算耶誕節與農曆新年倒數。
- 顯示 Bot 主機目前時間。
- 查詢中央氣象署未來 36 小時天氣與降雨機率。
- 查詢指定股票最新交易日行情與名稱。
- 查詢最近可用交易日的臺股大盤資訊。
- 依近期收盤價與均價產生規則式股票觀察。
- 支援 Windows 開發啟動及 OMV／Debian systemd 部署。

## 開發環境

- Go 1.25
- [discordgo](https://github.com/bwmarrin/discordgo) v0.26.1
- [godotenv](https://github.com/joho/godotenv) v1.4.0
- [lunar-go](https://github.com/6tail/lunar-go) v1.4.6：農曆與國曆日期換算
- [go-cwb](https://github.com/minchao/go-cwb)：串接中央氣象署開放資料 API

## 執行環境

- Go 1.25
- Python 3
- DiscordGo v0.26.1
- godotenv v1.4.0
- lunar-go v1.4.6
- go-cwb

Python 股票程式只使用標準函式庫，不需要安裝 `requests`、gRPC 或 protobuf 套件。

## 專案結構

```text
discord-bot/
├── .agent/
│   ├── chat.md             # 架構決策、限制與歷史討論
│   └── History.md          # 開發、變更與排錯歷程
├── .env                    # 本機環境變數，不納入版本控制
├── .gitignore              # Git 排除規則
├── discord-bot.service     # OMV／Debian systemd Service
├── DiscordGo.md            # DiscordGo 參數與功能備忘
├── go.mod                  # Go 模組與直接相依套件
├── go.sum                  # Go 相依套件校驗資訊
├── knowhow.md              # 部署及維護經驗整理
├── main.go                 # Discord 事件、規則載入與 action 分派
├── Readme.md               # 專案說明
├── start.bat               # Windows 啟動檔
├── start.sh                # Linux／systemd 前景啟動檔
├── stock.go                # 股票功能主體、Python 呼叫與資料格式化
├── stock.py                # 證交所即時資料擷取與 JSON 輸出
├── talk.txt                # 文字回覆與 action 規則
└── weather.go              # 中央氣象署天氣與降雨功能
```

`stock.proto`、`stock_pb2.py`、`stock_pb2_grpc.py`、`requirements.txt` 與 `stock_test.go` 已移除，目前專案不依賴這些檔案。

## Go/Python 混合式架構

```text
Discord 文字訊息
    ↓
main.go：messageCreate()
    ├── 帶股票代號的訊息
    │       例如：2377股價、2377股票建議
    │       ↓ goroutine
    │   executeStockEvent()
    │       ↓
    │   stock.go
    │       ↓ exec.CommandContext()
    │   stock.py
    │       ↓ urllib
    │   臺灣證券交易所
    │       ↓ CSV
    │   stock.py 輸出 UTF-8 JSON
    │       ↓
    │   stock.go 解析並回覆 Discord
    │
    └── talk.txt：exact／contains／action
            ↓
        executeTalkAction()
            ├── 耶誕節倒數
            ├── 農曆新年倒數
            ├── 現在時間
            ├── 天氣與降雨
            ├── 預設個股行情
            ├── 單日大盤
            └── 預設股票建議
```

股票資料不會在 Bot 啟動時預先查詢或儲存。每個股票事件都會建立獨立的非同步工作，並以 15 秒逾時限制 Python 與證交所查詢，避免阻塞其他 Discord 訊息。

## 環境變數

在專案根目錄建立 `.env`：

```dotenv
DCToken=你的_Discord_Bot_Token
CWA_API=你的_中央氣象署_API_Key
CWA_LOCATION=臺北市

# 選用：系統無法自動找到 Python 時指定執行檔名稱或完整路徑
PYTHON_BIN=python3

# 選用：stock.py 不在目前工作目錄時指定路徑
STOCK_PYTHON_SCRIPT=stock.py
```

環境變數說明：

| 名稱                  | 必要性             | 用途                                                      |
| --------------------- | ------------------ | --------------------------------------------------------- |
| `DCToken`             | 必要               | Discord Bot Token                                         |
| `CWA_API`             | 使用天氣功能時必要 | 中央氣象署 API Key                                        |
| `CWA_LOCATION`        | 選用               | `天氣` 與 `下雨` 的縣市；未設定時使用臺北市               |
| `PYTHON_BIN`          | 選用               | 指定 Python 3 執行檔；未設定時由程式依作業系統搜尋        |
| `STOCK_PYTHON_SCRIPT` | 選用               | 指定 `stock.py` 路徑；未設定時使用專案根目錄的 `stock.py` |

`.env`、Discord Token 與中央氣象署 API Key 不可提交至版本控制。若 Token 曾經公開，應立即在 Discord Developer Portal 重新產生。

## 安裝與執行

### 下載 Go 相依套件

```bash
go mod download
```

股票功能只需要 Python 3，不需要執行 `pip install`。

### 一般啟動

```bash
go run .
```

只需啟動 Go Bot，不要另外執行 `stock.py`。收到股票指令時，Go 會自動啟動 Python 子程序。

終端機出現下列訊息時代表 Bot 已啟動：

```text
Bot is now running.  Press CTRL-C to exit.
```

按下 `Ctrl+C` 可讓 Go 關閉 Discord Session 後結束。

### Windows

```bat
start.bat
```

如果程式找不到 Python，可在 `.env` 將 `PYTHON_BIN` 設為 Python 執行檔完整路徑。

### OMV／Debian systemd

`discord-bot.service` 預設使用 `/root/discord-bot` 作為專案目錄。若實際部署位置不同，安裝前應調整 `WorkingDirectory` 與 `ExecStart`。

`start.sh` 會將 Go 執行檔建置到 `DISCORD_BOT_STATE_DIR`，再以前景 `exec` 方式交由 systemd 管理。Python 不會隨 Service 啟動；只有股票事件發生時才會執行。

常用指令：

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now discord-bot.service
sudo systemctl status discord-bot.service
sudo journalctl -u discord-bot.service -f
sudo systemctl restart discord-bot.service
sudo systemctl stop discord-bot.service
```

更完整的 OMV 與維護注意事項請參考 `knowhow.md`。

## Discord 指令

### 一般對話與動態 action

| 輸入訊息                                                 | 行為                                     |
| -------------------------------------------------------- | ---------------------------------------- |
| `早安`、`午安`、`晚安`、`謝謝`                           | 固定文字回覆                             |
| `幫助`                                                   | 顯示可使用的文字指令                     |
| 包含 `笨蛋`、`肚子餓`、`好累`、`寫程式`、`bug`、`壓力大` | 對應關鍵字回覆                           |
| `現在時間`                                               | 顯示 Bot 主機目前時間                    |
| `瑪麗亞凱莉解凍`                                         | 計算距離下一個耶誕節的天數               |
| `劉德華解凍`                                             | 計算距離下一個農曆新年的天數             |
| `天氣`                                                   | 查詢 `CWA_LOCATION` 未來 36 小時天氣     |
| `下雨`                                                   | 查詢 `CWA_LOCATION` 未來 36 小時降雨機率 |

### 股票功能

| 輸入訊息       | 行為                                               |
| -------------- | -------------------------------------------------- |
| `個股行情`     | 查詢預設股票 `2377` 的最新交易日行情               |
| `股票建議`     | 依預設股票 `2377` 近期價格產生規則式觀察           |
| `單日大盤`     | 查詢證交所最近可用交易日的大盤資訊                 |
| `2377股價`     | 查詢指定股票代號的名稱與最新交易日行情，以2377為例 |
| `2377股票建議` | 依指定股票近期價格產生規則式觀察，以2377為例       |

個股代號接受四至六碼英數字。個股行情會顯示股票名稱、交易日期、收盤、漲跌、開盤、最高、最低、成交股數與成交筆數。

股票建議只依證交所歷史價格及均價規則產生，不構成投資建議。

## 對話規則

`talk.txt` 每行使用下列格式，欄位之間必須是半形減號：

```text
比對方式-觸發文字-回覆內容
```

| 比對方式   | 說明                                     | 範例                                            |
| ---------- | ---------------------------------------- | ----------------------------------------------- |
| `exact`    | 訊息必須完全符合觸發文字                 | `exact-早安-早安！今天也要保持好心情。`         |
| `contains` | 訊息包含觸發文字即可                     | `contains-bug-別慌，先看錯誤訊息和最近的變更。` |
| `action`   | 訊息完全相符時執行第三欄指定的 Go action | `action-單日大盤-daily_market`                  |

注意事項：

- 空白行及以 `#` 開頭的行會被忽略。
- 解析器只切割前兩個半形減號，第三欄仍可包含減號。
- 未知規則型別、空白欄位或未登錄 action 會讓 Bot 拒絕啟動並指出檔名與行號。
- `talk.txt` 只在 Bot 啟動時載入，修改後需要重新啟動。
- 新增 action 時必須同步更新 `main.go` 的 action 常數、驗證條件及 `executeTalkAction()`。

## 建置與檢查

格式化及確認 Go 專案可編譯：

```bash
gofmt -w main.go weather.go stock.go
go mod tidy
go test ./...
go vet ./...
```

確認 Python 語法：

```bash
python3 -m py_compile stock.py
```

建置 Go 執行檔：

```bash
go build .
```

`.gitignore` 已排除 Windows `.exe`、`.env` 與 `.env.*`。

## 目前限制

- `talk.txt` 不會在執行期間自動重新載入。
- 天氣與股票功能依賴外部服務，可能受到網路、服務狀態及資料提供時間影響。
- 每次股票查詢都會啟動新的 Python 程序，沒有快取或同時請求合併。
- 證交所休市時，單日大盤會向前尋找最近十天內可用的交易資料。
- Discord 訊息傳送錯誤目前未集中記錄。

## 相關文件

- `.agent/History.md`：開發與架構變更歷程。
- `.agent/chat.md`：架構決策、限制與歷史討論。
- `knowhow.md`：部署與維護經驗。
- `DiscordGo.md`：DiscordGo 系統參數備忘。

## 參考資料

- [DiscordGo](https://github.com/bwmarrin/discordgo)
- [中央氣象署開放資料平臺](https://opendata.cwa.gov.tw/)
- [臺灣證券交易所](https://www.twse.com.tw/)
- [lunar-go](https://github.com/6tail/lunar-go)

### 參考資料（較舊）

- [使用 Golang 打造 Discord 機器人 (二)](https://tw.coderbridge.com/series/0d06c0381803425290e745a4ead229a9/posts/c19eeab1839a4cd68a43ef844296b83b)
- [Discord Bot in Golang](https://youtu.be/myCtjnjV5YU)
- [🔴Building a Discord Bot with Go!](https://youtu.be/N8L1kPxxTJM)# discord-bot
