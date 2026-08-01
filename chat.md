# Discord Bot 深度交流紀錄

## 文件用途

這份文件用來承接 Phoenix 與 Codex 對專案架構、功能方向、取捨與風險的深入討論。

- `Readme.md` 面向使用者，說明已經可以使用的功能與操作方式。
- `History.md` 保存已完成工作、排錯紀錄，以及清楚標記的設計階段事項。
- `chat.md` 保存仍需要討論或選擇的方向，不代表內容已經實作。
- 當討論項目確定並完成後，再將結果同步到程式、`History.md` 與必要的操作文件。


## 目前結論

專案已經具備可運作的文字訊息 Bot、外部對話規則、節慶日期計算、中央氣象署 36 小時預報、降雨口語回覆、Discord Presence，以及 Windows、Debian 啟動檔。

Phoenix 已決定不採用 `/天氣`、Discord 城市輸入、Guild Command 與城市別名。天氣與降雨維持原本的文字 action，查詢城市繼續由 `.env` 的 `CWA_LOCATION` 管理。

Linux 部署則由原本的 `nohup` 與 PID 檔改為 systemd：`start.sh` 只負責編譯並以前景 `exec` 執行 Bot，systemd 負責開機啟動、停止、異常重啟、程序狀態與 journal 日誌。
---
答：`/天氣`這個我打算棄用，推測需要的是特殊插件，所以不用了。目前還是先透過[env](/.env)更新就好。利用指令介面操作`start.sh`的方式確實是個好手段，可以再討論。

---

## 目前架構盤點

| 檔案 | 現有責任 | 觀察 |
| --- | --- | --- |
| `main.go` | Discord Session、文字訊息事件、對話規則載入、action 分派、日期與其他功能 | 不掛載 Interaction；現在時間已納入 action 分派 |
| `weather.go` | CWA API 查詢、預報資料整併、天氣與降雨回覆 | 仍使用 `CWA_LOCATION`，未設定時預設臺北市 |
| `talk.txt` | `exact`、`contains`、`action` 規則 | 容易擴充，但啟動後不會自動重新載入 |
| `start.sh` | Debian 編譯後以前景 `exec` 啟動 Bot | 由 systemd 管理停止、重啟、狀態與日誌 |
| `discord-bot.service` | OMV／Debian systemd Service | 使用目前 `/root/discord-bot` 部署路徑 |
| `start.bat` | Windows 背景啟動並等待 Go 程式結束 | 功能簡單，適合開發使用 |
| `Readme.md` | 專案簡介、功能與基礎使用方式 | 已同步文字 action、環境設定與 systemd 架構 |
| `other.md` | Windows 與 OMV／Debian 詳細操作 | Debian 已改用 `systemctl` 與 `journalctl` |
| `History.md` | 開發、排錯、驗證與規劃紀錄 | 已區分目前限制與未來方向 |
| `DiscordGo.md` | Presence 活動型別速查 | 內容精簡，可保留為開發備忘 |
| `go.mod`、`go.sum` | Go 版本與相依套件 | 目前使用 Go 1.19 與 DiscordGo v0.26.1 |
| `.gitignore` | 排除 `.env`、`.env.*`、Windows 執行檔與暫存檔 | `.env` 已排除，仍需避免任何日誌輸出密鑰 |

## 現有訊息資料流

```text
Discord 文字訊息
    ↓
messageCreate()
    ├── talk.txt：exact／contains
    ├── talk.txt：action
    │       ↓
    │   executeTalkAction()
    │       ├── 耶誕節倒數
    │       ├── 農曆新年倒數
    │       ├── 36 小時天氣
    │       └── 降雨機率
    └── main.go 既有功能：九九乘法／算命
```

這個流程適合沒有參數的文字觸發，但不適合 `/天氣 臺中市` 這類帶參數、需要 Discord 原生輸入欄位的互動。若強行把所有參數都塞進 `talk.txt`，規則解析器會逐漸承擔不屬於它的命令列解析責任。
---


## 已否決方案：Discord 天氣互動資料流

本節保留最初的設計推演。Phoenix 已於第 317 行後決定不採用 `/天氣`、城市輸入、Guild Command 與城市別名，因此本節不再是待實作項目。

```text
Discord /天氣
    ↓
必填欄位：城市名稱
    ↓
InteractionCreate Handler
    ↓
立即送出 Deferred Response
    ↓
解析與正規化城市名稱
    ↓
loadWeatherPeriods(ctx, locationName)
    ↓
go-cwb／中央氣象署 API
    ↓
Discord Embed 欄位表格
```

