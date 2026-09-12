package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	//defaultStockNumber 是未指定股票代號時使用的範例股票。
	defaultStockNumber = "2377"
	//defaultStockScript 是 Go 每次查詢時執行的 Python 程式。
	defaultStockScript = "stock.py"
)

var (
	//stockPriceCommandPattern 支援「2377股價」格式的個股行情指令。
	stockPriceCommandPattern = regexp.MustCompile(`^([0-9A-Za-z]{4,6})股價$`)
	//stockSuggestionCommandPattern 支援「2377股票建議」格式的個股分析指令。
	stockSuggestionCommandPattern = regexp.MustCompile(`^([0-9A-Za-z]{4,6})股票建議$`)
)

// stockResponse 接收 Python 子程序以 JSON 回傳的證交所 CSV 與錯誤資訊。
type stockResponse struct {
	Date           string `json:"date"`
	AllMarketRaw   string `json:"all_market_raw"`
	SingleStockRaw string `json:"single_stock_raw"`
	Error          string `json:"error"`
}

// stockDailyRecord 保存證交所個股日成交資訊。
type stockDailyRecord struct {
	date          string
	tradeVolume   string
	openingPrice  float64
	highestPrice  float64
	lowestPrice   float64
	closingPrice  float64
	priceChange   float64
	transactionNo string
}

// stockCommand 保存 messageCreate 從 Discord 訊息解析出的股票事件內容。
type stockCommand struct {
	action       string
	stockNumber  string
	errorMessage string
}

// parseStockCommand 辨識帶有股票代號的動態指令，但不在解析階段執行 Python。
func parseStockCommand(content string) (stockCommand, bool) {
	command := strings.TrimSpace(content)
	//不攔截 talk.txt 的無代號「股票建議」action，讓統一分派流程使用預設代號。
	if command == "股票建議" {
		return stockCommand{}, false
	}
	if matches := stockPriceCommandPattern.FindStringSubmatch(command); len(matches) == 2 {
		return stockCommand{
			action:      stockPriceActionName,
			stockNumber: strings.ToUpper(matches[1]),
		}, true
	}
	if matches := stockSuggestionCommandPattern.FindStringSubmatch(command); len(matches) == 2 {
		return stockCommand{
			action:      stockSuggestionActionName,
			stockNumber: strings.ToUpper(matches[1]),
		}, true
	}
	if strings.HasSuffix(command, "股價") || strings.HasSuffix(command, "股票建議") {
		return stockCommand{
			errorMessage: "股票代號格式不正確，請輸入例如「2377股價」或「2377股票建議」。",
		}, true
	}
	return stockCommand{}, false
}

// getStockData 在每次收到指令時啟動 Python 子程序，不使用連接埠或常駐服務。
func getStockData(ctx context.Context, queryType string, stockNumber string) (*stockResponse, error) {
	pythonCommand, pythonArguments, err := resolvePythonCommand()
	if err != nil {
		return nil, err
	}

	stockScript := strings.TrimSpace(os.Getenv("STOCK_PYTHON_SCRIPT"))
	if stockScript == "" {
		stockScript = defaultStockScript
	}
	arguments := append(pythonArguments, stockScript, queryType)
	if stockNumber != "" {
		arguments = append(arguments, stockNumber)
	}

	command := exec.CommandContext(ctx, pythonCommand, arguments...)
	var standardError bytes.Buffer
	command.Stderr = &standardError
	standardOutput, commandErr := command.Output()
	if ctx.Err() != nil {
		return nil, fmt.Errorf("Python 股票查詢逾時：%w", ctx.Err())
	}

	response := &stockResponse{}
	decodeErr := json.Unmarshal(standardOutput, response)
	if decodeErr != nil {
		if commandErr != nil {
			return nil, fmt.Errorf("Python 股票查詢失敗：%v；%s", commandErr, strings.TrimSpace(standardError.String()))
		}
		return nil, fmt.Errorf("Python 回傳的 JSON 格式錯誤：%w", decodeErr)
	}
	if response.Error != "" {
		return nil, fmt.Errorf("%s", response.Error)
	}
	if commandErr != nil {
		return nil, fmt.Errorf("Python 股票查詢失敗：%v；%s", commandErr, strings.TrimSpace(standardError.String()))
	}
	return response, nil
}

// resolvePythonCommand 依環境變數與作業系統尋找可執行的 Python。
func resolvePythonCommand() (string, []string, error) {
	if configuredCommand := strings.TrimSpace(os.Getenv("PYTHON_BIN")); configuredCommand != "" {
		path, err := exec.LookPath(configuredCommand)
		if err != nil {
			return "", nil, fmt.Errorf("找不到 PYTHON_BIN 指定的 Python：%w", err)
		}
		return path, nil, nil
	}

	type pythonCandidate struct {
		command   string
		arguments []string
	}
	candidates := []pythonCandidate{{command: "python3"}, {command: "python"}}
	if runtime.GOOS == "windows" {
		candidates = []pythonCandidate{{command: "python"}, {command: "python3"}, {command: "py", arguments: []string{"-3"}}}
	}
	for _, candidate := range candidates {
		path, err := exec.LookPath(candidate.command)
		if err == nil {
			return path, candidate.arguments, nil
		}
	}
	return "", nil, fmt.Errorf("找不到 Python，請安裝 Python 3 或設定 PYTHON_BIN")
}

