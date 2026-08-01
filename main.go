package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	// 農曆日期換算套件，用來取得明年農曆正月初一的國曆日期。
	"github.com/6tail/lunar-go/calendar"
	//算命
	"github.com/bwmarrin/discordgo"
	_ "github.com/joho/godotenv/autoload"
)

const (
	maxNum1  = 10
	maxNum2  = 10
	talkFile = "talk.txt"
	// 動態功能識別名稱必須與 talk.txt 的 action 第三欄一致。
	christmasCountdownAction = "christmas_countdown"
	lunarNewYearAction       = "lunar_new_year"
)

type talkRule struct {
	matchType string
	trigger   string
	reply     string
}

var talkRules []talkRule

func main() {
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

	//Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	//Cleanly close down the Discord session.
	dg.Close()
}
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	//Ignore all messages created by the bot itself
	//This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
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
				executeTalkAction(s, m.ChannelID, rule.reply)
			}
		}
	}
	switch m.Content {
	case "九九乘法":
		printMultiplicationTable(s, m.ChannelID)
	case "算命":
		fortuneTelling(s, m.ChannelID)
	case "現在時間":
		getLocalTime(s, m.ChannelID)
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

		fields := strings.SplitN(line, "\t", 3)
		if len(fields) != 3 {
			return nil, fmt.Errorf("%s line %d must contain match type, trigger and reply separated by tabs", fileName, lineNumber)
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

		if rule.matchType == "action" && rule.reply != christmasCountdownAction && rule.reply != lunarNewYearAction {
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
	case lunarNewYearAction:
		sendNextLunarNewYear(s, channelID)
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
	reply := fmt.Sprintf("🎄 現在時間：%s\n距離 %s 耶誕節還有 %d 天，瑪麗亞凱莉正在解凍！", now.Format("2006-01-02 15:04:05"), christmas.Format("2006-01-02"), days)
	s.ChannelMessageSend(channelID, reply)
}

func sendNextLunarNewYear(s *discordgo.Session, channelID string) {
	//sendNextLunarNewYear 使用當下年份推算明年農曆正月初一的國曆日期與剩餘天數。
	now := time.Now()
	nextYear := now.Year() + 1
	lunarNewYear := calendar.NewLunarFromYmd(nextYear, 1, 1).GetSolar()
	targetDate := time.Date(lunarNewYear.GetYear(), time.Month(lunarNewYear.GetMonth()), lunarNewYear.GetDay(), 0, 0, 0, 0, now.Location())
	days := calendarDaysBetween(now, targetDate)
	reply := fmt.Sprintf("🧧 現在時間：%s\n%d 年農曆正月初一是 %s，距離當天還有 %d 天，劉德華正在解凍！", now.Format("2006-01-02 15:04:05"), nextYear, targetDate.Format("2006-01-02"), days)
	s.ChannelMessageSend(channelID, reply)
}

func calendarDaysBetween(from time.Time, to time.Time) int {
	//calendarDaysBetween 以 UTC 的日期零點計算差距，避免日光節約時間影響天數。
	fromDate := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	toDate := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC)
	return int(toDate.Sub(fromDate) / (24 * time.Hour))
}

func printMultiplicationTable(s *discordgo.Session, channelID string) {
	var sb strings.Builder

	//使用 Discord 程式碼區塊格式，確保等寬字體排版整齊
	sb.WriteString("```\n")

	for i := 1; i < maxNum1; i++ {
		for j := 1; j < maxNum2; j++ {
			//Sprintf 格式化字串並寫入 builder
			sb.WriteString(fmt.Sprintf("%d*%d=%2d  ", j, i, i*j))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("```")

	//發送訊息到 Discord
	s.ChannelMessageSend(channelID, sb.String())
}

func fortuneTelling(s *discordgo.Session, channelID string) {
	answers := []string{
		"大吉！你今天出門會踩到黃金。",
		"大凶……建議你今天不要點開這份程式碼。",
		"諸事不宜，特別是寫 Code，快去睡覺。",
		"看來你今天會被 Bug 狠狠愛上。",
	}
	//隨機選一個索引
	randomIndex := rand.Intn(len(answers))
	s.ChannelMessageSend(channelID, answers[randomIndex])
}

func getLocalTime(s *discordgo.Session, channelID string) {
	//取得當前時間
	now := time.Now()
	timeString := now.Format("2006-01-02 15:04:05")
	reply := fmt.Sprintf("⏰現在的時間是：%s", timeString)
	s.ChannelMessageSend(channelID, reply)
}
