package main // 天氣功能的 Discord 事件入口，複雜查詢與格式化交由 weather.py。

import ( // 僅使用標準函式庫與已安裝的 DiscordGo。
	"bytes"         // 收集 Python 的 JSON 輸出。
	"context"       // 控制一次性子程序總執行時間。
	"encoding/json" // 解碼 Python 回傳結果。
	"errors"        // 建立固定安全錯誤。
	"fmt"           // 輸出不含敏感資訊的操作日誌。
	"os"            // 讀取預設城市及 Python 腳本設定。
	"os/exec"       // 啟動 Python，完成後回收程序。
	"strings"       // 處理城市欄位與搜尋文字。
	"sync"          // 保護城市清單與同步中的工作。
	"time"          // 設定查詢總時限。

	"github.com/bwmarrin/discordgo" // 接收指令與回覆 Embed 卡片。
) // 結束套件引用。

const ( // 共用天氣設定。
	defaultWeatherLocation = "臺北市"            // ENV 沒有預設城市時保留原有備援。
	defaultWeatherScript   = "weather.py"     // 由此腳本處理 CWA 與資料格式化。
	weatherQueryTimeout    = 15 * time.Second // 包含 Python 啟動及 CWA 查詢的總時限。
) // 結束常數。

var weatherErrors = map[string]error{ // 只允許固定中文錯誤，避免把 Python stderr 或 API 本文送到 Discord。
	"location": errors.New("找不到指定縣市的預報，請從城市選項重新選擇。"),                        // 明確輸入無效，不回退成預設城市。
	"data":     errors.New("中央氣象署目前沒有可用的預報資料，請稍後再試。"),                       // 區分上游資料暫缺。
	"key":      errors.New("尚未設定 CWA_API，請聯絡 Bot 管理員。"),                     // 缺少金鑰設定。
	"auth":     errors.New("中央氣象署授權失敗，請聯絡 Bot 管理員檢查 CWA_API。"),              // 授權失敗。
	"rate":     errors.New("中央氣象署查詢次數暫受限制，請稍後再試。"),                          // 額度或頻率限制。
	"timeout":  errors.New("天氣查詢逾時，請稍後再試。"),                                 // Python 或 CWA 等待逾時。
	"service":  errors.New("目前無法取得中央氣象署資料，請稍後再試。"),                          // 上游連線與服務異常。
	"input":    errors.New("天氣指令格式不正確，請重新選擇城市與指令。"),                         // 子程序輸入不合法。
	"runtime":  errors.New("天氣處理程式執行失敗，請聯絡 Bot 管理員檢查 Python 與 weather.py。"), // Python 不存在或回傳格式異常。
} // 結束安全錯誤對照。

// weatherResponse 定義一次 Python 執行回傳的 JSON。
type weatherResponse struct { // 城市清單與預報共用同一份交換型別。
	Locations []string                `json:"locations"`  // 完整城市集合由 CWA 資料產生。
	Message   string                  `json:"message"`    // 已整理好的舊文字指令回覆。
	Embed     *discordgo.MessageEmbed `json:"embed"`      // 已整理好的斜線指令預報卡片。
	ErrorCode string                  `json:"error_code"` // 固定代碼，不傳遞原始錯誤本文。
} // 結束 JSON 型別。

// weatherQuery 表示一次性子程序查詢，測試可注入離線替身。
type weatherQuery func(context.Context, string, string, string) (weatherResponse, error) // 參數依序為操作、城市與顯示模式。

