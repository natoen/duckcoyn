package main

import (
	"fmt"
	"time"

	"github.com/adshao/go-binance/v2"
	"github.com/natoen/duckcoyn/helpers"
	"github.com/robfig/cron/v3"
	"github.com/slack-go/slack"
)

func main() {
	var (
		binanceApiKey    = "SjtKWLrEyswIwTvbGj4bpUAYLP4LjdZb02aMBcI0xOzMzbOsN17SVUbYH0b9rhMA"
		binanceSecretKey = "13JtnIW1pYLlRm3fWAVY3p6CzCQiwVTgEPZpccQwokClvEVd9VlIbEaiclLTm5H9"
		slackToken       = "xoxb-1953607810134-2082368693729-5ORkYiqyztdZsQAvijlMquRE"
	)

	bc := binance.NewClient(binanceApiKey, binanceSecretKey)
	sc := slack.New(slackToken)
	c := cron.New()
	pairs := helpers.GetUsdtPairs(bc)
	yesterdayUsdtPairs := helpers.GetYesterdayUsdtPairs(bc, pairs)

	fmt.Println(yesterdayUsdtPairs, "yesterday USDT pairs")

	// run every minute
	c.AddFunc("* * * * *", func() {
		t := time.Now().Add(-1 * time.Minute)

		if t.Hour() == 0 && t.Minute() == 0 {
			pairs = helpers.GetUsdtPairs(bc)
			yesterdayUsdtPairs = helpers.GetYesterdayUsdtPairs(bc, pairs)

			fmt.Println(yesterdayUsdtPairs, "yesterday USDT pairs")
		}

		helpers.CheckForSpikingCoins(pairs, yesterdayUsdtPairs, bc, sc, t)
	})
	c.Start()

	// make the program sleep for 1 year (8760 hours)
	time.Sleep(24 * 365 * time.Hour)
}