// stockPriceAction 即時取得個股當月資料，並顯示最新一個交易日行情。
func stockPriceAction(s *discordgo.Session, channelID string, stockNumber string) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	response, err := getStockData(ctx, "stock", stockNumber)
	if err != nil {
		s.ChannelMessageSend(channelID, fmt.Sprintf("目前無法取得 %s 股價：%v", stockNumber, err))
		return
	}
	reply, err := formatStockPriceReply(stockNumber, response.SingleStockRaw)
	if err != nil {
		s.ChannelMessageSend(channelID, fmt.Sprintf("目前無法解析 %s 的證交所資料：%v", stockNumber, err))
		return
	}
	s.ChannelMessageSend(channelID, reply)
}

// formatStockPriceReply 將證交所個股 CSV 整理為附圖使用的 Discord 訊息格式。
func formatStockPriceReply(stockNumber string, rawCSV string) (string, error) {
	records, err := parseStockDailyRecords(rawCSV)
	if err != nil {
		return "", err
	}

	stockName := extractStockName(rawCSV, stockNumber)
	displayName := stockNumber
	if stockName != "" {
		displayName = fmt.Sprintf("%s（%s）", stockNumber, stockName)
	}

	latest := records[len(records)-1]
	reply := fmt.Sprintf(
		"📈 名稱：%s\r\n 📈 %s 個股行情（%s）\r\n收盤：%.2f 元｜漲跌：%+.2f 元\r\n開盤：%.2f｜最高：%.2f｜最低：%.2f\r\n成交股數：%s｜成交筆數：%s\r\n資料來源：臺灣證券交易所",
		displayName,
		stockNumber,
		latest.date,
		latest.closingPrice,
		latest.priceChange,
		latest.openingPrice,
		latest.highestPrice,
		latest.lowestPrice,
		latest.tradeVolume,
		latest.transactionNo,
	)
	return reply, nil
}

// extractStockName 從證交所 STOCK_DAY CSV 標題擷取指定股票的名稱。
func extractStockName(rawCSV string, stockNumber string) string {
	reader := csv.NewReader(strings.NewReader(strings.TrimPrefix(rawCSV, "\ufeff")))
	reader.FieldsPerRecord = -1
	rows, err := reader.ReadAll()
	if err != nil {
		return ""
	}
	for _, row := range rows {
		for _, field := range row {
			parts := strings.Fields(strings.TrimSpace(field))
			for index, part := range parts {
				if part == stockNumber && index+1 < len(parts) {
					return strings.Trim(parts[index+1], "\"()（）")
				}
			}
		}
	}
	return ""
}

// dailyMarketAction 即時取得證交所單日大盤資料並整理主要市場指標。
func dailyMarketAction(s *discordgo.Session, channelID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	response, err := getStockData(ctx, "market", "")
	if err != nil {
		s.ChannelMessageSend(channelID, fmt.Sprintf("目前無法取得單日大盤資訊：%v", err))
		return
	}
	reply, err := formatDailyMarket(response.Date, response.AllMarketRaw)
	if err != nil {
		s.ChannelMessageSend(channelID, fmt.Sprintf("目前無法解析證交所大盤資料：%v", err))
		return
	}
	s.ChannelMessageSend(channelID, reply)
}

// stockSuggestionAction 以證交所近期收盤價計算均線趨勢並產生規則式建議。
func stockSuggestionAction(s *discordgo.Session, channelID string, stockNumber string) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	response, err := getStockData(ctx, "stock", stockNumber)
	if err != nil {
		s.ChannelMessageSend(channelID, fmt.Sprintf("目前無法取得 %s 的分析資料：%v", stockNumber, err))
		return
	}
	records, err := parseStockDailyRecords(response.SingleStockRaw)
	if err != nil {
		s.ChannelMessageSend(channelID, fmt.Sprintf("目前無法解析 %s 的證交所資料：%v", stockNumber, err))
		return
	}

	latest := records[len(records)-1]
	shortDays := min(5, len(records))
	longDays := min(20, len(records))
	shortAverage := averageClosingPrice(records[len(records)-shortDays:])
	longAverage := averageClosingPrice(records[len(records)-longDays:])
	suggestion := buildStockSuggestion(latest.closingPrice, shortAverage, longAverage)
	reply := fmt.Sprintf(
		"🧭 %s 股票建議（%s）\n最新收盤：%.2f 元\n%d 日均價：%.2f 元｜%d 日均價：%.2f 元\n判讀：%s\n\n這是依證交所歷史價格產生的規則式觀察，不構成投資建議。",
		stockNumber,
		latest.date,
		latest.closingPrice,
		shortDays,
		shortAverage,
		longDays,
		longAverage,
		suggestion,
	)
	s.ChannelMessageSend(channelID, reply)
}

