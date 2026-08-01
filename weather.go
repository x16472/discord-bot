package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	//引用 go-cwb/cwb，存取中央氣象署開放資料 API。
	"github.com/bwmarrin/discordgo"
	"github.com/minchao/go-cwb/cwb"
)

// get36HourWeather 建立 CWA Client 並取得完整的今明 36 小時原始預報資料。
func get36HourWeather(ctx context.Context, apiKey string) (*cwb.Forecast36HourWeather, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, fmt.Errorf("CWA_API 不可為空")
	}

	//將 API Key 字串轉成可呼叫 Forecasts 服務的 CWA Client。
	client := cwb.NewClient(apiKey, nil)
	forecast, _, err := client.Forecasts.Get36HourWeather(ctx, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("取得 36 小時天氣預報失敗：%w", err)
	}
	return forecast, nil
}

// send36HourWeather 取得 36 小時預報，並回覆目前取得的資料範圍。
func send36HourWeather(s *discordgo.Session, channelID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	forecast, err := get36HourWeather(ctx, os.Getenv("CWA_API"))
	if err != nil {
		s.ChannelMessageSend(channelID, fmt.Sprintf("目前無法取得 36 小時天氣預報：%v", err))
		return
	}

	reply := fmt.Sprintf(
		"🌤️ %s\n已取得 %d 個縣市的預報資料。",
		forecast.Records.DatasetDescription,
		len(forecast.Records.Location),
	)
	s.ChannelMessageSend(channelID, reply)
}
