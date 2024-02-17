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

func CheckForSpikingCoins(pairs []string, yesterdayUsdtPairs map[string]float64, bc *binance.Client, sc *slack.Client, t time.Time, sp *sync.Map) {
	for _, pair := range pairs {
		_, isSkipPair := sp.Load(pair)

		if isSkipPair {
			continue
		}

		wg.Add(1)

		go func(pair string) {
			kline := GetKlines(bc, pair, "1m", 1, t.UnixMilli())[0]

			klineUsdtVol, err := strconv.ParseFloat(kline.QuoteAssetVolume, 64)
			if err != nil {
				fmt.Println("ParseFloat klineUsdtVol error:", err)
				return
			}

			klineOpen, err := strconv.ParseFloat(kline.Open, 64)
			if err != nil {
				fmt.Println("ParseFloat klineOpen error:", err)
				return
			}

			klineClose, err := strconv.ParseFloat(kline.Close, 64)
			if err != nil {
				fmt.Println("ParseFloat klineClose error:", err)
				return
			}

			isGreen := klineOpen <= klineClose
			is1PercentUp := (klineClose / klineOpen) >= 1.007 // not really 1% but 0.7%
			isUsdtVol4PercentOfYesterday := (klineUsdtVol / yesterdayUsdtPairs[pair]) > 0.04
			isMoreThan20kUsdt := klineUsdtVol >= 20000.0

			if (isUsdtVol4PercentOfYesterday && isGreen && isMoreThan20kUsdt) || is1PercentUp {
				text := fmt.Sprintf("%s %.2f %s", pair, klineUsdtVol, t.String()[11:16])

				channelID, timestamp, err := sc.PostMessage(
					"C01V0V91NTS",
					slack.MsgOptionText(text, false),
				)

				if err != nil {
					fmt.Println("PostMessage", channelID, timestamp, err)
				}

				sp.Store(pair, klineUsdtVol)
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

func GetYesterdayUsdtPairs(bc *binance.Client, pairSymbols []string) map[string]float64 {
	yesterday := time.Now().Add(-24 * time.Hour).UnixMilli()
	m := &sync.Map{}

	for _, symbol := range pairSymbols {
		s := fmt.Sprintf("%v", symbol)
		wg.Add(1)

		go func() {
			kline := GetKlines(bc, s, "1d", 1, yesterday)[0]
			usdtVol, err := strconv.ParseFloat(kline.QuoteAssetVolume, 64)

			if err != nil {
				fmt.Println("ParseFloat klineUsdtVol error:", err)
				return
			}

			m.Store(s, usdtVol)
			defer wg.Done()
		}()
	}

	wg.Wait()

	pairUsdtMap := make(map[string]float64)

	m.Range(func(k, v interface{}) bool {
		pairUsdtMap[k.(string)] = v.(float64)
		return true
	})

	return pairUsdtMap
}
