package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	//引用 go-cwb/cwb，存取中央氣象署開放資料 API。
	"github.com/bwmarrin/discordgo"
	"github.com/minchao/go-cwb/cwb"
)

const defaultWeatherLocation = "臺北市"

// weatherPeriod 保存同一預報時段的天氣、降雨、溫度與舒適度。
type weatherPeriod struct {
	startTime       string
	endTime         string
	weather         string
	comfort         string
	minTemperature  string
	maxTemperature  string
	rainProbability *int
}

// get36HourWeather 建立 CWA Client 並取得完整的今明 36 小時原始預報資料。
func get36HourWeather(ctx context.Context, apiKey string, locationName string) (*cwb.Forecast36HourWeather, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, fmt.Errorf("CWA_API 不可為空")
	}

	//將 API Key 字串轉成可呼叫 Forecasts 服務的 CWA Client。
	client := cwb.NewClient(apiKey, nil)
	forecast, _, err := client.Forecasts.Get36HourWeather(
		ctx,
		[]string{locationName},
		[]string{"Wx", "PoP", "CI", "MinT", "MaxT"},
	)
	if err != nil {
		return nil, fmt.Errorf("取得 36 小時天氣預報失敗：%w", err)
	}
	return forecast, nil
}

// configuredWeatherLocation 讀取預設縣市，未設定時使用臺北市。
func configuredWeatherLocation() string {
	locationName := strings.TrimSpace(os.Getenv("CWA_LOCATION"))
	if locationName == "" {
		return defaultWeatherLocation
	}
	return locationName
}

// loadWeatherPeriods 查詢指定縣市並將不同氣象因子合併為預報時段。
func loadWeatherPeriods(ctx context.Context) (string, []weatherPeriod, error) {
	locationName := configuredWeatherLocation()
	forecast, err := get36HourWeather(ctx, os.Getenv("CWA_API"), locationName)
	if err != nil {
		return "", nil, err
	}
	if len(forecast.Records.Location) == 0 {
		return "", nil, fmt.Errorf("找不到縣市 %q 的 36 小時預報", locationName)
	}

	location := forecast.Records.Location[0]
	periodByTime := make(map[string]*weatherPeriod)
	for _, element := range location.WeatherElement {
		for _, forecastTime := range element.Time {
			key := forecastTime.StartTime + "\x00" + forecastTime.EndTime
			period, exists := periodByTime[key]
			if !exists {
				period = &weatherPeriod{
					startTime: forecastTime.StartTime,
					endTime:   forecastTime.EndTime,
				}
				periodByTime[key] = period
			}

			value := forecastTime.Parameter.ParameterName
			switch element.ElementName {
			case "Wx":
				period.weather = value
			case "PoP":
				probability, parseErr := strconv.Atoi(value)
				if parseErr == nil {
					period.rainProbability = &probability
				}
			case "CI":
				period.comfort = value
			case "MinT":
				period.minTemperature = value
			case "MaxT":
				period.maxTemperature = value
			}
		}
	}

	periods := make([]weatherPeriod, 0, len(periodByTime))
	for _, period := range periodByTime {
		periods = append(periods, *period)
	}
	sort.Slice(periods, func(i, j int) bool {
		return periods[i].startTime < periods[j].startTime
	})
	if len(periods) == 0 {
		return "", nil, fmt.Errorf("縣市 %q 沒有可用的預報時段", location.LocationName)
	}

	return location.LocationName, periods, nil
}

// send36HourWeather 取得並回覆指定縣市最近 36 小時的完整天氣預報。
func send36HourWeather(s *discordgo.Session, channelID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	locationName, periods, err := loadWeatherPeriods(ctx)
	if err != nil {
		s.ChannelMessageSend(channelID, fmt.Sprintf("目前無法取得 36 小時天氣預報：%v", err))
		return
	}

	var reply strings.Builder
	fmt.Fprintf(&reply, "🌤️ %s未來 36 小時天氣\n", locationName)
	for _, period := range periods {
		fmt.Fprintf(
			&reply,
			"\n%s～%s\n%s｜降雨 %s｜%s～%s°C｜%s\n",
			formatForecastTime(period.startTime),
			formatForecastTime(period.endTime),
			fallbackWeatherValue(period.weather),
			formatRainProbability(period.rainProbability),
			fallbackWeatherValue(period.minTemperature),
			fallbackWeatherValue(period.maxTemperature),
			fallbackWeatherValue(period.comfort),
		)
	}
	reply.WriteString("\n資料來源：中央氣象署 F-C0032-001")
	s.ChannelMessageSend(channelID, reply.String())
}