// runWeatherPython 每次事件只啟動一個 Python，Run 會等待結束並回收程序。
func runWeatherPython(ctx context.Context, operation string, location string, mode string) (weatherResponse, error) { // 不使用 shell 拼接使用者輸入。
	var response weatherResponse          // 保存單次 JSON 回應。
	python, err := resolvePythonCommand() // 沿用專案既有的 PYTHON_BIN 與跨平台 Python 搜尋。
	var output bytes.Buffer               // stdout 僅接受結構化 JSON。
	var runErr error                      // 保存程序退出狀態。
	switch err {                          // Python 存在才建立子程序。
	case nil: // 已取得 Python 執行路徑。
		script := strings.TrimSpace(os.Getenv("WEATHER_PYTHON_SCRIPT")) // 允許部署時指定腳本路徑。
		switch script {                                                 // 沒有設定時使用專案內腳本。
		case "": // 採用目前工作目錄。
			script = defaultWeatherScript // 預設 weather.py。
		} // 結束腳本路徑處理。
		arguments := append(append([]string{}, python.args...), script, operation) // 複製 launcher 參數，避免共享狀態。
		switch operation {                                                         // 只有預報操作需要城市與模式。
		case "forecast": // 使用者已送出天氣或下雨指令。
			arguments = append(arguments, location, mode) // 城市是獨立參數，不會被當成 shell 指令。
		} // 結束命令參數組合。
		command := exec.CommandContext(ctx, python.path, arguments...) // 逾時時終止該次 Python 程序。
		command.Stdout = &output                                       // 收集 UTF-8 JSON。
		runErr = command.Run()                                         // stderr 不轉送至使用者或日誌，程序結束後立即回收。
		err = json.Unmarshal(output.Bytes(), &response)                // 退出非零時仍解析固定錯誤代碼。
	default: // 找不到可用的 Python。
		err = weatherErrors["runtime"] // 使用安全環境設定提示。
	} // 結束子程序執行。
	switch { // 集中分流失敗原因，避免連續提前回傳。
	case ctx.Err() != nil: // 整體查詢超過時限或被取消。
		err = weatherErrors["timeout"] // 不讓子程序繼續占用資源。
	case err != nil: // Python 不存在或 JSON 無法解析。
		err = weatherErrors["runtime"] // 不顯示底層 stdout 或 stderr。
	case response.ErrorCode != "": // Python 已提供可辨識的失敗分類。
		err = weatherErrorCode(response.ErrorCode) // 未知代碼也不直接顯示。
	case runErr != nil: // Python 退出失敗卻沒有合法錯誤代碼。
		err = weatherErrors["runtime"] // 視為執行失敗。
	default: // 程序正常退出且 JSON 可解析。
		err = validateWeatherResponse(response, operation) // 驗證本次操作必要的回傳欄位。
	} // 結束結果分類。
	return response, err // 單一回傳出口。
} // 結束一次性 Python 呼叫。

// validateWeatherResponse 防止空資料被當成成功回覆。
func validateWeatherResponse(response weatherResponse, operation string) error { // 不在 Go 重做 Python 的氣象整理。
	var err error // 預設 JSON 符合操作需求。
	switch {      // 依操作檢查最少必要欄位。
	case operation == "locations" && len(response.Locations) == 0: // 沒有城市不能覆蓋指令選項。
		err = weatherErrors["data"] // 要求修正後重啟重新同步。
	case operation == "forecast" && (strings.TrimSpace(response.Message) == "" || response.Embed == nil): // 預報必須同時有文字與卡片。
		err = weatherErrors["data"] // 不送出空白 Discord 訊息。
	case operation != "locations" && operation != "forecast": // 拒絕未知操作。
		err = weatherErrors["input"] // 保留固定輸入錯誤。
	} // 結束資料檢查。
	return err // 單一回傳出口。
} // 結束 JSON 驗證。

// weatherErrorCode 只回傳允許的固定錯誤。
func weatherErrorCode(code string) error { // 排除未知 Python 錯誤內容。
	err, known := weatherErrors[code] // 查找安全提示。
	switch known {                    // 未知代碼統一歸類為執行失敗。
	case false: // 不支援的錯誤代碼。
		err = weatherErrors["runtime"] // 不使用任意原始文字。
	} // 結束代碼檢查。
	return err // 單一回傳出口。
} // 結束錯誤代碼轉換。

// configuredWeatherLocation 保留原有 ENV 預設。
func configuredWeatherLocation() string { // 只有省略城市選項才使用預設。
	location := normalizeWeatherLocation(os.Getenv("CWA_LOCATION")) // 接受台與臺及前後空白。
	switch location {                                               // ENV 空白時延續原行為。
	case "": // 沒有設定城市。
		location = defaultWeatherLocation // 使用臺北市。
	} // 結束預設處理。
	return location // 單一回傳出口。
} // 結束預設城市解析。

