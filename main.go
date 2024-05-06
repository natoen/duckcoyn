package main

import (
	"sync"
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
	skipPairsMapDay := sync.Map{}

	// run every minute
	c.AddFunc("* * * * *", func() {
		t := time.Now().Add(-1 * time.Minute)

		if (t.Hour() == 9) && t.Minute() == 0 {
			skipPairsMapDay = sync.Map{}
		}

		if t.Hour() == 9 && t.Minute() == 0 {
			pairs = helpers.GetUsdtPairs(bc)
			yesterdayUsdtPairs = helpers.GetYesterdayUsdtPairs(bc, pairs)
		}

		helpers.CheckForSpikingCoins(yesterdayUsdtPairs, bc, sc, t, &skipPairsMapDay)
	})
	c.Start()

	// make the program sleep for 1 year (8760 hours)
	time.Sleep(24 * 365 * time.Hour)
}
