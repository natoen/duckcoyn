package helpers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/adshao/go-binance/v2"
	"github.com/slack-go/slack"
)

var wg sync.WaitGroup

func CheckForSpikingCoins(yesterdayUsdtPairs map[string]float64, bc *binance.Client, sc *slack.Client, t time.Time, sp *sync.Map, spd *sync.Map) {
	numOfKlines := 240
	indexOfLastKline := numOfKlines - 1

	for pair := range yesterdayUsdtPairs {
		_, isSkipPair := sp.Load(pair)
		_, isSkipPairDay := spd.Load(pair)

		if isSkipPair || isSkipPairDay {
			continue
		}

		wg.Add(1)

		go func(pair string) {
			klines := GetKlines(bc, pair, "1m", numOfKlines, t.UnixMilli())
			latestKline := klines[indexOfLastKline]

			latestKlineClose, err := strconv.ParseFloat(latestKline.Close, 64)
			if err != nil {
				fmt.Println("ParseFloat latestKlineClose error:", latestKline, err)
				return
			}

			if isAHigherKlineOpenExists(indexOfLastKline, klines, latestKlineClose) {
				defer wg.Done()
				return
			}

			latestKlineOpen, err := strconv.ParseFloat(latestKline.Open, 64)
			if err != nil {
				fmt.Println("ParseFloat latestKlineOpen error:", latestKline, err)
				return
			}

			isGreen := latestKlineOpen <= latestKlineClose

			if !isGreen {
				defer wg.Done()
				return
			}

			latestKlineUsdtVol, err := strconv.ParseFloat(latestKline.QuoteAssetVolume, 64)
			if err != nil {
				fmt.Println("ParseFloat latestKlineUsdtVol error:", latestKline, err)
				return
			}

			percentUp := (latestKlineClose / latestKlineOpen)
			is09PercentUp := percentUp >= 1.009
			yesterdayUsdtVol := yesterdayUsdtPairs[pair]
			yesterdayTodayUsdtVolRate := latestKlineUsdtVol / yesterdayUsdtVol
			yesterdayUsdtVolPercentage := yesterdayTodayUsdtVolRate * 100
			isUsdtVol4PercentOfYesterday := (yesterdayTodayUsdtVolRate >= 0.04) && (latestKlineUsdtVol >= 40000.0)
			coinName := pair[0 : len(pair)-4]

			if !isUsdtVol4PercentOfYesterday && !is09PercentUp {
				defer wg.Done()
				return
			}

			channelID := "C01V0V91NTS"

			if isSurging15Min(indexOfLastKline, klines, yesterdayUsdtVol) {
				channelID = "C01UHA03VEY"
				spd.Store(pair, latestKlineUsdtVol)
			} else {
				sp.Store(pair, latestKlineUsdtVol)
			}

			message := fmt.Sprintf("<https://www.binance.com/en/trade/%s_USDT?type=spot|%s> %.0f%% %s %s %.2f%% %s", coinName, coinName, yesterdayUsdtVolPercentage, numShortener(latestKlineUsdtVol), numShortener(yesterdayUsdtVol), (percentUp-1)*100, t.String()[11:16])
			postSlackMessage(sc, channelID, message)
			defer wg.Done()
		}(pair)
	}

	wg.Wait()
}

func GetKlines(bc *binance.Client, s string, i string, l int, et int64) []*binance.Kline {
	var klines []*binance.Kline
	var err error
	isGetKlineNotSuccess := true

	for isGetKlineNotSuccess {
		klines, err = bc.NewKlinesService().
			Symbol(s).
			Interval(i).
			Limit(l).
			EndTime(et).
			Do(context.Background())

		if err != nil {
			fmt.Println("GetKlines error:", s, err)
			time.Sleep(5 * time.Second)
		} else {
			isGetKlineNotSuccess = false
		}
	}

	return klines
}