// normalizeWeatherLocation 處理 Discord 城市搜尋與預設名稱。
func normalizeWeatherLocation(value string) string { // Python 仍負責最終查詢驗證。
	return strings.ReplaceAll(strings.TrimSpace(value), "台", "臺") // 不猜測縣市簡稱。
} // 結束名稱正規化。

// send36HourWeather 保留舊文字天氣 action，收到事件才啟動 Python。
func send36HourWeather(session *discordgo.Session, channelID string) { // 仍使用 ENV 預設城市。
	sendWeatherText(session, channelID, "weather") // 指定完整預報模式。
} // 結束文字天氣入口。

// rainProbabilityAction 保留舊文字降雨 action，收到事件才啟動 Python。
func rainProbabilityAction(session *discordgo.Session, channelID string) { // 仍使用 ENV 預設城市。
	sendWeatherText(session, channelID, "rain") // 指定降雨口語模式。
} // 結束文字降雨入口。

// sendWeatherText 僅負責事件查詢與 Discord 發送。
func sendWeatherText(session *discordgo.Session, channelID string, mode string) { // Python 已完成氣象資料處理。
	ctx, cancel := context.WithTimeout(context.Background(), weatherQueryTimeout)         // 每個事件限制總時間。
	defer cancel()                                                                        // 釋放計時器。
	response, err := runWeatherPython(ctx, "forecast", configuredWeatherLocation(), mode) // 執行並回收一次 Python。
	reply := response.Message                                                             // 採用 Python 完成的文字。
	switch err {                                                                          // 發送安全提示或成功結果。
	case nil: // 已取得預報。
	default: // 查詢失敗。
		reply = err.Error() // 僅來自 Go 固定錯誤對照。
	} // 結束結果選擇。
	_, sendErr := session.ChannelMessageSend(channelID, reply) // 共用一次發送。
	switch sendErr {                                           // 記錄原本未檢查的發送失敗。
	case nil: // 訊息已送出。
	default: // Discord 未接受訊息。
		fmt.Println("天氣文字回覆發送失敗，請確認權限與連線。") // 不輸出 Token。
	} // 結束發送結果處理。
} // 結束文字事件處理。

// weatherInteractionSession 供正式 Discord 與離線測試共用。
type weatherInteractionSession interface { // 只包含本功能需要的方法。
	InteractionRespond(*discordgo.Interaction, *discordgo.InteractionResponse) error                    // 立即確認或回傳建議。
	InteractionResponseEdit(*discordgo.Interaction, *discordgo.WebhookEdit) (*discordgo.Message, error) // 更新原始訊息。
} // 結束互動介面。

// weatherInteractions 將 Discord 指令入口與 CWA 城市資料同步分開。
type weatherInteractions struct { // 城市同步失敗不能讓指令一起消失。
	locations    []string     // 官方城市集合由 Python 取得。
	query        weatherQuery // 每次查詢使用完成即結束的 Python。
	catalogMutex sync.RWMutex // 事件讀取城市時避免與同步寫入競爭。
	refreshMutex sync.Mutex   // 同一時間只允許一個城市同步工作。
	lastRefresh  time.Time    // 避免連續按鍵重複啟動失敗的同步。
} // 結束互動狀態。

// newWeatherInteractions 只建立事件處理器，不等待 Python 或 CWA。
func newWeatherInteractions() *weatherInteractions { // Gateway 與斜線指令能先完成註冊。
	return &weatherInteractions{query: runWeatherPython} // 城市資料在指令入口建立後同步。
} // 結束事件處理器建立。

// cityLocations 回傳城市快照，避免事件直接存取共享切片。
func (handler *weatherInteractions) cityLocations() []string { // 查詢與同步可並行進行。
	handler.catalogMutex.RLock()                          // 取得唯讀鎖。
	locations := append([]string{}, handler.locations...) // 使用獨立切片產生選項。
	handler.catalogMutex.RUnlock()                        // 釋放唯讀鎖。
	return locations                                      // 單一回傳出口。
} // 結束城市快照。

