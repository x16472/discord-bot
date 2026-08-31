# 開發歷程

## 2026-07-31：專案文件與基礎架構整理

- 盤點既有 Go Discord Bot 的單檔事件驅動架構，確認主要進入點為 `main.go`。
- 整理既有文字指令，包括問候、九九乘法、算命、現在時間及關鍵字回覆。
- 擴充 `Readme.md`，補上專案簡介、功能表、目錄結構、環境變數、安裝、執行及建置方式。
- 保留原有參考資料與既有內容，以增補方式完成第一版專案文件。
- 確認專案使用 Go 1.19、`discordgo` 與 `godotenv`。

## 2026-08-01：對話規則外部化

- 將一般文字對話從 `messageCreate()` 移至 `talk.txt`，讓新增對話不必直接修改 Go 程式。
- 建立 `talkRule` 結構與 `loadTalkRules()`，於 Bot 啟動時讀取規則。
- 支援三種規則型別：
  - `exact`：Discord 訊息必須完全符合觸發文字。
  - `contains`：Discord 訊息只要包含觸發文字即可。
  - `action`：將第三欄作為 Go 動態功能識別名稱。
- 初期使用 Tab 分隔欄位，後續因跨平台編輯時容易被轉成空格，統一改為半形減號格式：`比對方式-觸發文字-回覆內容`。
- 解析器改用 `strings.SplitN(line, "-", 3)`，只切割前兩個減號，使第三欄回覆內容仍能包含減號。
- 加入空白行、註解行、欄位數、空白欄位、規則型別及 action 白名單驗證。
- 增加早安、午安、晚安、謝謝、幫助及其他測試對話，確認規則可持續擴充。

## 2026-08-01：日期與節慶動態指令

- 新增「瑪麗亞凱莉解凍」指令，使用 `time.Now()` 計算距離下一個耶誕節的日曆天數。
- 新增「劉德華解凍」指令，以當前年份推算明年農曆正月初一所對應的國曆日期與剩餘天數。
- 引入 `github.com/6tail/lunar-go` v1.4.6，處理農曆與國曆日期換算。
- 建立 `calendarDaysBetween()`，以 UTC 日期零點計算日曆天數，避免日光節約時間造成誤差。
- 使用 action 名稱連接 `talk.txt` 與 `executeTalkAction()`，並在啟動時驗證動態功能是否受到支援。
- 使用 `rand.Seed(time.Now().UnixNano())` 初始化 Go 1.19 的亂數序列，避免算命及口語字庫在每次重新啟動後出現相同順序。

## 2026-08-01：Windows 與 Debian 啟動流程

- 建立 `start.bat`，讓 Windows 使用 `start /b /wait` 執行 `go run .` 並保留結束代碼。
- 建立 `start.sh`，初期採用背景程序加 `wait`，讓 Linux 能接收停止訊號並清理編譯檔。
- 排查 OMV 啟動時的 `talk.txt line ... separated by tabs` 錯誤，確認主因可能是：
  - `talk.txt` 分隔符號錯誤。
  - Windows 與 Linux 換行格式不同。
  - Debian 上的 `main.go`、`talk.txt` 與 `start.sh` 版本沒有同步。
- 將 `start.sh` 精簡為真正的背景啟動流程：
  - 使用 `go build` 建立執行檔。
  - 使用 `nohup` 脫離 SSH 終端。
  - 將標準輸出與錯誤輸出寫入日誌。
  - 將 PID 寫入 PID 檔。
  - 啟動後等待一秒，檢查 Bot 是否立即失敗。
  - 使用 `/proc/<PID>/exe` 避免將重複使用的舊 PID 誤認為 Bot。
- 將 Linux 執行檔、PID 與日誌集中於 `~/.local/state/discord-bot/`，避免在 Git 專案內產生執行期檔案。
- 在 `other.md` 補上背景程序查詢、日誌查看與停止方式，包括：
  - `ps aux | grep discord-bot`
  - `ps aux | grep '[d]iscord-bot'`
  - `pgrep -af 'discord-bot'`
  - 透過 PID 檔精確查詢及停止程序。
- 補充不應直接使用 `kill -9`、`nohup` 不提供自動重啟，以及長期運作建議改用 `systemd` 等注意事項。

## 2026-08-01：OMV 與跨平台文件

