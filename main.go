package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/adshao/go-binance/v2"
	"github.com/robfig/cron/v3"
	"github.com/slack-go/slack"
)

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

func SpikeAlert(bc *binance.Client, sc *slack.Client, t int64, s string) {
	klines, err := bc.NewKlinesService().
		Symbol(s).
		Interval("1m").
		Limit(61). // compare the current minute to the last 1 hour
		EndTime(t).
		Do(context.Background())

	if err != nil {
		fmt.Println("NewKlinesService error:", err)
		return
	}

	lenKlines := len(klines)
	sumOfLastMin := float64(0)
	highestVolOfLastMin := float64(0)

	for i := 0; i < lenKlines-1; i++ {
		klineVol, err := strconv.ParseFloat(klines[i].Volume, 64)

		if err != nil {
			fmt.Println("ParseFloat volume error:", err)
			return
		}

		sumOfLastMin += klineVol

		if highestVolOfLastMin < klineVol {
			highestVolOfLastMin = klineVol
		}
	}

	meanVol1Min := sumOfLastMin / float64(60)
	kLast := klines[lenKlines-1]
	buyVol, err := strconv.ParseFloat(kLast.TakerBuyBaseAssetVolume, 64)

	if err != nil {
		fmt.Println("ParseFloat buyVol error:", err)
		return
	}

	cryptoVol, err := strconv.ParseFloat(kLast.Volume, 64)

	if err != nil {
		fmt.Println("ParseFloat cryptoVol error:", err)
		return
	}

	usdtVol, err := strconv.ParseFloat(kLast.QuoteAssetVolume, 64)

	if err != nil {
		fmt.Println("ParseFloat usdtVol error:", err)
		return
	}

	openPrice, err := strconv.ParseFloat(kLast.Open, 64)

	if err != nil {
		fmt.Println("ParseFloat openPrice error:", err)
		return
	}

	closePrice, err := strconv.ParseFloat(kLast.Close, 64)

	if err != nil {
		fmt.Println("ParseFloat closePrice error:", err)
		return
	}

	isMoreThan20kUsdt := usdtVol > 20000.0
	volFromLastMin := (cryptoVol / meanVol1Min)
	isCandleGreen := openPrice < closePrice
	buyPercentage := buyVol / cryptoVol

	if isMoreThan20kUsdt && isCandleGreen {
		text := fmt.Sprintf("%s %.2f", s, buyPercentage*100)
		chanID := ""

		if cryptoVol/highestVolOfLastMin > 3 {
			chanID = "C01V0V91NTS"
		} else if volFromLastMin >= 10 {
			chanID = "C01V0VD0KUG"
		} else if volFromLastMin >= 5 {
			chanID = "C01UPH33NTB"
		} else {
			return
		}

		channelID, timestamp, err := sc.PostMessage(
			chanID,
			slack.MsgOptionText(text, false),
		)

		if err != nil {
			fmt.Println("Slack post message error", channelID, timestamp, err)
		}
	}
}

func main() {
	var (
		binanceApiKey       = "SjtKWLrEyswIwTvbGj4bpUAYLP4LjdZb02aMBcI0xOzMzbOsN17SVUbYH0b9rhMA"
		binanceSecretKey    = "13JtnIW1pYLlRm3fWAVY3p6CzCQiwVTgEPZpccQwokClvEVd9VlIbEaiclLTm5H9"
		slackToken          = "xoxb-1953607810134-2082368693729-5ORkYiqyztdZsQAvijlMquRE"
		usdtSymbolsFilename = "usdt_symbols.txt"
	)

	symbols := ReadUsdtSymbolsFile(usdtSymbolsFilename)
	bc := binance.NewClient(binanceApiKey, binanceSecretKey)
	sc := slack.New(slackToken)
	c := cron.New()

	c.AddFunc("@every 1m", func() {
		t := time.Now().Add(time.Duration(-1) * time.Minute).UnixMilli()

		for _, symbol := range symbols {
			s := fmt.Sprintf("%v", symbol)

			if s == "BTCUSDT" || s == "ETHUSDT" {
				continue
			} else {
				SpikeAlert(bc, sc, t, s)
			}
		}
	})
	c.Start()

	time.Sleep(24 * time.Hour)
}
