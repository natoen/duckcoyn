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

type intervalVolume struct {
	Vol         float64
	Interval    int
	Change      float64
	IntervalStr string
}

var intervalStr1m = "1m"
var intervalStr3m = "3m"
var intervalStr5m = "5m"
var intervalStr15m = "15m"
var intervalStr30m = "30m"
var intervalStr1H = "1H"
var intervalStr2H = "2H"

var intervalVolumes = []intervalVolume{{
	Change:      1.0295,
	Interval:    1,
	Vol:         0.0495,
	IntervalStr: intervalStr1m,
}, {
	Change:      1.0295,
	Interval:    3,
	Vol:         0.0495,
	IntervalStr: intervalStr3m,
}, {
	Change:      1.0295,
	Interval:    5,
	Vol:         0.0495,
	IntervalStr: intervalStr5m,
}, {
	Change:      1.0295,
	Interval:    15,
	Vol:         0.0995,
	IntervalStr: intervalStr15m,
}, {
	Change:      1.0295,
	Interval:    30,
	Vol:         0.1195,
	IntervalStr: intervalStr30m,
}, {
	Change:      1.0295,
	Interval:    60,
	Vol:         0.1495,
	IntervalStr: intervalStr1H,
}, {
	Change:      1.0295,
	Interval:    120,
	Vol:         0.1995,
	IntervalStr: intervalStr2H,
}}