// rainProbabilityAction 讀取未來 36 小時降雨機率並以隨機口語字庫回覆。
func rainProbabilityAction(s *discordgo.Session, channelID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	locationName, periods, err := loadWeatherPeriods(ctx)
	if err != nil {
		s.ChannelMessageSend(channelID, fmt.Sprintf("目前無法取得降雨機率：%v", err))
		return
	}

	var highestProbability *int
	var highestPeriod weatherPeriod
	for _, period := range periods {
		if period.rainProbability == nil {
			continue
		}
		if highestProbability == nil || *period.rainProbability > *highestProbability {
			probability := *period.rainProbability
			highestProbability = &probability
			highestPeriod = period
		}
	}
	if highestProbability == nil {
		s.ChannelMessageSend(channelID, fmt.Sprintf("目前找不到%s未來 36 小時的降雨機率。", locationName))
		return
	}

	var reply strings.Builder
	fmt.Fprintf(
		&reply,
		"🌧️ %s未來 36 小時最高降雨機率為 %d%%\n時段：%s～%s\n%s\n",
		locationName,
		*highestProbability,
		formatForecastTime(highestPeriod.startTime),
		formatForecastTime(highestPeriod.endTime),
		rainProbabilityMessage(*highestProbability),
	)
	reply.WriteString("\n各時段降雨機率：")
	for _, period := range periods {
		fmt.Fprintf(
			&reply,
			"\n%s～%s：%s",
			formatForecastTime(period.startTime),
			formatForecastTime(period.endTime),
			formatRainProbability(period.rainProbability),
		)
	}
	reply.WriteString("\n\n資料來源：中央氣象署 F-C0032-001")
	s.ChannelMessageSend(channelID, reply.String())
}

// rainProbabilityMessage 依降雨機率區間選取字庫，再隨機回傳其中一句。
func rainProbabilityMessage(probability int) string {
	var messages []string
	switch {
	case probability <= 10:
		messages = []string{
			"天空看起來很給面子，今天大致不用擔心雨來亂。",
			"雨神今天應該在休假，輕裝出門就可以了。",
			"下雨機會很低，雨傘可以先留在家裡顧門。",
		}
	case probability <= 30:
		messages = []string{
			"下雨機會不高，但帶把輕便傘會比較安心。",
			"大多時候應該是乾的，偶爾可能有幾滴來打招呼。",
			"雨勢出現的機會偏低，怕麻煩可以帶把折傘備用。",
		}
	case probability <= 50:
		messages = []string{
			"天空有點猶豫，帶傘出門比較不會後悔。",
			"下不下雨差不多五五波，建議不要跟天氣賭。",
			"雨可能突然插隊，包包裡放把傘比較保險。",
		}
	case probability <= 70:
		messages = []string{
			"雨來報到的機會不小，出門記得把傘帶上。",
			"看來雲層有自己的計畫，今天最好準備雨具。",
			"這個機率不太適合鐵齒，雨傘請務必同行。",
		}
	case probability <= 90:
		messages = []string{
			"大概率會下雨，雨傘和防水鞋都可以準備了。",
			"雨神已經在路上，今天別想空手出門。",
			"幾乎躲不掉一場雨，行程最好預留避雨時間。",
		}
	default:
		messages = []string{
			"雨幾乎確定會來，請把自己和重要物品都顧好。",
			"天空已經把下雨排進行程，完整雨具直接帶上。",
			"這不是會不會下的問題，是什麼時候開始下。",
		}
	}
	return messages[rand.Intn(len(messages))]
}

// formatForecastTime 將中央氣象署時間轉成適合 Discord 閱讀的格式。
func formatForecastTime(value string) string {
	for _, layout := range []string{"2006-01-02 15:04:05", time.RFC3339} {
		parsedTime, err := time.Parse(layout, value)
		if err == nil {
			return parsedTime.Format("01/02 15:04")
		}
	}
	return value
}

// formatRainProbability 將降雨機率轉成百分比，缺少資料時顯示未知。
func formatRainProbability(probability *int) string {
	if probability == nil {
		return "未知"
	}
	return fmt.Sprintf("%d%%", *probability)
}

// fallbackWeatherValue 將缺少的氣象欄位統一顯示為未知。
func fallbackWeatherValue(value string) string {
	if strings.TrimSpace(value) == "" {
		return "未知"
	}
	return value
}
