package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/adshao/go-binance/v2"
)

func WriteUsdtSymbolsFile(bc *binance.Client, filename string) {
	// `prices` is an array of prices and symbols key-pair
	prices, err := bc.NewListPricesService().Do(context.Background())

	if err != nil {
		panic(err)
	}

	var symbols []string
	usdtStr := "USDT"
	lenUsdt := len(usdtStr)

	for _, p := range prices {
		lenSymbol := len(p.Symbol)
		// filtering only USDT symbols by checking if the last part is "USDT"
		// "TLMUSDT" length is 7 and "USDT" is 4
		// 7 - 4 = 3 so we slice "TLMUSDT" from 3 and we get "USDT"
		if p.Symbol[lenSymbol-lenUsdt:] == usdtStr && !((lenSymbol > 8) && p.Symbol[lenSymbol-lenUsdt-4:] == "DOWNUSDT" || (lenSymbol > 5) && p.Symbol[lenSymbol-lenUsdt-2:] == "UPUSDT") && p.Symbol != "BCHSVUSDT" && p.Symbol != "TUSDUSDT" && p.Symbol != "BUSDUSDT" && p.Symbol != "ETHUSDT" && p.Symbol != "BTCUSDT" && p.Symbol != "NBTUSDT" && p.Symbol != "USDCUSDT" && p.Symbol != "EUROUSDT" && p.Symbol != "USTUSDT" && p.Symbol != "AUDUSDT" && p.Symbol != "TUSDUSDT" && p.Symbol != "USDPUSDT" && p.Symbol != "EURUSDT" {
			symbols = append(symbols, p.Symbol)
		}
	}

	jsonData, err := json.Marshal(symbols)

	if err != nil {
		panic(err)
	}

	f, err := os.Create(filename)

	if err != nil {
		panic(err)
	}

	b, err := f.WriteString(string(jsonData))

	if err != nil {
		panic(err)
	}

	fmt.Println(b, "bytes written successfully")
	err = f.Close()

	if err != nil {
		panic(err)
	}
}

func main() {
	var (
		binanceApiKey       = "SjtKWLrEyswIwTvbGj4bpUAYLP4LjdZb02aMBcI0xOzMzbOsN17SVUbYH0b9rhMA"
		binanceSecretKey    = "13JtnIW1pYLlRm3fWAVY3p6CzCQiwVTgEPZpccQwokClvEVd9VlIbEaiclLTm5H9"
		usdtSymbolsFilename = "usdt_symbols.txt"
	)

	bc := binance.NewClient(binanceApiKey, binanceSecretKey)

	WriteUsdtSymbolsFile(bc, usdtSymbolsFilename)
}