- 重整 `Readme.md` 的 Windows 與 OMV（Debian）啟動章節。
- 補上 Go 版本、`.env`、Token、檔案權限、CRLF／LF、時區及部署版本同步檢查。
- 說明 OMV 主機的 `time.Now()` 會受到系統時區影響，並提供 `timedatectl` 檢查方式。
- 整理常見錯誤對照表，包括舊版 Tab 錯誤、權限、Token、Discord 頻道權限及 SSH 中斷問題。
- 在開發環境章節加入 `lunar-go` 與 `go-cwb` 的版本及用途。

## 2026-08-01：中央氣象署天氣功能

- 在 `.env` 使用 `CWA_API` 保存中央氣象署 API Key，程式只讀取變數名稱及內容，不將金鑰輸出至日誌。
- 引入 `github.com/minchao/go-cwb/cwb`，使用資料集 `F-C0032-001` 取得今明 36 小時天氣預報。
- 修正將 API Key 字串直接當成 Client 使用的錯誤，改以 `cwb.NewClient(apiKey, nil)` 建立 CWA Client。
- 新增 `weather.go`，將天氣查詢、資料整理、Discord 回覆及降雨口語字庫從 `main.go` 分離。
- 建立 `get36HourWeather()`，查詢指定縣市的 `Wx`、`PoP`、`CI`、`MinT` 與 `MaxT`。
- 建立 `configuredWeatherLocation()`，優先讀取 `CWA_LOCATION`，未設定時使用臺北市。
- 建立 `loadWeatherPeriods()`，以開始時間與結束時間作為索引，合併不同順序的氣象因子。
- 建立 10 秒 `context` 逾時，避免中央氣象署 API 無回應時長時間占用 Discord 訊息處理。
- 建立 36 小時天氣 action：
  - 目前觸發規則為 `action-天氣-weather_36h`。
  - 回覆各預報時段、天氣現象、降雨機率、最低與最高溫度及舒適度。
  - 在訊息末尾標示資料來源為中央氣象署 `F-C0032-001`。
- 將原本「下雨」的靜態回覆改為動態降雨機率 action：
  - 目前觸發規則為 `action-下雨-rainProbability_action`。
  - 找出未來 36 小時的最高降雨機率與對應時段。
  - 同時列出各時段降雨機率。
  - 依降雨機率使用 `switch-case` 分成六個區間。
  - 每個區間提供多句口語字庫，再隨機選取一句，使相同機率也能產生不同回覆。
- 排查 `unsupported action "weather_36h"`，同步更新 action 常數、白名單與 `executeTalkAction()` 分派。
- 將 `go-cwb` 調整為直接相依套件，並補齊 `go-querystring` 間接相依與 `go.sum`。

## 2026-08-01：Discord Presence 與遊戲狀態

- 排查 `error updating game status, no websocket connection exists`，確認 `UpdateGameStatus()` 在 `dg.Open()` 前呼叫，WebSocket 尚未建立。
- 將 Presence 更新移至 `dg.Open()` 成功之後，確保狀態能透過 Discord Gateway 傳送。
- 由單純的 `UpdateGameStatus()` 改為 `UpdateStatusComplex()`，可設定活動名稱、活動型別、狀態及開始時間。
- 曾以 `ForzaHorizon6` 測試「正在遊玩」效果，後續調整為目前的自訂活動內容。
- 釐清 Discord Presence 是 Bot 主動宣告的狀態，不代表主機真的執行該遊戲或程式，活動名稱可以自行設定。
- 確認 Bot 透過 Gateway 能可靠設定的 Activity 欄位以 `name`、`state`、`type`、`url` 為主；`details`、圖片與時間等 Rich Presence 欄位可能被 Discord 客戶端忽略。
- 整理 Activity 型別用途：Playing、Streaming、Listening、Watching、Custom 與 Competing。

## 錯誤排查與驗證紀錄

- 修正 `forecast` 宣告但未使用，以及將 `string` 當成 CWA Client 呼叫 `Forecasts` 的編譯錯誤。
- 修正 `talk.txt` Tab、空格與半形減號混用造成的載入失敗。
- 修正 Linux 部署時只同步 `talk.txt`、未同步新版 `main.go` 所造成的新舊格式不相容。
- 修正動態 action 已寫入 `talk.txt`，但未加入程式白名單與分派 switch 的問題。
- 多次執行 `gofmt`、`go test ./...`、`go vet ./...`、Bash 語法檢查及 `git diff --check`。
- 使用獨立 Go build cache 處理本機系統快取路徑衝突，避免修改專案範圍外的既有快取。
- 天氣功能的編譯與靜態檢查已完成；基於 API Key 安全及避免未授權外部請求，開發期間未使用 `.env` 金鑰執行真實 API 連線測試。

