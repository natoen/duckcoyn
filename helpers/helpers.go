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

func CheckForSpikingCoins(yesterdayUsdtPairs map[string]float64, bc *binance.Client, sc *slack.Client, t time.Time, sp *sync.Map) {
	numOfKlines := 180
	indexOfLastKline := numOfKlines - 1

	for pair := range yesterdayUsdtPairs {
		_, isSkipPair := sp.Load(pair)

		if isSkipPair {
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

			for i := 0; i < indexOfLastKline; i++ {
				currentKlineOpen, err := strconv.ParseFloat(klines[i].Open, 64)
				if err != nil {
					fmt.Println("ParseFloat currentKlineOpen error:", klines[i], err)
					return
				}

				if currentKlineOpen > latestKlineClose {
					defer wg.Done()
					return
				}
			}

			latestKlineUsdtVol, err := strconv.ParseFloat(latestKline.QuoteAssetVolume, 64)
			if err != nil {
				fmt.Println("ParseFloat latestKlineUsdtVol error:", latestKline, err)
				return
			}

			latestKlineOpen, err := strconv.ParseFloat(latestKline.Open, 64)
			if err != nil {
				fmt.Println("ParseFloat latestKlineOpen error:", latestKline, err)
				return
			}

			isGreen := latestKlineOpen <= latestKlineClose
			is1PercentUp := (latestKlineClose / latestKlineOpen) >= 1.007 // not really 1% but 0.7%
			yesterdayUsdtVol := latestKlineUsdtVol / yesterdayUsdtPairs[pair]
			isUsdtVol4PercentOfYesterday := (yesterdayUsdtVol) > 0.04
			isMoreThan20kUsdt := latestKlineUsdtVol >= 20000.0

			if isGreen && isMoreThan20kUsdt && (isUsdtVol4PercentOfYesterday || is1PercentUp) {
				text := fmt.Sprintf("<https://www.binance.com/en/trade/%s_USDT?type=spot|%s> %.0f%% %.0f %s", pair[0:len(pair)-4], pair, yesterdayUsdtVol*100, latestKlineUsdtVol, t.String()[11:16])

				channelID, timestamp, err := sc.PostMessage(
					"C01V0V91NTS",
					slack.MsgOptionText(text, false),
				)

				if err != nil {
					fmt.Println("PostMessage", channelID, timestamp, err)
				}

				sp.Store(pair, latestKlineUsdtVol)
			}

			defer wg.Done()
		}(pair)

		wg.Wait()
	}
}

func GetKlines(bc *binance.Client, s string, i string, l int, et int64) []*binance.Kline {
	klines, err := bc.NewKlinesService().
		Symbol(s).
		Interval(i).
		Limit(l).
		EndTime(et).
		Do(context.Background())

	if err != nil {
		fmt.Println("GetKlines error:", err)
		panic(err)
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
		isUsdtPair := strings.Contains(p.Symbol, "USDT")

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