// refreshLocations 由 Python 同步完整城市，失敗時保留指令與既有城市。
func (handler *weatherInteractions) refreshLocations() bool { // 不在自動完成事件內等待外部查詢。
	updated := false                        // 預設本次沒有更新資料。
	switch handler.refreshMutex.TryLock() { // 避免短時間重複的城市同步。
	case true: // 本次取得同步工作權限。
		defer handler.refreshMutex.Unlock() // 不論結果都釋放同步鎖。
		switch {                            // 失敗後至少間隔 30 秒才重試。
		case time.Since(handler.lastRefresh) >= 30*time.Second: // 首次同步的零時間也符合。
			handler.lastRefresh = time.Now()                                              // 在啟動 Python 前記錄嘗試時間。
			ctx, cancel := context.WithTimeout(context.Background(), weatherQueryTimeout) // 每次同步仍有總期限。
			defer cancel()                                                                // 回收計時器。
			response, err := handler.query(ctx, "locations", "", "")                      // 不傳城市篩選，取得 CWA 全部縣市。
			switch err {                                                                  // 成功才替換完整城市資料。
			case nil: // 已取得 Python 整理的官方名稱。
				handler.catalogMutex.Lock()                                   // 排他更新城市集合。
				handler.locations = append([]string{}, response.Locations...) // 不手動補入或截斷城市。
				handler.catalogMutex.Unlock()                                 // 完成後讓互動讀取。
				updated = true                                                // 允許更新固定城市選項。
				fmt.Printf("CWA 城市同步完成：%d 個縣市。\n", len(response.Locations))   // 與指令註冊日誌分開。
			default: // 城市資料暫時無法取得。
				fmt.Println("CWA 城市同步失敗；斜線指令入口仍保留，後續互動會觸發重試：", err) // 使用固定安全錯誤。
			} // 結束同步結果分派。
		} // 結束重試間隔判斷。
	} // 結束同步工作分派。
	return updated // 單一回傳出口。
} // 結束城市同步。

// commands 無論城市是否載入，都會定義兩個可用的斜線指令入口。
func (handler *weatherInteractions) commands() []*discordgo.ApplicationCommand { // 城市來源仍完全依 CWA。
	locations := handler.cityLocations()                    // 固定本次註冊使用的資料快照。
	commands := make([]*discordgo.ApplicationCommand, 0, 2) // 永遠建立天氣與下雨指令。
	for _, name := range []string{"天氣", "下雨"} {             // 此集合只描述功能名稱。
		option := &discordgo.ApplicationCommandOption{ // 保留原介面的城市名稱欄位。
			Type:         discordgo.ApplicationCommandOptionString,   // 城市使用正式名稱字串。
			Name:         "城市名稱",                                     // 與使用者提供的原本畫面一致。
			Description:  "選擇中央氣象署縣市，未選擇時使用預設城市",                     // 保留先前確認的 ENV 預設。
			Required:     false,                                      // 未指定城市仍可查詢預設地區。
			Autocomplete: len(locations) == 0 || len(locations) > 25, // 未載入或超過上限時提供搜尋入口。
		} // 結束城市欄位定義。
		switch option.Autocomplete { // 固定選項與搜尋不能同時使用。
		case false: // 全部 CWA 城市可列入選單。
			for _, location := range locations { // 使用本次快照的完整資料。
				option.Choices = append(option.Choices, &discordgo.ApplicationCommandOptionChoice{Name: location, Value: location}) // 顯示與查詢名稱一致。
			} // 結束全部城市選項建立。
		} // 結束選項模式分派。
		description := "查詢縣市未來 36 小時天氣預報" // 完整天氣用途。
		switch name {                     // 區分降雨入口用途。
		case "下雨": // 提供最高降雨機率與建議。
			description = "查詢縣市未來 36 小時降雨機率與建議" // 保留原功能說明。
		} // 結束說明分派。
		commands = append(commands, &discordgo.ApplicationCommand{Name: name, Type: discordgo.ChatApplicationCommand, Description: description, Options: []*discordgo.ApplicationCommandOption{option}}) // 不再因城市清單空白而略過指令。
	} // 結束兩個入口建立。
	return commands // 單一回傳出口。
} // 結束指令定義。

