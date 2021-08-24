package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/adshao/go-binance/v2"
	"github.com/slack-go/slack"
)

/*
	When the app starts:
		1. Get all the USDT pairs from Binance and save it from the database.
		2. Get all the pairs from the database and loop through all of them.
		3. Check the latest 1 minute candle record of that particular pair in the database. If there is no record in the database, get the oldest record from Binance and save it to the database.
		4. Get the next record
*/

func RunEveryMinute(bc *binance.Client, sc *slack.Client) {
	channelID, timestamp, err := sc.PostMessage(
		"C01UPH33NTB",
		slack.MsgOptionText("Some text", false),
	)

	if err != nil {
		fmt.Printf("%s\n", err)
		fmt.Printf("%sSlack Channel ID:\n", channelID)
	}

	fmt.Printf("Message successfully sent to channel %s at %s\n", channelID, timestamp)

	klines, err := bc.NewKlinesService().
		Symbol("TLMUSDT").
		Interval("1m").
		Limit(15).
		Do(context.Background())

	if err != nil {
		fmt.Println(err)
		return
	}
	for _, k := range klines {
		fmt.Println(k.Volume, k.QuoteAssetVolume, k.TakerBuyBaseAssetVolume, k.Open, k.Close)
	}
	// 1624211221464486000
	// 1624210500000
	// 1624210518
	fmt.Println(time.Now().Unix())
	fmt.Println(time.Now().UnixNano())
}

func ReadUsdtSymbolsFile(filename string) []interface{} {
	data, err := os.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	var decodedData []interface{}

	err = json.Unmarshal(data, &decodedData)
	if err != nil {
		panic(err)
	}

	return decodedData
}

func WriteUsdtSymbolsFile(bc *binance.Client, filename string) {
	// `prices` is an array of prices and symbols key-pair
	prices, err := bc.NewListPricesService().Do(context.Background())
	if err != nil {
		panic(err)
	}

	var symbols []string
	usdt := "USDT"
	lenUsdt := len(usdt)

	for _, p := range prices {
		lenSymbol := len(p.Symbol)

		// filtering only USDT symbols by checking if the last part is "USDT"
		// "TLMUSDT" length is 7 and "USDT" is 4
		// 7 - 4 = 3 so we slice "TLMUSDT" from 3 and we get "USDT"
		if p.Symbol[lenSymbol-lenUsdt:] == usdt {
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

func DateToMilliseconds(year, month, day, hour, minute, int) int {

}

func main() {
	// var (
	// 	binanceApiKey       = "SjtKWLrEyswIwTvbGj4bpUAYLP4LjdZb02aMBcI0xOzMzbOsN17SVUbYH0b9rhMA"
	// 	binanceSecretKey    = "13JtnIW1pYLlRm3fWAVY3p6CzCQiwVTgEPZpccQwokClvEVd9VlIbEaiclLTm5H9"
	// 	slackToken          = "xoxb-1953607810134-2082368693729-5ORkYiqyztdZsQAvijlMquRE"
	// 	usdtSymbolsFilename = "usdt_symbols.txt"
	// )

	// bc := binance.NewClient(binanceApiKey, binanceSecretKey)
	// sc := slack.New(slackToken)

	// WriteUsdtSymbolsFile(bc, usdtSymbolsFilename)

	// c := cron.New()
	// c.AddFunc("@every 1m", func() {
	// 	RunEveryMinute(bc, sc)
	// })
	// c.Start()

	// time.Sleep(8760 * time.Hour)
	fmt.Println()
}