### 城市輸入原則

- 程式不應保留 `defaultWeatherLocation = "臺北市"`。
- 城市名稱的唯一業務輸入來源應是 Discord Interaction Options。
- `loadWeatherPeriods()` 應明確要求 `locationName string`，空字串直接回傳輸入錯誤。
- 可將使用者輸入的「台」轉為「臺」，並移除前後空白。
- 不建議用固定字串陣列限制所有城市，避免每次資料名稱變更都要更新程式。
- `go-cwb` 的方法要求 `[]string` 時，只在 API 邊界建立 `[]string{locationName}`；這是型別轉換，不是城市清單。
- 若 API 回傳的 `Records.Location` 為空，應回覆找不到該縣市，而不是回退到預設城市。
- 鄉鎮、市區與縣市層級需要明確區分；目前使用的 F-C0032-001 以縣市預報為主。

### `/天氣` 與「下雨」的關係

目前有三種可選方向：

| 方案 | 操作方式 | 優點 | 缺點 |
| --- | --- | --- | --- |
| A | `/天氣 城市名稱`，結果同時包含降雨機率 | 操作最簡單，符合目前需求 | 原本「下雨」口語功能入口不明顯 |
| B | `/天氣 城市名稱 模式`，模式可選完整預報或降雨建議 | 單一指令可擴充，保留口語回覆 | 多一個選項，介面稍複雜 |
| C | `/天氣 城市名稱` 與 `/下雨 城市名稱` | 功能分工直覺 | 需要註冊與維護兩個指令 |

我的建議是先採方案 A，讓 `/天氣` 表格包含降雨機率；下一階段再加入選用的「模式」欄位。舊的文字 `天氣` 與 `下雨` 可以暫時回覆使用說明，引導使用者改用斜線指令，而不是繼續查詢預設城市。

### Discord 回覆形式

Discord 沒有原生 Markdown 表格，直接用空格排版會受到中文字寬與行動版換行影響。較穩定的做法是：

- Embed 標題顯示實際查詢縣市。
- 每個預報時段建立一個 Inline Field。
- Field 內依序顯示天氣、降雨、最低至最高溫及舒適度。
- Footer 標示中央氣象署資料集來源。
- API 或輸入錯誤改用簡短訊息，不將底層請求內容直接回覆給 Discord 使用者。

### Interaction 時間限制

Discord Interaction 必須很快收到確認，但外部 API 不保證能在限制內回應。因此建議：

1. Handler 先送出 `InteractionResponseDeferredChannelMessageWithSource`。
2. 使用 goroutine 執行天氣查詢，避免阻塞其他事件處理。
3. 使用 `context.WithTimeout` 限制外部請求時間。
4. 查詢完成後呼叫 `InteractionResponseEdit` 更新原始訊息。
5. 查詢失敗時只顯示安全的使用者訊息，完整錯誤寫入已遮蔽敏感資料的日誌。

### 斜線指令的註冊範圍

開發階段可使用指定伺服器的 Guild Command，更新通常較容易立即看到；正式部署再改為 Global Command。可以在 `.env` 增加選用的 Guild ID：

```dotenv
DiscordGuildID=開發伺服器_ID
```

若有 `DiscordGuildID` 就註冊 Guild Command，沒有則註冊 Global Command。這能兼顧開發速度與正式發布，但需要先由 Phoenix 決定是否接受新增環境變數。

## 原問題：`start.sh` 無法停止

調整前的腳本只負責啟動，停止功能是印出：

```bash
kill <PID>
```

這個方式可以傳送訊號，但不構成完整的程序管理流程：

- 不會驗證 PID 檔內容是否仍屬於 Bot。
- 不會在停止後主動清除 PID 檔。
- 不會等待 `main.go` 完成 Discord Session 關閉。
- 不會提供一致的 status 或 restart 操作。
- 使用者若直接使用過期 PID，可能對錯誤程序送出訊號。

### 建議的腳本操作契約

```bash
bash start.sh start
bash start.sh stop
bash start.sh restart
bash start.sh status
```