// register 先建立可見入口，再同步 CWA 並補上全部城市選項。
func (handler *weatherInteractions) register(session *discordgo.Session) { // Python 失敗不阻止第一階段註冊。
	handler.registerCommands(session)   // 先讓 /天氣 與 /下雨 出現在 Discord。
	switch handler.refreshLocations() { // 再取得 CWA 資料。
	case true: // 城市同步成功。
		handler.registerCommands(session) // 以完整官方城市更新欄位選單。
	} // 結束城市選項同步。
} // 結束入口與資料分階段註冊。

// registerCommands 只更新本功能的兩個指令，失敗日誌記錄可追查的狀態碼。
func (handler *weatherInteractions) registerCommands(session *discordgo.Session) { // 不批次覆蓋其他功能。
	guildID := strings.TrimSpace(os.Getenv("DISCORD_COMMAND_GUILD_ID")) // 留白是全域，指定值為測試伺服器。
	scope := "全域"                                                       // 讓日誌能區分指令註冊範圍。
	switch guildID {                                                    // 有指定伺服器時顯示其 ID。
	case "": // 保留全域描述。
	default: // 使用測試伺服器範圍。
		scope = "伺服器 " + guildID // 不包含任何密鑰。
	} // 結束範圍描述。
	for _, command := range handler.commands() { // 兩個入口分別嘗試註冊。
		registered, err := session.ApplicationCommandCreate(session.State.User.ID, guildID, command) // 使用目前 Bot 的應用程式 ID。
		switch err {                                                                                 // 分開記錄實際成功或失敗。
		case nil: // Discord 已接受指令註冊。
			fmt.Printf("已註冊 /%s：範圍=%s，指令 ID=%s，城市選項=%d，自動完成=%t。\n",
			command.Name,
			scope,
			registered.ID,
			len(command.Options[0].Choices),
			command.Options[0].Autocomplete) // 明確記錄可見入口與選單狀態。
		default: // 使用狀態碼區分授權與其他 REST 問題。
			logWeatherDiscordError("註冊 /"+command.Name+"（"+scope+"）", err) // 不輸出含 Token 的請求內容。
		} // 結束註冊結果分派。
	} // 結束兩個指令同步。
} // 結束指令註冊。

// logWeatherDiscordError 記錄 HTTP 與 Discord 錯誤代碼，避免只留下籠統失敗訊息。
func logWeatherDiscordError(action string, err error) { // 用於註冊、初始確認與結果更新。
	status, code := 0, 0               // 非 REST 錯誤保留零值。
	var restError *discordgo.RESTError // 安全擷取 API 的狀態欄位。
	switch {                           // 不記錄原始錯誤字串或請求本文。
	case errors.As(err, &restError) && restError != nil: // REST 回應包含可追查的分類。
		switch { // HTTP 回應可能缺漏。
		case restError.Response != nil: // 已收到 HTTP 回應。
			status = restError.Response.StatusCode // 保存 HTTP 狀態。
		} // 結束 HTTP 狀態擷取。
		switch { // Discord JSON 錯誤可能缺漏。
		case restError.Message != nil: // 已解析 Discord 錯誤代碼。
			code = restError.Message.Code // 不輸出回應的任意文字。
		} // 結束 Discord 代碼擷取。
	} // 結束錯誤分類。
	fmt.Printf("%s失敗：HTTP=%d，Discord=%d，錯誤型別=%T。\n", action, status, code, err) // 提供排錯所需證據。
} // 結束安全 Discord 日誌。
// citySuggestions 只處理 Discord 欄位搜尋，不為每次按鍵啟動 Python。
func (handler *weatherInteractions) citySuggestions(input string) []*discordgo.ApplicationCommandOptionChoice { // 搜尋完整的啟動城市集合。
	choices := make([]*discordgo.ApplicationCommandOptionChoice, 0) // 無符合項目時仍回傳合法空陣列。
	query := normalizeWeatherLocation(input)                        // 搜尋接受台與臺。
	for _, location := range handler.cityLocations() {              // 不預先截斷城市資料來源。
		switch { // 遵守 Discord 每次建議數量，仍能搜尋清單尾端。
		case len(choices) < 25 && strings.Contains(normalizeWeatherLocation(location), query): // 選擇符合輸入的城市。
			choices = append(choices, &discordgo.ApplicationCommandOptionChoice{Name: location, Value: location}) // 顯示與送出使用同一官方名稱。
		} // 結束搜尋分派。
	} // 結束全部城市搜尋。
	return choices // 單一回傳出口。
} // 結束城市建議。

