package main

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/adshao/go-binance/v2"
)

func main() {
	var (
		binanceApiKey    = "SjtKWLrEyswIwTvbGj4bpUAYLP4LjdZb02aMBcI0xOzMzbOsN17SVUbYH0b9rhMA"
		binanceSecretKey = "13JtnIW1pYLlRm3fWAVY3p6CzCQiwVTgEPZpccQwokClvEVd9VlIbEaiclLTm5H9"
	)

	bc := binance.NewClient(binanceApiKey, binanceSecretKey)
	loc, err := time.LoadLocation("Asia/Tokyo")

	if err != nil {
		fmt.Println("LoadLocation error:", err)
		return
	}

	s := "DOCKUSDT"
	t := time.Date(2021, 11, 7, 20, 00, 0, 0, loc).Add(time.Duration(-1) * time.Minute).UnixMilli()
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

	isCandleGreen := openPrice <= closePrice
	isMoreThan20kUsdt := usdtVol >= 20000.0
	isLast6MinAllGreenAndIsPriceChange1Percent := numOfGreen >= 6 && (closePrice/last6MinOpen) >= 1.015
	isCurrentUsdtVol25xOfHighestFromLastMin := klineVol/highestVolOfLastMin >= 3.0 && usdtVol >= 100000.0
	isPriceChange2Percent := (closePrice / openPrice) >= 1.017
	isMoreThan20kUsdtWithConditions := isMoreThan20kUsdt && (isCurrentUsdtVol25xOfHighestFromLastMin || isPriceChange2Percent)
	fmt.Println(isCandleGreen, usdtVol, isPriceChange2Percent)
	if isCandleGreen && (isLast6MinAllGreenAndIsPriceChange1Percent || isMoreThan20kUsdtWithConditions) {
		label := ""

		if isLast6MinAllGreenAndIsPriceChange1Percent {
			label = fmt.Sprintf("%d分", numOfGreen)
		} else if isPriceChange2Percent {
			label = "2%"
		} else {
			label = "3X"
		}

		strFormat := "%s %s %.2f %.2f %.2f %s"
		isMoreThan400kUsdt := usdtVol >= 400000.0

		if isMoreThan400kUsdt {
			strFormat = "%s %s *%.2f* %.2f %.2f %s"
		}

		sNoUSDT := s[0 : len(s)-4]
		buyPercentage := buyVol / klineVol
		meanUsdtVolOfLastMin := sumOfLastMinUsdtVol / 60
		currentUsdtVolByLastMin := (usdtVol / meanUsdtVolOfLastMin)
		timeStr := time.UnixMilli(kLast.OpenTime).String()[11:16]
		text := fmt.Sprintf(strFormat, sNoUSDT, label, usdtVol, buyPercentage*100, currentUsdtVolByLastMin, timeStr)

		fmt.Println(text)
	}
}
