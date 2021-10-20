package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
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
	sumOfLastMinUsdtVol := float64(0)
	highestVolOfLastMin := float64(0)
	isLast6CandlesGreen := true
	isLastMinNo20k := true
	max := math.Inf(-1)
	min := math.Inf(1)

	for i := 0; i < lenKlines-1; i++ {
		klineVol, err := strconv.ParseFloat(klines[i].Volume, 64)

		if err != nil {
			fmt.Println("ParseFloat volume error:", err)
			return
		}

		klineUsdtVol, err := strconv.ParseFloat(klines[i].QuoteAssetVolume, 64)

		if err != nil {
			fmt.Println("ParseFloat klineUsdtVol error:", err)
			return
		}

		klineOpen, err := strconv.ParseFloat(klines[i].Open, 64)

		if err != nil {
			fmt.Println("ParseFloat klineOpen error:", err)
			return
		}

		klineClose, err := strconv.ParseFloat(klines[i].Close, 64)

		if err != nil {
			fmt.Println("ParseFloat klineClose error:", err)
			return
		}

		if klineClose > max {
			max = klineClose
		}

		if klineClose < min {
			min = klineClose
		}

		sumOfLastMinUsdtVol += klineUsdtVol
		isGreen := klineOpen <= klineClose

		if isGreen && highestVolOfLastMin < klineVol {
			highestVolOfLastMin = klineVol
		}

		isVolMoreThan20k := klineUsdtVol > 20000.0

		if isLastMinNo20k && isVolMoreThan20k {
			isLastMinNo20k = false
		}
	}

	last6MinOpen := float64(0)
	numOfGreen := 0

	for i := lenKlines - 1; -1 < i; i-- {
		klineOpen, err := strconv.ParseFloat(klines[i].Open, 64)

		if err != nil {
			fmt.Println("ParseFloat klineOpen error:", err)
			return
		}

		klineClose, err := strconv.ParseFloat(klines[i].Close, 64)

		if err != nil {
			fmt.Println("ParseFloat klineClose error:", err)
			return
		}

		if klineOpen <= klineClose { // isGreen
			last6MinOpen = klineOpen
			numOfGreen++

			if !isLast6CandlesGreen && i == 55 {
				isLast6CandlesGreen = true
			}
		} else {
			break
		}
	}

	kLast := klines[lenKlines-1]
	buyVol, err := strconv.ParseFloat(kLast.TakerBuyBaseAssetVolume, 64)

	if err != nil {
		fmt.Println("ParseFloat buyVol error:", err)
		return
	}

	klineVol, err := strconv.ParseFloat(kLast.Volume, 64)

	if err != nil {
		fmt.Println("ParseFloat klineVol error:", err)
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

	tStr := time.UnixMilli(kLast.OpenTime).String()[11:16]
	isMoreThan20kUsdt := usdtVol > 20000.0
	isMoreThan500kUsdt := usdtVol >= 500000.0
	isCandleGreen := openPrice < closePrice
	buyPercentage := buyVol / klineVol
	isLast6MinAllGreenAndUpBy2Percent := isLast6CandlesGreen && (closePrice/last6MinOpen) >= 1.017
	isCurrentChange2Percent := (closePrice / openPrice) >= 1.017
	isCurrentVol3xOfLastMin := klineVol/highestVolOfLastMin >= 3
	isCurrent30kAndNo20kFromLastMin := isLastMinNo20k && usdtVol >= 30000.0
	sNoUSDT := s[0 : len(s)-4]
	isYesNo := (sumOfLastMinUsdtVol/60 <= 2000.0 && (max/min) <= 1.015) || numOfGreen >= 10

	if isMoreThan20kUsdt && isCandleGreen {
		label := ""

		if isLast6MinAllGreenAndUpBy2Percent {
			label = "6分"
		} else if isCurrentChange2Percent {
			label = "2%"
		} else if isCurrentVol3xOfLastMin || isCurrent30kAndNo20kFromLastMin {
			label = "3X"
		} else {
			return
		}

		yesNo := "*YES*"

		if !isYesNo {
			yesNo = "NO"
		}

		text := fmt.Sprintf("%s %s %s %.2f %.2f %s", sNoUSDT, label, yesNo, buyPercentage*100, usdtVol, tStr)

		if isMoreThan500kUsdt {
			text = fmt.Sprintf("%s %s %s %.2f *%.2f* %s", sNoUSDT, label, yesNo, buyPercentage*100, usdtVol, tStr)
		}

		channelID, timestamp, err := sc.PostMessage(
			"C01UHA03VEY",
			slack.MsgOptionText(text, false),
		)

		if err != nil {
			fmt.Println("Slack post message error", channelID, timestamp, err)
		}
	}
}

func CheckForSpikes(symbols []interface{}, bc *binance.Client, sc *slack.Client) {
	t := time.Now().Add(time.Duration(-1) * time.Minute).UnixMilli()

	for _, symbol := range symbols {
		s := fmt.Sprintf("%v", symbol)

		if s == "BTCUSDT" || s == "ETHUSDT" {
			continue
		} else {
			SpikeAlert(bc, sc, t, s)
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

	// check right away when starting the program
	CheckForSpikes(symbols, bc, sc)

	c.AddFunc("@every 1m", func() {
		CheckForSpikes(symbols, bc, sc)
	})
	c.Start()

	time.Sleep(24 * 365 * 100 * time.Hour)
}