- 未提供參數時等同 `start`，保留現有習慣。
- `start` 先驗證 PID 與 `/proc/<PID>/exe`，確認不是重複啟動。
- `stop` 先送 `SIGTERM`，讓 Go 程式執行 `dg.Close()`。
- `stop` 等待一段合理時間，再次確認同一 PID 仍是 Bot，最後才考慮 `SIGKILL`。
- `restart` 必須確認舊程序已停止，才重新編譯及啟動。
- `status` 應同時顯示 PID、執行狀態與日誌位置。
- PID 無效或程序已結束時，腳本應安全清除舊 PID 檔。

### 長期部署仍建議 systemd

即使補齊上述子指令，`nohup` 仍不會提供：

- 開機自動啟動。
- 程序異常後自動重啟。
- 啟動順序及網路相依管理。
- 統一的日誌查閱與輪替。
- 服務權限及資源限制。

OMV 長期運作可在腳本穩定後建立 `systemd` Service。屆時 `ExecStart` 應直接執行編譯完成的 Bot，而不是在服務內呼叫會自行背景化的 `nohup` 腳本。

## 程式架構建議

### 1. 將 action 白名單與分派集中

目前新增 action 時需要同時修改常數、`loadTalkRules()` 白名單及 `executeTalkAction()` switch，容易漏掉其中一處。可以改為 action registry：

```go
type talkAction func(*discordgo.Session, string)

var talkActions = map[string]talkAction{
    "christmas_countdown": sendChristmasCountdown,
    "lunar_new_year":      sendNextLunarNewYear,
}
```

載入規則與執行 action 都查詢同一張 map，能減少重複條件。帶有城市參數的斜線指令則維持獨立 Interaction Handler，不必勉強放入這張文字 action 表。

### 2. 集中環境設定驗證

目前 `DCToken` 與 `CWA_API` 分散在不同函式讀取。可以建立設定結構，在啟動階段一次驗證：

```go
type config struct {
    DiscordToken string
    CWAAPIKey    string
    GuildID      string
}
```

這樣能在啟動時明確指出缺少哪一個必要設定，也能避免每次查詢都重新讀取環境變數。任何錯誤輸出都不得包含實際 Token 或 API Key。

### 3. 讓 CWA Client 可測試

目前 `get36HourWeather()` 直接建立 Client，測試時很難替換外部 API。可將 CWA 查詢包成小型介面，正式環境使用 go-cwb，測試環境使用固定資料：

- 不需要真實 API Key。
- 不消耗 API 額度。
- 可以穩定測試無效城市、空資料、逾時及 API 錯誤。
- 可以檢查三個預報時段是否正確合併及排序。

### 4. 統一 Discord 傳送錯誤

目前多數 `ChannelMessageSend()` 沒有檢查錯誤。建議建立小型傳送函式，至少記錄頻道 ID、功能名稱與錯誤類型；不可記錄私密訊息全文或密鑰。

### 5. 明確處理隨機字庫

目前使用全域 `math/rand` 並在啟動時 Seed。這對一般口語選句足夠，但測試結果不容易重現。可以讓口語選句函式接收亂數來源，測試時使用固定 Seed，正式執行時再使用時間 Seed。

### 6. 檢視 Go 與相依套件升級

目前模組宣告 Go 1.19。升級前應先確認 OMV 上可安裝的 Go 版本、DiscordGo 與 go-cwb 的相容性，再以獨立分支執行測試；不建議只修改 `go.mod` 的版本數字而沒有更新部署環境。

## 測試與驗收建議

| 類別 | 測試案例 | 預期結果 |
| --- | --- | --- |
| 對話規則 | 正確、缺欄位、未知 match type、未知 action | 正確載入或指出檔名與行號 |
| 城市輸入 | `臺中市`、`台中市`、空白、不存在名稱 | 正規化、拒絕空白、清楚回覆無資料 |
| 天氣資料 | 三個完整時段、缺少 PoP、欄位順序不同 | 正確合併、排序並顯示未知值 |
| Discord Interaction | 正常、API 逾時、API 錯誤 | 先 deferred，再更新成功或錯誤訊息 |
| Linux 啟動 | 首次 start、重複 start、失效 PID | 正常啟動、不重複、清除失效 PID |
| Linux 停止 | stop、重複 stop、TERM 逾時 | 正常關閉、冪等處理、必要時安全強制停止 |
| 文件 | README 指令與實際程式比對 | 不把規劃中功能寫成已完成 |

Go 程式至少應執行：

```bash
gofmt -w main.go weather.go
go test ./...
go vet ./...
```

Debian 腳本至少應執行：

```bash
bash -n start.sh
shellcheck start.sh
```