func GetUsdtPairs(bc *binance.Client) []string {
	// `prices` is an array of prices and symbols key-pair
	prices, err := bc.NewListPricesService().Do(context.Background())

	if err != nil {
		panic(err)
	}

	excludedSymbols := map[string]bool{
		"BCHSVUSDT": true,
		"TUSDUSDT":  true,
		"BUSDUSDT":  true,
		"NBTUSDT":   true,
		"USDCUSDT":  true,
		"EUROUSDT":  true,
		"USTUSDT":   true,
		"AUDUSDT":   true,
		"USDPUSDT":  true,
		"EURUSDT":   true,
		"TVKUSDT":   true,
		"ERDUSDT":   true,
		"LENDUSDT":  true,
		"WBTCUSDT":  true,
		"BCCUSDT":   true,
	}

	var symbols []string

	for _, p := range prices {
		isDownToken := strings.Contains(p.Symbol, "DOWN")
		isUpToken := strings.Contains(p.Symbol, "UP")
		isBullToken := strings.Contains(p.Symbol, "BULL")
		isBearToken := strings.Contains(p.Symbol, "BEAR")
		isNotLeverageToken := (!isDownToken && !isUpToken && !isBullToken && !isBearToken)
		symbolLen := len(p.Symbol)
		isUsdtPair := p.Symbol[symbolLen-4:symbolLen] == "USDT"

		if isUsdtPair && isNotLeverageToken && !excludedSymbols[p.Symbol] {
			symbols = append(symbols, p.Symbol)
		}
	}

	return symbols
}

func GetYesterdayUsdtPairs(bc *binance.Client, pairs []string) map[string]float64 {
	yesterday := time.Now().Add(-24 * time.Hour).UnixMilli()
	m := &sync.Map{}

	for _, pair := range pairs {
		p := fmt.Sprintf("%v", pair)
		wg.Add(1)

		go func(pair string) {
			klines := GetKlines(bc, pair, "1d", 1, yesterday)

			if len(klines) == 0 {
				defer wg.Done()
				return
			}

			usdtVol, err := strconv.ParseFloat(klines[0].QuoteAssetVolume, 64)
			if err != nil {
				fmt.Println("ParseFloat klines[0].QuoteAssetVolume error:", err, pair)
				defer wg.Done()
				return
			}

			m.Store(pair, usdtVol)
			defer wg.Done()
		}(p)
	}

	wg.Wait()

	pairUsdtMap := make(map[string]float64)

	m.Range(func(k, v interface{}) bool {
		pairUsdtMap[k.(string)] = v.(float64)
		return true
	})

	return pairUsdtMap
}

func numShortener(n float64) string {
	suffix := "K"
	divisor := 1000.0
	million := 1000000.0

	if n >= million {
		suffix = "M"
		divisor = million
	}

	return fmt.Sprintf("%.3f%s", n/divisor, suffix)
}

func isAHigherKlineOpenExists(lastIndex int, k []*binance.Kline, c float64) bool {
	for i := 0; i < lastIndex; i++ {
		currentKlineOpen, err := strconv.ParseFloat(k[i].Open, 64)

		if err != nil {
			fmt.Println("ParseFloat currentKlineOpen error:", k[i], err)
			return true
		}

		if currentKlineOpen > c {
			return true
		}
	}

	return false
}

func postSlackMessage(sc *slack.Client, channelId string, message string) {
	channelId, timestamp, err := sc.PostMessage(
		channelId,
		slack.MsgOptionText(message, false),
	)

	if err != nil {
		fmt.Println("PostMessage", channelId, timestamp, err)
	}
}

func isSurging15Min(index int, k []*binance.Kline, usdtYesterday float64) bool {
	var totalUsdtVol float64
	latestKlineClose, _ := strconv.ParseFloat(k[index].Close, 64)

	for i := index; i >= 0; i-- {
		kline := k[i]
		usdtVol, _ := strconv.ParseFloat(kline.QuoteAssetVolume, 64)
		totalUsdtVol = totalUsdtVol + usdtVol
		open, _ := strconv.ParseFloat(kline.Open, 64)
		is15thMin := i%15 == 0
		is16PercentOfUsdtVolYesterday := (totalUsdtVol/usdtYesterday >= 0.16)
		isUp7Percent := (latestKlineClose/open >= 1.007)

		if is15thMin && (is16PercentOfUsdtVolYesterday || isUp7Percent) {
			return true
		}
	}

	return false
}