## 目前限制與後續方向

- `talk.txt` 於 Bot 啟動時載入，修改規則後需要重新啟動。
- 目前天氣查詢每次都會呼叫中央氣象署 API，尚未加入快取、同時請求合併與 API 重試機制。
- 天氣指令預設使用臺北市；正式部署可在 `.env` 設定 `CWA_LOCATION`。
- Discord 訊息傳送錯誤多數仍沿用既有忽略方式，後續可統一加入日誌與錯誤處理。
- `nohup` 適合基本背景執行，但不提供開機啟動、異常自動重啟或日誌輪替；OMV 長期部署仍建議建立 `systemd` 服務。
- Discord Bot Presence 屬於自訂宣告，不具備偵測真實遊戲程序的能力；完整 Rich Presence 需由應用程式整合 Discord SDK。

## 後續規劃：由使用者指定天氣縣市

- 將天氣指令由固定讀取 `CWA_LOCATION`，調整為接受使用者輸入的縣市名稱，例如 `天氣 臺中市` 與 `下雨 高雄市`。
- 使用者只輸入 `天氣` 或 `下雨` 而未提供縣市時，先回覆正確用法與範例；若日後仍需要預設地區，再考慮將 `CWA_LOCATION` 保留為選用的備援設定。
- 現有 `talk.txt` 的 `action` 採完整字串比對，無法直接解析指令後方的縣市參數；後續可在 `messageCreate()` 增加前綴指令解析，或擴充一種可攜帶參數的對話規則類型。
- 指令解析可使用 `strings.Fields()` 分離指令與參數，再將縣市名稱去除前後空白，避免使用者輸入多餘空格時查詢失敗。
- 天氣查詢函式對外維持接收單一 `locationName string`，例如將函式調整為 `loadWeatherPeriods(ctx, locationName)`、`send36HourWeather(s, channelID, locationName)` 與 `rainProbabilityAction(s, channelID, locationName)`。
- `go-cwb` 的 `Get36HourWeather()` 方法本身要求 `[]string`，因此僅在呼叫 API 的邊界使用 `[]string{locationName}`；這是套件介面的必要轉換，不代表程式需要維護固定的城市字串陣列。
- 不建立可選城市的硬編碼字串陣列，直接將使用者提供的縣市名稱交由中央氣象署 API 查詢，再依回傳結果判斷是否為有效地點。
- 可使用別名對照表處理常見寫法，例如將 `台北` 正規化為 `臺北市`、`台中` 正規化為 `臺中市`；對照表只負責名稱正規化，不作為限制使用者輸入的城市清單。
- 36 小時預報資料以縣市層級為主，若使用者輸入鄉鎮、市區或不存在的名稱，應回覆目前僅支援中央氣象署資料中的縣市名稱，並提供有效格式範例。
- API 回傳成功但沒有符合的 `Records.Location` 時，視為無效縣市，而不是顯示空白天氣資訊；API 逾時、授權失敗或服務異常則使用不同的錯誤訊息回覆。
- 回覆內容應清楚顯示實際查詢的縣市，讓使用者確認名稱正規化與 API 查詢結果是否符合預期。
- 第一階段不建立快取，每次有效的天氣或降雨指令都即時呼叫 API，並沿用逾時控制，避免外部服務無回應時長時間占用處理流程。
- 未加入快取期間需留意 API 呼叫額度、短時間重複查詢與回應延遲；後續可再加入依縣市分類的短期快取、同時請求合併或使用者冷卻時間。
- Discord 訊息處理仍應讓各次查詢獨立執行，並將錯誤限制在該次請求，避免單一縣市查詢失敗影響其他對話指令。
- 日誌只記錄指令、縣市、耗時與錯誤類型，不得輸出 `.env` 中的 API Key，也不應將金鑰組合進回覆訊息。
- 測試項目至少包含 `天氣 臺中市`、`下雨 高雄市`、`台` 與 `臺` 的別名、未輸入縣市、不存在的縣市、API 逾時，以及原有非天氣對話規則是否仍能正常運作。
- 第一階段完成標準為：程式內沒有固定城市字串陣列、使用者可從 Discord 指定有效縣市、兩種天氣 action 共用相同的查詢流程，且無效輸入能收到明確提示。