// handle 符合 DiscordGo 事件簽章。
func (handler *weatherInteractions) handle(session *discordgo.Session, event *discordgo.InteractionCreate) { // DiscordGo 預設獨立處理每個事件。
	handler.respond(session, event)                       // 先完成必要的 Discord 初始確認或建議回覆。
	switch data := weatherCommandData(event); data.Name { // 只讓天氣功能的互動觸發城市恢復。
	case "天氣", "下雨": // 資料空白時於背景同步，不阻塞初始回覆。
		switch len(handler.cityLocations()) { // 已有城市時不啟動額外 Python。
		case 0: // 首次同步失敗後可由互動恢復。
			go handler.refreshLocations() // 同步鎖與重試間隔限制重複工作。
		} // 結束城市可用性判斷。
	} // 結束背景恢復分派。
} // 結束事件轉接。

// weatherCommandData 安全取得本功能支援的事件資料。
func weatherCommandData(event *discordgo.InteractionCreate) discordgo.ApplicationCommandInteractionData { // 不使用連續提前回傳。
	var data discordgo.ApplicationCommandInteractionData // 無效事件保持空指令。
	switch {                                             // 集中判斷事件是否可解碼。
	case event == nil || event.Interaction == nil: // 無效事件不處理。
	case event.Type == discordgo.InteractionApplicationCommand || event.Type == discordgo.InteractionApplicationCommandAutocomplete: // 接收斜線指令及建議。
		data, _ = event.Data.(discordgo.ApplicationCommandInteractionData) // 異常型別保留空值，避免 panic。
	} // 結束事件資料分派。
	return data // 單一回傳出口。
} // 結束事件資料擷取。

// respond 依指令與事件種類分派，不在入口處理氣象資料。
func (handler *weatherInteractions) respond(session weatherInteractionSession, event *discordgo.InteractionCreate) { // 所有請求參數獨立保存。
	data := weatherCommandData(event) // 篩選無效事件。
	switch data.Name {                // 不干涉其他指令。
	case "天氣", "下雨": // 兩個入口共用同一事件流程。
		fmt.Printf("已收到 /%s 互動：種類=%d。\n", data.Name, event.Type)      // 證明事件已進入目前 Bot 的處理器。
		location, focused := weatherInteractionLocation(data.Options) // 城市省略時才使用 ENV。
		switch event.Type {                                           // 區分輸入建議與正式送出。
		case discordgo.InteractionApplicationCommandAutocomplete: // 不啟動外部查詢。
			handler.respondSuggestions(session, event.Interaction, location, focused) // 即時回覆記憶體內的城市選項。
		case discordgo.InteractionApplicationCommand: // 使用者已送出查詢。
			handler.respondForecast(session, event.Interaction, data.Name, location) // 確認後啟動一次 Python。
		} // 結束互動分派。
	} // 結束指令分派。
} // 結束事件入口。

// weatherInteractionLocation 區分未提供城市與明確空白。
func weatherInteractionLocation(options []*discordgo.ApplicationCommandInteractionDataOption) (string, bool) { // 不依欄位順序解析。
	location := configuredWeatherLocation() // 沒有城市欄位時才使用預設。
	focused := false                        // 預設不是正在輸入城市。
	for _, option := range options {        // 逐一辨識欄位。
		switch { // 接受目前與舊指令的欄位名稱。
		case option == nil: // 忽略空欄位。
		case option.Name == "城市" || option.Name == "城市名稱": // 相容舊註冊尚未更新的情況。
			value, _ := option.Value.(string)          // 錯誤型別留下空字串，不改查預設城市。
			location = normalizeWeatherLocation(value) // 最終城市驗證仍由 Python 處理。
			focused = option.Focused                   // 保存建議欄位狀態。
		} // 結束欄位分派。
	} // 結束城市解析。
	return location, focused // 單一回傳出口。
} // 結束城市參數處理。

