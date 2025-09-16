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
		// These are all disabled. Add your own keys here.
		binanceApiKey    = "SjtKWLrEyswIwTvbGj4bpUAYLP4LjdZb02aMBcI0xOzMzbOsN17SVUbYH0b9rhMA"
		binanceSecretKey = "13JtnIW1pYLlRm3fWAVY3p6CzCQiwVTgEPZpccQwokClvEVd9VlIbEaiclLTm5H9"
		slackToken       = "xoxb-1953607810134-2082368693729-5ORkYiqyztdZsQAvijlMquRE"
	)

	bc := binance.NewClient(binanceApiKey, binanceSecretKey)
	sc := slack.New(slackToken)
	c := cron.New()
	pairs := helpers.GetUsdtPairs(bc)
	yesterdayUsdtPairs := helpers.GetYesterdayUsdtPairs(bc, pairs)

	// We store pairs here that we want to skip. 1m means 1 minute. These 3 have
	// different purpose of skipping. They could have better naming.
	skipPair1mMap := sync.Map{}
	skipPair1mMap2 := sync.Map{}
	skipPair1mMap3 := sync.Map{}

	// run every minute
	c.AddFunc("* * * * *", func() {
		t := time.Now().Add(-1 * time.Minute)

		// reset by 9:00 as that is the start of a new day in crypto
		if (t.Hour() == 9) && t.Minute() == 0 {
			skipPair1mMap = sync.Map{}
			skipPair1mMap2 = sync.Map{}
			skipPair1mMap3 = sync.Map{}
			pairs = helpers.GetUsdtPairs(bc)
			yesterdayUsdtPairs = helpers.GetYesterdayUsdtPairs(bc, pairs)
		}

		helpers.CheckForSpikingCoins(yesterdayUsdtPairs, bc, sc, t, &skipPair1mMap2, &skipPair1mMap, &skipPair1mMap3)
	})
	c.Start()

	// run in the background; make the program sleep for 1 year (8760 hours)
	time.Sleep(24 * 365 * time.Hour)
}