## 2026-08-01：README 功能分類與啟動檔整理

- 將 `Readme.md` 的功能表增加「文字訊息」與「即時訊息」分類，讓固定回覆及動態計算功能更容易辨識。
- 將 `笨蛋`、`肚子餓` 等既有回覆內容直接列入功能表，方便核對 `talk.txt` 的實際行為。
- 將 Bot Presence 的活動型別由 `ActivityTypeStreaming` 調整為 `ActivityTypeCompeting`，目前 Discord 顯示內容仍是自訂的 Golang 開發狀態，不代表偵測到真實應用程式。
- 精簡 `start.bat` 與 `start.sh` 的空白及註解格式，維持 Windows 前景等待與 Debian `nohup` 背景啟動的既有架構。
- 本次文件編輯開始前 Git 工作樹為乾淨狀態，最近一次已提交修改包含 `Readme.md`、`main.go`、`start.bat` 與 `start.sh`。

## 2026-08-01：Discord 天氣互動輸入介面設計

- 參考 Discord 斜線指令介面，將需求拆成兩段式互動：先輸入 `/天氣`，再於 Discord 顯示的必填欄位輸入城市名稱。
- 確認城市名稱應由 Discord Interaction 傳入程式，不應由固定城市字串陣列、程式常數或預設臺北市代替使用者輸入。
- 規劃使用 `ApplicationCommandOptionString` 建立「城市名稱」欄位，並將輸入內容依序傳給地區解析、36 小時預報查詢及回覆格式化函式。
- 規劃在收到互動後先送出 deferred response，再以 goroutine 執行中央氣象署 API 查詢，完成後編輯原始回覆，避免外部 API 延遲造成 Discord 互動逾時。
- 規劃以 Discord Embed 的欄位排列呈現各預報時段；由於 Discord 訊息不支援原生 Markdown 表格，Embed 欄位比純文字對齊更適合桌面版與行動版。
- 規劃將常見的「台」正規化為「臺」，但不建立限制使用者輸入的固定城市陣列；`go-cwb` 所需的 `[]string{locationName}` 僅保留在 API 呼叫邊界。
- 重新確認後，城市輸入的唯一來源應是 Discord 的 Interaction Options，`loadWeatherPeriods()` 只接收解析完成的 `locationName`。
- 目前工作樹尚未包含上述斜線指令實作：`main.go` 仍只掛載 `messageCreate`，`weather.go` 仍由 `CWA_LOCATION` 取得預設地區，未設定時仍使用臺北市。
- 因此本段屬於已完成設計與需求確認、但尚未在目前版本落地的開發紀錄，不應在正式功能表中標示為已完成。

## 2026-08-01：Linux 背景程序停止流程設計

- 檢查 `start.sh` 的 PID 管理後，確認目前腳本能避免重複啟動、保存 PID 並顯示日誌位置，但沒有解析 `start`、`stop`、`restart` 或 `status` 動作。
- 目前停止方式仍是腳本輸出的 `kill <PID>`，不會主動清除 PID 檔，也不會等待 Bot 完成 Discord Session 關閉。
- 規劃將 PID 讀取、數字格式驗證及 `/proc/<PID>/exe` 執行檔核對集中為共用函式，避免 PID 檔過期後誤操作其他程序。
- 規劃讓 `bash start.sh stop` 先送出 `SIGTERM`，等待 `main.go` 接收訊號並執行 `dg.Close()`；等待逾時後再次核對程序身分，才使用 `SIGKILL` 作為最後手段。
- 規劃新增 `bash start.sh start|stop|restart|status` 操作介面，未提供參數時仍維持原有啟動行為。
- 目前工作樹中的 `start.sh` 尚未包含上述停止函式與子指令介面，因此這項修正同樣屬於已設計、尚未落地。

## 2026-08-01：專案現況與文件一致性盤點