若開發環境沒有 Bash，必須在 OMV、WSL 或 CI 補做腳本語法及實際程序生命週期測試，不能只依靠 Windows 靜態閱讀。

## README 原始更新建議（本次已處理）

目前建議修改 `Readme.md`，但這次依照 Phoenix 的要求先不直接變更。需要更新的內容如下：

1. 專案架構加入 `weather.go`、`History.md`、`other.md` 與 `DiscordGo.md`。
2. `.env` 範例加入 `CWA_API`；在目前版本仍使用預設地區時，也應說明選用的 `CWA_LOCATION`。
3. 對話規則支援類型補上 `action`，並說明第三欄必須是程式支援的 action 名稱。
4. 目前功能表可將「即時訊息」改成更明確的「動態 action」，避免與 Discord Interaction 混淆。
5. 若完成 `/天氣`，新增斜線指令的城市名稱欄位、權限、使用範例及全域指令同步時間說明。
6. 若完成新版 `start.sh`，將 Debian 啟動、停止、重啟與狀態查詢指令同步到 README 或導向更新後的 `other.md`。
7. 將 `CWA_LOCATION` 的說明與最終決策同步：保留為備援，或在城市完全由 Discord 輸入後移除。

## 原始建議實作順序（已由 Phoenix 的答案取代）

### 第一階段：先讓版本描述一致

- Phoenix 決定是否以目前工作樹重新實作 `/天氣` 與新版 `start.sh`。（）
- 完成後立即執行 Go 與 Bash 驗證。
- 根據實際結果更新 README、`other.md` 與 History，不提前宣告完成。

### 第二階段：完成 Discord 城市輸入

- 註冊 Interaction Handler。
- 建立 `/天氣` 與必填的「城市名稱」選項。
- 移除程式內預設城市查詢路徑。
- 建立 deferred response、API timeout 與 Embed 回覆。
- 決定舊文字 `天氣`、`下雨` 的相容策略。

### 第三階段：完成 Linux 程序管理

- 加入 `start|stop|restart|status`。
- 驗證 PID 與執行檔路徑。
- 測試 SIGTERM 正常關閉及逾時處理。
- 更新 `other.md`，再評估是否建立 systemd Service。

### 第四階段：可維護性與測試

- 集中 config 與 action registry。
- 將 CWA API 包裝成可替換介面。
- 為規則解析、天氣資料整理及城市輸入增加單元測試。
- 視 API 使用量再加入快取、請求合併及重試，不在需求尚未穩定時過早增加狀態管理。

## Phoenix 已回答的決策題目

1. `/天氣` 要只提供完整預報，還是增加「模式」欄位切換完整預報與口語降雨建議？
2. 舊文字 `天氣`、`下雨` 要保留預設城市、引導改用斜線指令，還是完全移除？
3. 開發期間是否接受新增 `DiscordGuildID`，以 Guild Command 加快斜線指令更新？
4. 城市輸入是否只接受中央氣象署正式縣市名稱，還是要支援 `台中` 自動補成 `臺中市` 等別名？
5. `start.sh` 完成子指令後，下一步是否直接建立 OMV 使用的 systemd Service？
6. README 是否要在下一次修改中，同步處理本文件列出的七項落差？

以上問題不需要一次全部決定。建議先回答第 1、2、3 項，因為它們會直接決定 `/天氣` 的程式介面與相容策略。

## 回答：
1.  答：不要用`/天氣`，未持原本狀態。
2.  答：不要加入斜線指令，維持原本的`天氣`、`下雨`，目前維持存放於`ENV`的方式。
3.  答：不要。
4.  答：不要。
5.  答：好，[Shell](/start.sh)的程式碼，基於精簡化來調整`*.sh`成新版的的`systemd`執行環境。
6.  答：[Readme](/Readme.md)和[main.go](/main.go)一併處理，將架構改成以下這樣，
```text
Discord 文字訊息
    ↓
messageCreate()
    ├── talk.txt：exact／contains
    ├── talk.txt：action
    │       ↓
    │   executeTalkAction()
    │       ├── 耶誕節倒數
    │       ├── 農曆新年倒數
    │       ├── 36 小時天氣
    │       ├── 降雨機率
    │       └── 現在時間
    └── main.go
```

## Codex 的回答：2026-08-01 執行結果

