# DiscordBot
Golang版本
## 參考資料
- [使用 Golang 打造 Discord 機器人 (二)](https://tw.coderbridge.com/series/0d06c0381803425290e745a4ead229a9/posts/c19eeab1839a4cd68a43ef844296b83b)
- [Discord Bot in Golang](https://youtu.be/myCtjnjV5YU)
- [🔴 Building a Discord Bot with Go!](https://youtu.be/N8L1kPxxTJM)# discord-bot
## 專案簡介

這是一個使用Go開發的 Discord 機器人。
程式採用事件驅動架構，啟動後會監聽伺服器中的文字訊息，並依照訊息內容回覆或執行對應功能。
小助手是Codex。

## 開發環境

- Go 1.19
- [discordgo](https://github.com/bwmarrin/discordgo) v0.26.1
- [godotenv](https://github.com/joho/godotenv) v1.4.0
- [lunar-go](https://github.com/6tail/lunar-go) v1.4.6：農曆與國曆日期換算
- [go-cwb](https://github.com/minchao/go-cwb)：串接中央氣象署開放資料 API


## 專案架構
```text
discord-bot/
├── main.go      #主程式、Discord 事件處理與功能型指令
├── talk.txt     #可擴充的文字對話規則
├── start.sh     #Linux CLI 啟動檔
├── start.bat    #Windows CLI 啟動檔
├── go.mod       #Go 模組與直接相依套件
├── go.sum       #相依套件版本校驗資訊
└── Readme.md    #專案說明
```

## 設定方式（使用Go環境）

1. 在 [Discord Developer Portal](https://discord.com/developers/applications) 建立應用程式及 Bot。
2. 將 Bot 加入要使用的 Discord 伺服器，並授予檢視頻道、讀取訊息與傳送訊息所需權限。
3. 在專案根目錄建立 `.env`，內容如下：

```dotenv
DCToken=你的_Discord_Bot_Token
```

請勿將 Bot Token 提交至版本控制；本專案的 `.gitignore` 已排除 `.env` 與 `.env.*`。

## 各系統環境的啟動流程
[參照該檔案](/other.md)

## 安裝與執行

下載相依套件：

```bash
go mod download
```

啟動機器人：

```bash
go run .
```

終端機出現以下訊息時，代表機器人已開始執行：

```text
Bot is now running.  Press CTRL-C to exit.
```
停止機器人可按下 `Ctrl+C`。

## 建置

```bash
go build .
```
建置完成後會在專案目錄產生可執行檔；`.gitignore` 已排除 Windows 的 `.exe` 檔案。

## 目前功能

| 輸入訊息 | 機器人行為 |
| --- | --- |
| `早安` | 早安！今天也要保持好心情。 | 
| `午安` | 午安！記得吃午餐。 | 
| `晚安` | 晚安，記得讓電腦和自己都休息一下。 | 
| `謝謝` | 不客氣，很高興能幫上忙！ | 
| `幫助` | 你可以試試看：九九乘法、算命、現在時間。 | 
| `說` | 回覆提示訊息 |
| `九九乘法` | 顯示九九乘法表 |
| 包含 `笨蛋` | 回覆預設訊息 |
| 包含 `肚子餓` | 回覆預設訊息 |
| `算命` | 隨機回覆一則算命結果 |
| `現在時間` | 回覆執行機器人所在環境的目前時間 |
| `瑪麗亞凱莉解凍` | 距離耶誕節的倒數時間 |
| `劉德華解凍` | 距離農曆新年的倒數時間 |
| `天氣` | 最近36小時的天氣 |
| `下雨` | 最近36小時的降雨機率 |

## 對話規則維護

一般文字對話已從 `messageCreate()` 移至 `talk.txt`。新增或修改對話時，不需要再修改 Go 程式碼，只要在 `talk.txt` 中維護規則即可。

每一筆規則包含以下三個欄位，欄位之間必須使用半形減號 `-` 分隔：

```text
比對方式-觸發文字-回覆內容
```
支援的比對方式如下：

| 比對方式 | 說明 | 範例行為 |
| --- | --- | --- |
| `exact` | Discord 訊息必須與觸發文字完全相同 | `ping` 只會符合完整內容為 `ping` 的訊息 |
| `contains` | Discord 訊息中包含觸發文字即可 | `你是笨蛋` 會符合觸發文字 `笨蛋` |

新增規則範例：

```text
exact-你好-你好，很高興見到你！
contains-晚安-晚安，祝你有個好夢！
```
分隔符號必須使用半形減號 `-`，不可使用全形破折號、Tab或空格。解析器只會切割前兩個減號，因此第三欄的回覆內容仍可包含減號。

維護規則時請注意：

- 空白行會被忽略。
- 以 `#` 開頭的內容為註解，不會成為對話規則。
- 可以持續加入任意筆數的規則。
- 同一則訊息符合多筆規則時，機器人會依照檔案順序逐一回覆。
- `talk.txt` 會在程式啟動時載入，修改後需要重新啟動機器人。
- `九九乘法`、`算命`、`現在時間` 等功能型指令仍由 `main.go` 處理。

如果規則的欄位數量錯誤、比對方式不受支援，或觸發文字及回覆內容為空白，程式會顯示錯誤並停止啟動，以便修正有問題的規則。
