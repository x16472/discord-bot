package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	//農曆日期換算套件，用來取得明年農曆正月初一的國曆日期。
	"github.com/6tail/lunar-go/calendar"

	"github.com/bwmarrin/discordgo"
	_ "github.com/joho/godotenv/autoload"
)

const (
	talkFile = "talk.txt"
	//動態功能識別名稱必須與 talk.txt 的 action 第三欄一致。
	christmasCountdownAction  = "christmas_countdown"
	localTimeAction           = "local_time"
	lunarNewYearAction        = "lunar_new_year"
	rainProbabilityActionName = "rainProbability_action"
	dailyMarketActionName     = "daily_market"
	stockPriceActionName      = "stock_price"
	stockSuggestionActionName = "stock_suggestion"
	weather36HourAction       = "weather_36h"
)

type talkRule struct {
	matchType string
	trigger   string
	reply     string
}

var talkRules []talkRule

func main() {
	//載入文字規則；降雨口語字庫由 Go 1.20 以上版本自動初始化亂數來源。
	var err error
	talkRules, err = loadTalkRules(talkFile)
	if err != nil {
		fmt.Println("error loading talk rules,", err)
		return
	}
	token := os.Getenv("DCToken")

	//creates a new Discord session
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	//Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)

	//只監聽訊息
	dg.Identify.Intents = discordgo.IntentsGuildMessages

	//開啟連線
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	//WebSocket 連線完成後，設定 Discord Bot 的遊戲動態資訊。
	if err := updateBotGameStatus(dg); err != nil {
		fmt.Println("error updating game status,", err)
	}

	//Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	//Cleanly close down the Discord session.
	dg.Close()
}

// updateBotGameStatus 將 Bot 顯示為正在遊玩
func updateBotGameStatus(s *discordgo.Session) error {
	return s.UpdateStatusComplex(discordgo.UpdateStatusData{
		Status: "online",
		AFK:    false,
		Activities: []*discordgo.Activity{
			{
				Name:    "Enjoying Golang",
				Type:    discordgo.ActivityTypeListening,
				Details: "正在探索Golang",
				State:   "正在Golang中撰寫 Discord Bot",
				Timestamps: discordgo.TimeStamps{
					StartTimestamp: time.Now().UnixMilli(),
				},
			},
		},
	})
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	//Ignore all messages created by the bot itself
	//This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}

	//解析帶股票代號的訊息事件，只有符合指令時才建立非同步查詢工作。
	if command, matched := parseStockCommand(m.Content); matched {
		if command.errorMessage != "" {
			s.ChannelMessageSend(m.ChannelID, command.errorMessage)
			return
		}
		go executeStockEvent(s, m.ChannelID, command)
		return
	}

	// 根據 talk.txt 的比對方式執行一般回覆或動態日期指令。
	for _, rule := range talkRules {
		switch rule.matchType {
		case "exact":
			if m.Content == rule.trigger {
				s.ChannelMessageSend(m.ChannelID, rule.reply)
			}
		case "contains":
			if strings.Contains(m.Content, rule.trigger) {
				s.ChannelMessageSend(m.ChannelID, rule.reply)
			}
		case "action":
			if m.Content == rule.trigger {
				//動態 action 使用背景工作執行，避免外部資料查詢阻塞其他訊息事件。
				go executeTalkAction(s, m.ChannelID, rule.reply)
				return
			}
		}
	}
}

// executeStockEvent 依 messageCreate 解析出的事件執行 stock.go 股票功能。
func executeStockEvent(s *discordgo.Session, channelID string, command stockCommand) {
	switch command.action {
	case stockPriceActionName:
		stockPriceAction(s, channelID, command.stockNumber)
	case stockSuggestionActionName:
		stockSuggestionAction(s, channelID, command.stockNumber)
	}
}