// parseStockDailyRecords 將證交所 STOCK_DAY CSV 解析為依日期排列的每日資料。
func parseStockDailyRecords(rawCSV string) ([]stockDailyRecord, error) {
	if strings.TrimSpace(rawCSV) == "" || strings.Contains(rawCSV, "Error fetching") || strings.Contains(rawCSV, "Network Error") {
		return nil, fmt.Errorf("Python 服務未傳回可用的個股行情")
	}

	reader := csv.NewReader(strings.NewReader(strings.TrimPrefix(rawCSV, "\ufeff")))
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("CSV 格式錯誤：%w", err)
	}

	parsed := make([]stockDailyRecord, 0)
	for _, row := range records {
		if len(row) < 9 || strings.TrimSpace(row[0]) == "日期" {
			continue
		}
		openingPrice, openingErr := parseTWSEFloat(row[3])
		highestPrice, highestErr := parseTWSEFloat(row[4])
		lowestPrice, lowestErr := parseTWSEFloat(row[5])
		closingPrice, closingErr := parseTWSEFloat(row[6])
		priceChange, changeErr := parseTWSEFloat(strings.TrimPrefix(strings.TrimSpace(row[7]), "X"))
		if openingErr != nil || highestErr != nil || lowestErr != nil || closingErr != nil || changeErr != nil {
			continue
		}
		parsed = append(parsed, stockDailyRecord{
			date:          strings.TrimSpace(row[0]),
			tradeVolume:   strings.TrimSpace(row[1]),
			openingPrice:  openingPrice,
			highestPrice:  highestPrice,
			lowestPrice:   lowestPrice,
			closingPrice:  closingPrice,
			priceChange:   priceChange,
			transactionNo: strings.TrimSpace(row[8]),
		})
	}
	if len(parsed) == 0 {
		return nil, fmt.Errorf("找不到有效的交易日資料")
	}
	return parsed, nil
}

// formatDailyMarket 從 MI_INDEX CSV 擷取發行量加權股價指數與市場成交統計。
func formatDailyMarket(date string, rawCSV string) (string, error) {
	if strings.TrimSpace(rawCSV) == "" || strings.Contains(rawCSV, "Error fetching") || strings.Contains(rawCSV, "Network Error") {
		return "", fmt.Errorf("Python 服務未傳回可用的大盤行情")
	}

	reader := csv.NewReader(strings.NewReader(strings.TrimPrefix(rawCSV, "\ufeff")))
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		return "", fmt.Errorf("CSV 格式錯誤：%w", err)
	}

	var indexLine string
	var marketLine string
	for _, row := range records {
		if len(row) >= 5 && strings.TrimSpace(row[0]) == "發行量加權股價指數" {
			indexLine = fmt.Sprintf("加權指數：%s｜漲跌：%s%s（%s%%）", strings.TrimSpace(row[1]), formatTWSESign(row[2]), strings.TrimSpace(row[3]), strings.TrimSpace(row[4]))
		}
		if len(row) >= 4 && strings.TrimSpace(row[0]) == "1.一般股票" {
			marketLine = fmt.Sprintf("一般股票成交金額：%s 元\n成交股數：%s 股｜成交筆數：%s 筆", strings.TrimSpace(row[1]), strings.TrimSpace(row[2]), strings.TrimSpace(row[3]))
		}
	}
	if indexLine == "" {
		return "", fmt.Errorf("找不到發行量加權股價指數")
	}
	if marketLine == "" {
		marketLine = "一般股票成交統計：證交所本次回應未提供可辨識資料"
	}
	return fmt.Sprintf("📊 單日大盤（%s）\n%s\n%s\n\n資料來源：臺灣證券交易所 MI_INDEX", date, indexLine, marketLine), nil
}

// parseTWSEFloat 移除證交所數字中的千分位逗號後轉為浮點數。
func parseTWSEFloat(value string) (float64, error) {
	normalized := strings.ReplaceAll(strings.TrimSpace(value), ",", "")
	normalized = strings.ReplaceAll(normalized, "+", "")
	return strconv.ParseFloat(normalized, 64)
}

// averageClosingPrice 計算指定交易日範圍的平均收盤價。
func averageClosingPrice(records []stockDailyRecord) float64 {
	var total float64
	for _, record := range records {
		total += record.closingPrice
	}
	return total / float64(len(records))
}

// buildStockSuggestion 依短期、較長期均價與最新收盤價產生中性觀察建議。
func buildStockSuggestion(latest float64, shortAverage float64, longAverage float64) string {
	switch {
	case latest > shortAverage && shortAverage > longAverage:
		return "短期價格位於均價之上且趨勢偏強，可續看量價是否同步，避免追高。"
	case latest < shortAverage && shortAverage < longAverage:
		return "短期價格位於均價之下且趨勢偏弱，宜先觀望並設定可承受的風險範圍。"
	default:
		return "短期與較長期趨勢尚未一致，方向不明顯，可等待更明確訊號。"
	}
}

// formatTWSESign 將證交所漲跌符號欄位轉為容易閱讀的正負號。
func formatTWSESign(value string) string {
	switch strings.TrimSpace(value) {
	case "+":
		return "+"
	case "-":
		return "-"
	default:
		return ""
	}
}
