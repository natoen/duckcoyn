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

	isMoreThan20kUsdt := usdtVol > 20000.0
	isMoreThan400kUsdt := usdtVol >= 400000.0
	isCandleGreen := openPrice < closePrice
	buyPercentage := buyVol / klineVol
	isLast6MinAllGreen := numOfGreen >= 6 && (closePrice/last6MinOpen) >= 1.01
	isCurrentChange2Percent := (closePrice / openPrice) >= 1.017
	isCurrentVol3xOfLastMin := klineVol/highestVolOfLastMin >= 3
	isCurrent30kAndNo20kFromLastMin := isLastMinNo20k && usdtVol >= 30000.0
	sNoUSDT := s[0 : len(s)-4]
	meanUsdtVolOfLastMin := sumOfLastMinUsdtVol / 60
	currentUsdtVolByLastMin := (usdtVol / meanUsdtVolOfLastMin)
	isCurrentUsdtVol25xOfLastMin := currentUsdtVolByLastMin >= 25.0
	isFlat := (max/min <= 1.015)
	isYesNo := (isFlat && isCurrentUsdtVol25xOfLastMin) || isLast6MinAllGreen || isCurrentChange2Percent || isMoreThan400kUsdt
	timeStr := time.UnixMilli(kLast.OpenTime).String()[11:16]

	if (isMoreThan20kUsdt || isLast6MinAllGreen) && isCandleGreen {
		label := ""

		if isLast6MinAllGreen {
			label = fmt.Sprintf("%d分", numOfGreen)
		} else if isCurrentChange2Percent {
			label = "2%"
		} else if isCurrentVol3xOfLastMin || isCurrent30kAndNo20kFromLastMin {
			label = "3X"
		} else {
			return
		}

		text := fmt.Sprintf("%s %s %.2f %.2f %.2f %s", sNoUSDT, label, buyPercentage*100, usdtVol, currentUsdtVolByLastMin, timeStr)

		if isMoreThan400kUsdt {
			text = fmt.Sprintf("%s %s %.2f *%.2f* %.2f %s", sNoUSDT, label, buyPercentage*100, usdtVol, currentUsdtVolByLastMin, timeStr)
		}

		chanID := "C01UHA03VEY"

		if isYesNo {
			chanID = "C01V0V91NTS"
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
