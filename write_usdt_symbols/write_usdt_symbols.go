package main

import (
	"github.com/adshao/go-binance/v2"
	"github.com/natoen/duckcoyn/helpers"
)

func main() {
	var (
		binanceApiKey       = "SjtKWLrEyswIwTvbGj4bpUAYLP4LjdZb02aMBcI0xOzMzbOsN17SVUbYH0b9rhMA"
		binanceSecretKey    = "13JtnIW1pYLlRm3fWAVY3p6CzCQiwVTgEPZpccQwokClvEVd9VlIbEaiclLTm5H9"
		usdtSymbolsFilename = "usdt_symbols.txt"
	)

	bc := binance.NewClient(binanceApiKey, binanceSecretKey)

	helpers.WriteUsdtSymbolsFile(bc, usdtSymbolsFilename)
}
