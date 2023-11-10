package main

import (
	"github.com/adshao/go-binance/v2"
	"github.com/natoen/duckcoyn/helpers"
	"github.com/slack-go/slack"
)

func main() {
	var (
		binanceApiKey       = "SjtKWLrEyswIwTvbGj4bpUAYLP4LjdZb02aMBcI0xOzMzbOsN17SVUbYH0b9rhMA"
		binanceSecretKey    = "13JtnIW1pYLlRm3fWAVY3p6CzCQiwVTgEPZpccQwokClvEVd9VlIbEaiclLTm5H9"
		slackToken          = "xoxb-1953607810134-2082368693729-5ORkYiqyztdZsQAvijlMquRE"
		usdtSymbolsFilename = "usdt_symbols.txt"
	)

	symbols := helpers.ReadUsdtSymbolsFile(usdtSymbolsFilename)
	bc := binance.NewClient(binanceApiKey, binanceSecretKey)
	sc := slack.New(slackToken)

	helpers.CheckForSpikingCoins(symbols, bc, sc)
}