func loadTalkRules(fileName string) ([]talkRule, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	rules := make([]talkRule, 0)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(strings.TrimPrefix(scanner.Text(), "\ufeff"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		//每筆規則只使用前兩個半形減號切割，回覆內容可繼續包含減號。
		fields := strings.SplitN(line, "-", 3)
		if len(fields) != 3 {
			return nil, fmt.Errorf("%s line %d must contain match type, trigger and reply separated by hyphens", fileName, lineNumber)
		}

		rule := talkRule{
			matchType: strings.TrimSpace(fields[0]),
			trigger:   strings.TrimSpace(fields[1]),
			reply:     strings.TrimSpace(fields[2]),
		}

		if rule.matchType != "exact" && rule.matchType != "contains" && rule.matchType != "action" {
			//action 類型會把第三欄當成 main.go 中的動態功能識別名稱。
			return nil, fmt.Errorf("%s line %d has unsupported match type %q", fileName, lineNumber, rule.matchType)
		}
		if rule.trigger == "" || rule.reply == "" {
			return nil, fmt.Errorf("%s line %d has an empty trigger or reply", fileName, lineNumber)
		}

		if rule.matchType == "action" && rule.reply != christmasCountdownAction && rule.reply != localTimeAction && rule.reply != lunarNewYearAction && rule.reply != weather36HourAction && rule.reply != rainProbabilityActionName && rule.reply != dailyMarketActionName && rule.reply != stockPriceActionName && rule.reply != stockSuggestionActionName {
			//啟動時先驗證動態功能名稱，避免輸入指令後沒有任何回覆。
			return nil, fmt.Errorf("%s line %d has unsupported action %q", fileName, lineNumber, rule.reply)
		}
		rules = append(rules, rule)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return rules, nil
}

func executeTalkAction(s *discordgo.Session, channelID string, action string) {
	//executeTalkAction 依照 talk.txt 第三欄指定的功能產生動態回覆。
	switch action {
	case christmasCountdownAction:
		sendChristmasCountdown(s, channelID)
	case localTimeAction:
		getLocalTime(s, channelID)
	case lunarNewYearAction:
		sendNextLunarNewYear(s, channelID)
	case weather36HourAction:
		send36HourWeather(s, channelID)
	case rainProbabilityActionName:
		rainProbabilityAction(s, channelID)
	case dailyMarketActionName:
		dailyMarketAction(s, channelID)
	case stockPriceActionName:
		stockPriceAction(s, channelID, defaultStockNumber)
	case stockSuggestionActionName:
		stockSuggestionAction(s, channelID, defaultStockNumber)
	}
}

func sendChristmasCountdown(s *discordgo.Session, channelID string) {
	//sendChristmasCountdown使用當下時間計算距離下一個耶誕節的日曆天數。
	now := time.Now()
	christmas := time.Date(now.Year(), time.December, 25, 0, 0, 0, 0, now.Location())
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if today.After(christmas) {
		christmas = time.Date(now.Year()+1, time.December, 25, 0, 0, 0, 0, now.Location())
	}

	days := calendarDaysBetween(now, christmas)
	reply := fmt.Sprintf("🎄 現在時間：%s\n距離 %s 耶誕節還有 %d 天！", now.Format("2006-01-02 15:04:05"), christmas.Format("2006-01-02"), days)
	s.ChannelMessageSend(channelID, reply)
}

func sendNextLunarNewYear(s *discordgo.Session, channelID string) {
	//sendNextLunarNewYear 使用當下年份推算明年農曆正月初一的國曆日期與剩餘天數。
	now := time.Now()
	nextYear := now.Year() + 1
	lunarNewYear := calendar.NewLunarFromYmd(nextYear, 1, 1).GetSolar()
	targetDate := time.Date(lunarNewYear.GetYear(), time.Month(lunarNewYear.GetMonth()), lunarNewYear.GetDay(), 0, 0, 0, 0, now.Location())
	days := calendarDaysBetween(now, targetDate)
	reply := fmt.Sprintf("🧧 現在時間：%s\n%d年農曆正月初一是 %s，距離當天還有 %d 天！", now.Format("2006-01-02 15:04:05"), nextYear, targetDate.Format("2006-01-02"), days)
	s.ChannelMessageSend(channelID, reply)
}

func calendarDaysBetween(from time.Time, to time.Time) int {
	//calendarDaysBetween 以 UTC 的日期零點計算差距，避免日光節約時間影響天數。
	fromDate := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	toDate := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC)
	return int(toDate.Sub(fromDate) / (24 * time.Hour))
}

func getLocalTime(s *discordgo.Session, channelID string) {
	//取得當前時間
	now := time.Now()
	timeString := now.Format("2006/01/02 15:04:05")
	reply := fmt.Sprintf("⏰現在的時間是：%s", timeString)
	s.ChannelMessageSend(channelID, reply)
}