// respondSuggestions 使用 Discord 自動完成專用回覆，不進行 CWA 查詢。
func (handler *weatherInteractions) respondSuggestions(session weatherInteractionSession, interaction *discordgo.Interaction, location string, focused bool) { // 與正式查詢分開。
	choices := make([]*discordgo.ApplicationCommandOptionChoice, 0) // 未知聚焦欄位也回覆合法空結果。
	switch focused {                                                // 只有城市欄位需要搜尋。
	case true: // 城市輸入中。
		choices = handler.citySuggestions(location) // 使用啟動時 CWA 的完整集合。
	} // 結束聚焦分派。
	err := session.InteractionRespond(interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionApplicationCommandAutocompleteResult, Data: &discordgo.InteractionResponseData{Choices: choices}}) // 立即回傳建議。
	switch err {                                                                                                                                                                                                // 記錄建議發送失敗。
	case nil: // 建議已送出。
	default: // Discord 未接受建議。
		logWeatherDiscordError("天氣城市建議回覆", err) // 不顯示互動 Token。
	} // 結束建議結果處理。
} // 結束城市建議回覆。

// respondForecast 必須先確認互動，再啟動耗時的 Python 工作。
func (handler *weatherInteractions) respondForecast(session weatherInteractionSession, interaction *discordgo.Interaction, command string, location string) { // 避免 Discord 初始確認逾時。
	err := session.InteractionRespond(interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredChannelMessageWithSource}) // 先告知正在處理。
	switch err {                                                                                                                                        // 確認失敗不啟動子程序。
	case nil: // 已可以編輯原始回覆。
		edit := handler.forecastReply(command, location)                 // 單次 Python 工作完成後取得卡片或錯誤。
		_, editErr := session.InteractionResponseEdit(interaction, edit) // 更新同一份原始訊息。
		switch editErr {                                                 // 更新結果與查詢本身分開記錄。
		case nil: // 已完成互動。
		default: // 結果更新失敗。
			logWeatherDiscordError("天氣指令結果更新", editErr) // 不輸出敏感請求內容。
		} // 結束更新結果分派。
	default: // 初始確認未成功。
		logWeatherDiscordError("天氣指令初始確認（本次查詢已停止）", err) // 不再消耗 CWA 查詢。
	} // 結束初始確認分派。
} // 結束查詢互動。

// forecastReply 僅選擇 Python 操作模式與 Discord 回覆型別。
func (handler *weatherInteractions) forecastReply(command string, location string) *discordgo.WebhookEdit { // 不在 Go 解析氣象因子。
	ctx, cancel := context.WithTimeout(context.Background(), weatherQueryTimeout) // 每個事件有獨立總期限。
	defer cancel()                                                                // 釋放計時器。
	mode := "weather"                                                             // 預設完整預報模式。
	switch command {                                                              // 選擇降雨摘要模式。
	case "下雨": // 降雨入口。
		mode = "rain" // Python 負責最高機率與口語建議。
	} // 結束顯示模式分派。
	response, err := handler.query(ctx, "forecast", location, mode) // 查詢完成後子程序已結束。
	edit := &discordgo.WebhookEdit{}                                // 保存原始訊息的更新內容。
	switch err {                                                    // 查詢成功與失敗都必須完成 Deferred 回覆。
	case nil: // Python 已建立完整卡片。
		embeds := []*discordgo.MessageEmbed{response.Embed} // 每次查詢一張預報卡片。
		edit.Embeds = &embeds                               // Go 只負責交給 Discord。
	default: // 子程序或 CWA 查詢失敗。
		message := err.Error()                         // 正式查詢只回傳固定安全錯誤。
		edit.Content = &message                        // 以文字完成互動。
		fmt.Printf("/%s 查詢未完成：%s\n", command, message) // 不記錄原始 stderr 或 API 本文。
	} // 結束回覆內容選擇。
	return edit // 單一回傳出口。
} // 結束查詢回覆建立。
