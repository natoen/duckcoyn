package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/adshao/go-binance/v2"
	"github.com/slack-go/slack"
)

func AlertForSpikingCoin(bc *binance.Client, sc *slack.Client, t int64, s string) {
	highestVolOfLastMin := float64(0)
	isLastMinNo20k := true
	// because we will compare the current minute to the last 1 hour
	klines := GetKlines(bc, s, "1m", 61, t)
	lenKlines := len(klines)
	max := math.Inf(-1)
	min := math.Inf(1)
	sumOfLastMinUsdtVol := float64(0)

	for i := 0; i < lenKlines-1; i++ {
		klineVol, err := strconv.ParseFloat(klines[i].Volume, 64)
		fmt.Println(klineVol)
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
	isMoreThan10kUsdt := usdtVol >= 10000.0
	isMoreThan20kUsdt := usdtVol >= 20000.0
	isLast6MinAllGreenAndIsPriceChange1Percent := numOfGreen >= 6 && (closePrice/last6MinOpen) >= 1.015 && isMoreThan10kUsdt
	isCurrentUsdtVol25xOfHighestFromLastMin := klineVol/highestVolOfLastMin >= 3.0 && usdtVol >= 100000.0
	isPriceChange2Percent := (closePrice / openPrice) >= 1.017
	isMoreThan20kUsdtWithConditions := isMoreThan20kUsdt && (isCurrentUsdtVol25xOfHighestFromLastMin || isPriceChange2Percent)

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
		usdtVolByLastMin := (usdtVol / meanUsdtVolOfLastMin)
		timeStr := time.UnixMilli(kLast.OpenTime).String()[11:16]
		text := fmt.Sprintf(strFormat, sNoUSDT, label, usdtVol, buyPercentage*100, usdtVolByLastMin, timeStr)

		channelID, timestamp, err := sc.PostMessage(
			"C01V0V91NTS",
			slack.MsgOptionText(text, false),
		)

		if err != nil {
			fmt.Println("PostMessage", channelID, timestamp, err)
		}
	}
}

func CheckForSpikingCoins(symbols []interface{}, bc *binance.Client, sc *slack.Client) {
	// get the time last minute in millisecond
	t := time.Now().Add(time.Duration(-1) * time.Minute).UnixMilli()

	AlertForSpikingCoin(bc, sc, t, "OGUSDT")
	// for _, symbol := range symbols {
	// 	s := fmt.Sprintf("%v", symbol)
	// 	AlertForSpikingCoin(bc, sc, t, s)
	// }
}

func CheckIfCloseIsHigher(o float64, c float64, p float64) bool {
	// check if the difference of the open and close is higher than the given
	// percentage
	return p <= ((c - o) / o)
}

func CheckIfUsdtVolIsHigher(kv float64, gv float64) bool {
	// check if given USDT volume is higher than kline USDT volume
	return gv <= kv
}

func GetKlines(bc *binance.Client, s string, i string, l int, t int64) []*binance.Kline {
	klines, err := bc.NewKlinesService().
		Symbol(s).
		Interval(i).
		Limit(l).
		EndTime(t).
		Do(context.Background())

	if err != nil {
		fmt.Println("GetKlines error:", err)
		panic(err)
	}

	return klines
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

	excludedSymbols := map[string]bool{
		"BCHSVUSDT": true,
		"TUSDUSDT":  true,
		"BUSDUSDT":  true,
		"ETHUSDT":   true,
		"BTCUSDT":   true,
		"NBTUSDT":   true,
		"USDCUSDT":  true,
		"EUROUSDT":  true,
		"USTUSDT":   true,
		"AUDUSDT":   true,
		"USDPUSDT":  true,
		"EURUSDT":   true,
	}

	var symbols []string
	downStr := "DOWN"
	upStr := "UP"
	usdtStr := "USDT"
	lenUsdt := len(usdtStr)
	lenDownUsdt := len(downStr) + lenUsdt
	lenUpUsdt := len(upStr) + lenUsdt

	for _, p := range prices {
		lenSymbol := len(p.Symbol)
		// check if the length is more than 8 ("DOWNUSDT") or 6 ("UPUSDT")
		// before slicing and comparing itself to "DOWNUSDT" and "UPUSDT"
		isDownToken := lenSymbol > lenDownUsdt && p.Symbol[lenSymbol-lenDownUsdt:] == downStr+usdtStr
		isUpToken := lenSymbol > lenUpUsdt && p.Symbol[lenSymbol-lenUpUsdt:] == upStr+usdtStr
		isNotLeveragedToken := !(isDownToken && isUpToken)
		// filtering only USDT symbols by checking if the last part is "USDT"
		// "TLMUSDT" length is 7 and "USDT" is 4
		// 7 - 4 = 3 so we slice "TLMUSDT" from 3 and we get "USDT"
		isUsdtCoin := p.Symbol[lenSymbol-lenUsdt:] == usdtStr

		if isUsdtCoin && isNotLeveragedToken && !excludedSymbols[p.Symbol] {
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