func CheckForSpikingCoins(yesterdayUsdtPairs map[string]float64, bc *binance.Client, sc *slack.Client, t time.Time, lastVolRateMap *sync.Map, skipPair1mMap *sync.Map) {
	surgingMsg := ""

	for pair := range yesterdayUsdtPairs {
		_, isSkipPair1m := skipPair1mMap.Load(pair)

		wg.Add(1)

		go func(pair string) {
			minuteKlines := GetKlines(bc, pair, "1m", 1000, t.UnixMilli())
			indexOfLastMinuteKline := len(minuteKlines) - 1
			minuteKline := minuteKlines[indexOfLastMinuteKline]
			minuteKlineClose, _ := strconv.ParseFloat(minuteKline.Close, 64)
			minuteKlineOpen, _ := strconv.ParseFloat(minuteKline.Open, 64)
			minuteKlineUsdtVol, _ := strconv.ParseFloat(minuteKline.QuoteAssetVolume, 64)

			todayKline := GetKlines(bc, pair, "1d", 1, t.UnixMilli())[0]
			todayKlineClose, _ := strconv.ParseFloat(todayKline.Close, 64)
			todayKlineOpen, _ := strconv.ParseFloat(todayKline.Open, 64)
			todayKlineUsdtVol, _ := strconv.ParseFloat(todayKline.QuoteAssetVolume, 64)
			todayPriceRatio := todayKlineClose / todayKlineOpen
			yesterdayUsdtVol := yesterdayUsdtPairs[pair]
			todayVolRatio := todayKlineUsdtVol / yesterdayUsdtVol
			coinName := pair[0 : len(pair)-4]
			isGreen := minuteKlineOpen <= minuteKlineClose
			isTodayVolMorethan100k := todayKlineUsdtVol >= 100000.0
			isMinuteVol2p5PercentOfYesterdayVol := (minuteKlineUsdtVol/yesterdayUsdtVol >= 0.025) && (todayKlineUsdtVol >= 40000.0)
			isMinuteChangeUpBy4Percent := minuteKlineClose/minuteKlineOpen >= 1.04

			// dayMinutesRatio := float64(hour*60+t.Minute()+1) / 1440.0
			// volRate := todayKlineUsdtVol / (yesterdayUsdtVol * dayMinutesRatio)
			// isNxVolRate := 1.49 <= volRate

			isSurgingMinutes, isSurgingMinutesStr := SurgingMinutes(indexOfLastMinuteKline, minuteKlines, yesterdayUsdtVol, intervalVolumes, t)

			if !isSkipPair1m && isTodayVolMorethan100k {
				message := fmt.Sprintf("<https://www.binance.com/en/trade/%s_USDT?type=spot|%s> %s %.2f%% %.2f%% %s", coinName, coinName, numShortener(yesterdayUsdtVol), todayVolRatio*100, (todayPriceRatio-1)*100, t.String()[11:16])

				if isGreen && (isMinuteVol2p5PercentOfYesterdayVol || isMinuteChangeUpBy4Percent) /* || isSurgingMinutes */ {
					skipPair1mMap.Store(pair, t)

					if isMinuteVol2p5PercentOfYesterdayVol {
						message = message + " 2.5%YV"
					}

					if isMinuteChangeUpBy4Percent {
						message = message + " 4%"
					}

					if isSurgingMinutes {
						message = message + isSurgingMinutesStr
					}

					surgingMsg = surgingMsg + message + "\n"
				}
			}

			defer wg.Done()
		}(pair)
	}

	wg.Wait()

	if surgingMsg != "" {
		postSlackMessage(sc, "C01UHA03VEY", surgingMsg)
	}
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
	var prices []*binance.SymbolPrice
	var err error
	isGetPairsNotSuccess := true

	for isGetPairsNotSuccess {
		prices, err = bc.NewListPricesService().Do(context.Background())

		if err != nil {
			fmt.Println("NewListPricesService error:", err)
			time.Sleep(5 * time.Second)
		} else {
			isGetPairsNotSuccess = false
		}
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
		"PAXUSDT":   true,
		"USDSBUSDT": true,
		"MFTUSDT":   true,
		"BTSUSDT":   true,
		"XZCUSDT":   true,
		"PLAUSDT":   true,
		"TOMOUSDT":  true,
		"XMRUSDT":   true,
		"AEURUSDT":  true,
		"FDUSD":     true,
		"XEMUSDT":   true,
		"COCOSUSDT": true,
		"BTTUSDT":   true,
		"STORMUSDT": true,
		"MITHUSDT":  true,
		"BZRXUSDT":  true,
		"NANOUSDT":  true,
		"PNTUSDT":   true,
		"DNTUSDT":   true,
		"WAVESUSDT": true,
		"AGIXUSDT":  true,
		"OCEANUSDT": true,
		"FETUSDT":   true,
		"MULTIUSDT": true,
		"FRONTUSDT": true,
		"VGXUSDT":   true,
		"EPXUSDT":   true,
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

func IsAHigher1mKlineOpenExistsBefore2Hours(lastIndex int, k []*binance.Kline, c float64) bool {
	minsIn2Hours := 120

	for i := 0; i < lastIndex-minsIn2Hours; i++ {
		kline := k[i]
		currentKlineOpen, _ := strconv.ParseFloat(kline.Open, 64)

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

func IsAHigher15mKlineOpenExists(lastIndex int, k []*binance.Kline, c float64) bool {
	for i := lastIndex; i >= 0; i-- {
		is15thMin := i%15 == 0

		if is15thMin {
			currentKlineOpen, _ := strconv.ParseFloat(k[i].Open, 64)

			if currentKlineOpen > c {
				return true
			}
		}
	}

	return false
}

func SurgingMinutes(lastIndex int, k []*binance.Kline, yesterdayUsdtVol float64, intervalVolumes []intervalVolume, t time.Time) (bool, string) {
	latestKlineClose, _ := strconv.ParseFloat(k[lastIndex].Close, 64)

	for _, v := range intervalVolumes {

		if (v.IntervalStr == intervalStr3m && (t.Minute()+1)%3 != 0) || (v.IntervalStr == intervalStr5m && (t.Minute()+1)%5 != 0) || (v.IntervalStr == intervalStr15m && (t.Minute()+1)%15 != 0) || (v.IntervalStr == intervalStr30m && (t.Minute()+1)%30 != 0) || (v.IntervalStr == intervalStr1H && (t.Minute()+1)%60 != 0) || (v.IntervalStr == intervalStr2H && (t.Hour()%2 == 0 || t.Minute()+1 != 0)) {
			continue
		}

		accumUsdtVol := 0.0
		inc := 0
		count := 0

		for j := lastIndex; j >= 0; j-- {
			kline := k[j]
			usdtVol, _ := strconv.ParseFloat(kline.QuoteAssetVolume, 64)
			accumUsdtVol = accumUsdtVol + usdtVol
			inc = inc + 1 // add 1 right away before checking if it is an interval
			isInterval := inc%v.Interval == 0

			if isInterval {
				open, _ := strconv.ParseFloat(kline.Open, 64)
				close, _ := strconv.ParseFloat(k[j+v.Interval-1].Close, 64)
				isGreen := close >= open

				if !isGreen {
					break
				} else {
					count = count + 1
				}

				isChangeUp := latestKlineClose/open >= v.Change
				isAccumUsdtVol40k := accumUsdtVol >= 40000.0
				isPercentOfYesterdayUsdtVol := (accumUsdtVol / yesterdayUsdtVol) >= v.Vol

				if isChangeUp && isAccumUsdtVol40k && isPercentOfYesterdayUsdtVol {
					return true, " " + v.IntervalStr
				}
			}
		}
	}

	return false, ""
}
