package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"
)

const (
	// defaultStockNumber 是 talk.txt 未指定股票代號時使用的示範代號。
	defaultStockNumber = "2377"
	// defaultStockScript 是事件發生時才執行的 Python 股票查詢程式。
	defaultStockScript = "stock.py"
	// stockQueryTimeout 限制單次股票事件占用背景工作的時間。
	stockQueryTimeout = 40 * time.Second
)

var (
	// stockPriceCommandPattern 辨識「2377股價」形式的個股行情指令。
	stockPriceCommandPattern = regexp.MustCompile(`^([0-9]{4,6}[A-Za-z]?)股價$`)
	// stockSuggestionCommandPattern 辨識「2377股票建議」形式的分析指令。
	stockSuggestionCommandPattern = regexp.MustCompile(`^([0-9]{4,6}[A-Za-z]?)股票建議$`)
)

// stockCommand 保存 main.go 執行股票事件所需的最少資訊。
type stockCommand struct {
	action       string
	stockNumber  string
	errorMessage string
}

// stockResponse 對應 stock.py 回傳的精簡 JSON 格式。
type stockResponse struct {
	Message string `json:"message"`
	Error   string `json:"error"`
}

// pythonCommand 保存 Python 執行檔與可能需要的啟動參數。
type pythonCommand struct {
	path string
	args []string
}

// parseStockCommand 將 Discord 訊息轉成可由事件處理器執行的股票命令。
func parseStockCommand(content string) (stockCommand, bool) {
	// 移除使用者輸入前後空白，並統一英文字母大小寫。
	message := strings.ToUpper(strings.TrimSpace(content))
	// 優先辨識個股行情指令。
	if matches := stockPriceCommandPattern.FindStringSubmatch(message); len(matches) == 2 {
		return stockCommand{action: stockPriceActionName, stockNumber: matches[1]}, true
	}
	// 接著辨識股票建議指令。
	if matches := stockSuggestionCommandPattern.FindStringSubmatch(message); len(matches) == 2 {
		return stockCommand{action: stockSuggestionActionName, stockNumber: matches[1]}, true
	}
	// 包含指令關鍵字但格式錯誤時，回傳可直接顯示的操作提示。
	if strings.HasSuffix(message, "股價") || strings.HasSuffix(message, "股票建議") {
		return stockCommand{
			errorMessage: "股票代號格式不正確，請輸入四至六碼英數字，例如：2377股價。",
		}, true
	}
	// 其餘訊息交回 main.go 的一般 talk 規則處理。
	return stockCommand{}, false
}

// getStockReply 將 main.go 的股票事件轉成 Python 查詢並取得可直接發送的訊息。
func getStockReply(action string, stockNumber string) (string, error) {
	// 將 Go action 名稱映射成 stock.py 接受的查詢類型。
	queryType := ""
	// 只允許已註冊的三種股票事件進入 Python。
	switch action {
	case dailyMarketActionName:
		queryType = "market"
	case stockPriceActionName:
		queryType = "price"
	case stockSuggestionActionName:
		queryType = "suggestion"
	default:
		return "", fmt.Errorf("不支援的股票功能：%s", action)
	}

	// 每次 Discord 事件建立獨立逾時，不常駐 Python 服務或預存行情。
	ctx, cancel := context.WithTimeout(context.Background(), stockQueryTimeout)
	defer cancel()

	// 執行一次 Python 程式並解析其精簡 JSON 回應。
	response, err := runStockPython(ctx, queryType, stockNumber)
	if err != nil {
		return "", err
	}
	// 防止 Python 成功結束但沒有提供 Discord 回覆內容。
	if strings.TrimSpace(response.Message) == "" {
		return "", fmt.Errorf("股票資料程式未回傳訊息")
	}
	// 回傳已由 Python 完成格式化的繁體中文訊息。
	return response.Message, nil
}

// runStockPython 啟動一次 stock.py，完成後立即結束子程序。
func runStockPython(ctx context.Context, queryType string, stockNumber string) (stockResponse, error) {
	// 尋找目前環境可用的 Python 命令。
	python, err := resolvePythonCommand()
	if err != nil {
		return stockResponse{}, err
	}
	// 允許以環境變數指定腳本位置，同時保留專案根目錄預設值。
	stockScript := strings.TrimSpace(os.Getenv("STOCK_PYTHON_SCRIPT"))
	if stockScript == "" {
		stockScript = defaultStockScript
	}
	// 組合 Python 啟動參數；大盤查詢不需要股票代號。
	arguments := append(append([]string{}, python.args...), stockScript, queryType)
	if strings.TrimSpace(stockNumber) != "" {
		arguments = append(arguments, strings.ToUpper(strings.TrimSpace(stockNumber)))
	}
	// 將本次查詢交給獨立 Python 子程序。
	command := exec.CommandContext(ctx, python.path, arguments...)
	// 分別保存標準輸出與錯誤輸出，避免錯誤文字破壞 JSON。
	var standardOutput bytes.Buffer
	var standardError bytes.Buffer
	command.Stdout = &standardOutput
	command.Stderr = &standardError
	// 等待一次性查詢完成。
	runError := command.Run()
	// 逾時時回傳明確訊息。
	if ctx.Err() == context.DeadlineExceeded {
		return stockResponse{}, fmt.Errorf("股票資料查詢逾時")
	}

	// Python 無論查詢成功或失敗都應回傳固定 JSON。
	var response stockResponse
	if err := json.Unmarshal(standardOutput.Bytes(), &response); err != nil {
		if runError != nil {
			return stockResponse{}, fmt.Errorf("股票資料程式執行失敗：%s", strings.TrimSpace(standardError.String()))
		}
		return stockResponse{}, fmt.Errorf("股票資料回應格式錯誤：%w", err)
	}
	// 優先使用 Python 提供的可讀錯誤原因。
	if strings.TrimSpace(response.Error) != "" {
		return stockResponse{}, fmt.Errorf("%s", response.Error)
	}
	// JSON 沒有錯誤內容但子程序失敗時，保留 stderr 供除錯。
	if runError != nil {
		return stockResponse{}, fmt.Errorf("股票資料程式執行失敗：%s", strings.TrimSpace(standardError.String()))
	}
	// 回傳 Python 已整理完成的訊息。
	return response, nil
}

// resolvePythonCommand 依作業系統尋找可用的 Python 3 命令。
func resolvePythonCommand() (pythonCommand, error) {
	// PYTHON_BIN 可讓部署環境明確指定 Python 執行檔。
	if configuredPath := strings.TrimSpace(os.Getenv("PYTHON_BIN")); configuredPath != "" {
		return pythonCommand{path: configuredPath}, nil
	}
	// Windows 同時支援 python、python3 與 py -3 啟動方式。
	candidates := []pythonCommand{{path: "python3"}, {path: "python"}}
	if runtime.GOOS == "windows" {
		candidates = []pythonCommand{{path: "python"}, {path: "python3"}, {path: "py", args: []string{"-3"}}}
	}
	// 依序確認命令是否存在於 PATH。
	for _, candidate := range candidates {
		if path, err := exec.LookPath(candidate.path); err == nil {
			candidate.path = path
			return candidate, nil
		}
	}
	// 所有候選命令都不存在時提示部署方式。
	return pythonCommand{}, fmt.Errorf("找不到 Python 3，請安裝 Python 或設定 PYTHON_BIN")
}