1. 不實作 `/天氣` 斜線指令，`main.go` 不加入 Interaction Handler。
2. 保留文字指令 `天氣` 與 `下雨`，並繼續從 `.env` 的 `CWA_LOCATION` 取得查詢縣市。
3. 不新增 `DiscordGuildID`，也不註冊 Guild Command。
4. 不加入 `台中` 自動補成 `臺中市` 等城市別名。
5. `start.sh` 已精簡為 systemd 前景入口，並新增 `discord-bot.service`；服務管理改用 `systemctl`，日誌改用 `journalctl`。
6. `現在時間` 已從 `messageCreate()` 的直接 switch 移至 `talk.txt` 的 `action-現在時間-local_time`，並由 `executeTalkAction()` 分派。
7. `Readme.md` 已同步專案樹、訊息架構、`.env` 天氣設定、action 規則與 systemd 說明。
8. `other.md` 已移除舊版 `nohup`、PID 檔及手動 `kill` 的主要操作流程，改為 OMV systemd 安裝、啟停、狀態與日誌指令。

補充說明：Discord 斜線指令本身不需要額外插件，DiscordGo 已提供 Application Command API；本次不實作是依照 Phoenix 決定維持文字 action 的產品方向，而不是受到套件能力限制。

### 本次採用的 systemd 理念

- Shell 腳本不自行背景化，避免 systemd 只追蹤到外層 Shell 或失去真正的 Bot PID。
- `exec` 讓 Go Bot 取代 Shell 程序，systemd 可直接傳送 `SIGTERM` 給 `main.go`。
- `main.go` 已監聽 `SIGTERM`，因此停止 Service 時會正常執行 `dg.Close()`。
- 編譯檔放在 `/var/lib/discord-bot`，不寫入 Git 專案目錄。
- `Restart=on-failure` 處理異常結束；人工執行 `systemctl stop` 不會觸發重啟。
- 標準輸出與錯誤交由 journal 保存，不再維護自訂日誌檔及 PID 檔。
---

## Codex 同事回覆：Discord 系統參數盤點

1. 答：我已將 `DiscordGo.md` 從原本只有 Activity 狀態的簡表，擴充成 Bot 開發用的系統參數速查，並保留原狀態表，不刪除既有內容。
2. 答：內容以目前專案鎖定的 `discordgo v0.26.1` 為實作基準，另外標出 Discord 現行官方規格，避免把新版 API 常數誤當成目前套件可直接使用。
3. 答：本次整理範圍包括 Gateway Intents、頻道類型、權限位元、Application Command、Interaction、訊息元件、Message Flags、訊息限制、Webhook、OAuth2 Scope 及重要時限。
4. 答：沒有修改 `main.go`、`go.mod`、`Readme.md` 或其他程式檔；這一輪只整理文件與提出下一步需要共同決定的事項。

## 想和 Phoenix 討論：

1. 問：目前 `main.go` 只啟用 `IntentsGuildMessages`，但 `messageCreate()` 需要讀取 `m.Content` 才能比對 `talk.txt`。依官方規格，伺服器的一般訊息內容還需要 `MESSAGE_CONTENT` 特權 Intent。我建議下一次將程式改為同時啟用 `IntentGuildMessages | IntentMessageContent`，並在 Discord Developer Portal 開啟 Message Content Intent。這會直接影響現有文字指令是否能穩定運作，你是否同意列為下一個修正項目？
2. 問：`discordgo v0.26.1` 與現行 Discord API 已有明顯版本差距，例如 Forum／Media Channel、投票 Intent、新權限、Components V2 與新的 Message Flags 都沒有完整具名常數。我傾向先維持目前版本，等真的要使用其中一項功能時，再安排一次套件升級及 OMV（Debian）相容性測試，避免為了文件更新而改動正式執行環境。你是否也希望採取這個節奏？
3. 問：依你先前的決定，天氣功能繼續採用 `talk.txt` 的文字 action，不加入 `/天氣`。因此 Application Command 與 Interaction 章節目前定位為技術備查，不代表要更改既有產品方向；如果之後沒有斜線指令需求，邀請 Bot 時也不必特別加入 `applications.commands` Scope。
4. 問：目前 Bot 的核心需求是讀取訊息並回覆文字。邀請權限建議維持最小化，通常只給 `VIEW_CHANNEL`、`SEND_MESSAGES`、`READ_MESSAGE_HISTORY`，使用 Embed 或附件時才加對應權限，不授予 `ADMINISTRATOR`。若你同意，日後整理部署文件時可以補上一組可重建的最小權限邀請設定。
