package main

import (
	"fmt"

	"github.com/adshao/go-binance/v2"
	"github.com/natoen/duckcoyn/helpers"
)

func main() {
	var (
		binanceApiKey    = "SjtKWLrEyswIwTvbGj4bpUAYLP4LjdZb02aMBcI0xOzMzbOsN17SVUbYH0b9rhMA"
		binanceSecretKey = "13JtnIW1pYLlRm3fWAVY3p6CzCQiwVTgEPZpccQwokClvEVd9VlIbEaiclLTm5H9"
		// slackToken       = "xoxb-1953607810134-2082368693729-5ORkYiqyztdZsQAvijlMquRE"
	)

	bc := binance.NewClient(binanceApiKey, binanceSecretKey)
	// sc := slack.New(slackToken)
	kline := helpers.GetKlines(bc, "CKBUSDT", "1m", 1, 1707782400000)[0]
	fmt.Println(kline.OpenTime == 1707782400000)

}