- 盤點目前專案檔案包含 `.gitignore`、`DiscordGo.md`、`History.md`、`Readme.md`、`other.md`、`main.go`、`weather.go`、`talk.txt`、`start.sh`、`start.bat`、`go.mod` 與 `go.sum`。
- 專案根目錄存在 `.env`，本次只確認檔案存在且已被 `.gitignore` 排除，未讀取或記錄其中的 Token 與 API Key。
- `Readme.md` 的專案架構尚未列出 `weather.go`、`History.md`、`other.md` 與 `DiscordGo.md`。
- `Readme.md` 的 `.env` 範例目前只有 `DCToken`，尚未說明天氣功能需要 `CWA_API`，以及現行程式可選用的 `CWA_LOCATION`。
- `Readme.md` 的對話規則章節只列出 `exact` 與 `contains`，但 `talk.txt` 與 `main.go` 已支援 `action`。
- `Readme.md` 目前將 `天氣` 與 `下雨` 列為文字輸入功能，這與現行程式一致；若日後完成 `/天氣` 斜線指令，功能表、設定方式與操作範例必須同步更新。
- `other.md` 仍以 `kill <PID>` 說明停止方式，與目前 `start.sh` 一致；完成 `stop|restart|status` 後需要同步更新 Debian 操作章節。
- 其餘架構風險、實作順序與待確認決策集中整理於 `chat.md`，作為後續深入討論與需求收斂的文件。

## 2026-08-01：天氣介面決策與文字 action 架構確認

- Phoenix 決定不採用 `/天氣` 斜線指令，先前的 Discord 城市輸入、Guild Command、Embed 互動及城市別名規劃不進入目前實作。
- 保留 `talk.txt` 的 `action-天氣-weather_36h` 與 `action-下雨-rainProbability_action`，查詢縣市繼續由 `.env` 的 `CWA_LOCATION` 管理。
- 將 `現在時間` 從 `messageCreate()` 的直接 switch 移至 `talk.txt`，新增 `action-現在時間-local_time`。
- 在 `main.go` 登錄 `local_time` action，並由 `executeTalkAction()` 呼叫既有 `getLocalTime()`。
- 目前 Discord 文字訊息架構維持 `messageCreate()` 讀取 `exact`、`contains` 與 `action`，動態 action 負責節慶倒數、天氣、降雨及現在時間，九九乘法與算命仍由 `main.go` 直接處理。

## 2026-08-01：OMV systemd 執行環境

- 將 `start.sh` 由 `nohup`、背景程序、PID 檔及自訂日誌精簡為 systemd 前景入口。
- `start.sh` 先將 Go 程式編譯至 `/var/lib/discord-bot/discord-bot`，再使用 `exec` 讓 Bot 取代 Shell 程序。
- 新增 `discord-bot.service`，目前以 OMV 的 `/root/discord-bot` 作為 `WorkingDirectory` 與啟動路徑。
- 使用 `StateDirectory=discord-bot` 管理 `/var/lib/discord-bot`，避免編譯檔寫入 Git 工作樹。
- 設定 `Restart=on-failure`、`SIGTERM` 與 15 秒停止期限，讓異常結束自動重啟，正常停止則交由 `main.go` 關閉 Discord Session。
- 將標準輸出與錯誤交由 systemd journal 管理，操作方式改為 `systemctl start|stop|restart|status` 與 `journalctl -u discord-bot.service`。
- 重寫 `other.md` 的 OMV 部署流程，補充 Service 安裝、路徑調整、狀態查閱、日誌、更新部署及常見錯誤。

## 2026-08-01：README 與深度交流文件同步

- 更新 `Readme.md` 專案樹，加入 `weather.go`、`discord-bot.service`、`other.md`、`DiscordGo.md`、`History.md` 與 `chat.md`。
- 新增目前訊息處理架構圖，說明 `talk.txt` action 與 `main.go` 直接功能的分工。
- 補充 `.env` 的 `CWA_API` 與 `CWA_LOCATION`，並明確說明目前沒有 Discord 斜線指令或城市輸入欄位。
- 將功能分類改為文字規則、動態 action 與 `main.go` 功能，避免「即時訊息」與 Discord Interaction 混淆。
- 在對話規則章節補上 `action` 格式、白名單限制及 `現在時間` 範例。
- 將 Phoenix 於 `chat.md` 第 317 行後的六項答案轉換為實際決策與執行結果，保留未採用方案作為設計歷程。
