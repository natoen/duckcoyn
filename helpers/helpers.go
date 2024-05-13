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

func CheckForSpikingCoins(yesterdayUsdtPairs map[string]float64, bc *binance.Client, sc *slack.Client, t time.Time, spd *sync.Map) {
	numOfMinuteKlines := 1000
	indexOfLastMinuteKline := numOfMinuteKlines - 1

	for pair := range yesterdayUsdtPairs {
		_, isSkipPairDay := spd.Load(pair)

		wg.Add(1)

		go func(pair string) {
			minuteKlines := GetKlines(bc, pair, "1m", numOfMinuteKlines, t.UnixMilli())
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
			isAHigher1mKlineOpenExists := IsAHigher1mKlineOpenExists(indexOfLastMinuteKline, minuteKlines, minuteKlineClose)
			hour := t.Hour()
			isTodayVolMorethan100k := todayKlineUsdtVol >= 100000.0
			isMinuteVolMorethan40k := minuteKlineUsdtVol >= 40000.0
			isMinuteVol2p5PercentOfYesterdayVol := minuteKlineUsdtVol/yesterdayUsdtVol >= 0.025
			isMinuteChangeUpByPoint9Percent := minuteKlineClose/minuteKlineOpen >= 1.009
			isMinuteChangeUpBy4Percent := minuteKlineClose/minuteKlineOpen >= 1.04
			isMinuteSpike := (isMinuteVolMorethan40k && isMinuteVol2p5PercentOfYesterdayVol && isMinuteChangeUpByPoint9Percent)
			isSurgingMinutes := SurgingMinutes(indexOfLastMinuteKline, minuteKlines, yesterdayUsdtVol)

			if hour < 9 {
				hour = hour + 15
			} else {
				hour = hour - 9
			}

			dayMinutesRatio := float64(hour*60+t.Minute()+1) / 1440.0
			isTodayVolRate2x := (yesterdayUsdtVol * dayMinutesRatio * 2) <= todayKlineUsdtVol

			message := fmt.Sprintf("<https://www.binance.com/en/trade/%s_USDT?type=spot|%s> %s %s %.2f%% %.2f%% %s", coinName, coinName, numShortener(todayKlineUsdtVol), numShortener(yesterdayUsdtVol), todayVolRatio*100, (todayPriceRatio-1)*100, t.String()[11:16])

			if !isAHigher1mKlineOpenExists && isGreen && isTodayVolMorethan100k {
				if !isSkipPairDay && (isTodayVolRate2x || isMinuteSpike || isMinuteChangeUpBy4Percent || isSurgingMinutes) {
					spd.Store(pair, minuteKlineUsdtVol)

					if isTodayVolRate2x {
						message = message + " 2X"
					} else if isMinuteChangeUpBy4Percent {
						message = message + " 4%"
					} else if isMinuteSpike {
						message = message + " 1M"
					} else if isSurgingMinutes {
						message = message + " 1MS"
					}

					postSlackMessage(sc, "C01UHA03VEY", message)

				}

			}

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

func IsAHigher1mKlineOpenExists(lastIndex int, k []*binance.Kline, c float64) bool {
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

// checks the 15th minute bars to be increasing in price moving forward while
// having an accumulative volume of 16% of yesterday
// e.g. 00:00 is 150, 00:15 is 160, etc
func Surging15Min(lastIndex int, k []*binance.Kline, usdtYesterday float64) bool {
	accumUsdtVol := 0.0
	counter15thMin := 0.0
	redCounter := 0
	latestKlineClose, _ := strconv.ParseFloat(k[lastIndex].Close, 64)

	for i := lastIndex; i >= 0; i-- {
		kline := k[i]
		lastIndexOpen, _ := strconv.ParseFloat(kline.Open, 64)
		usdtVol, _ := strconv.ParseFloat(kline.QuoteAssetVolume, 64)
		accumUsdtVol = accumUsdtVol + usdtVol
		is15thMin := i%15 == 0

		if is15thMin {
			counter15thMin = counter15thMin + 1
			every15MinClose, _ := strconv.ParseFloat(k[i+14].Close, 64)
			isGreen := lastIndexOpen <= every15MinClose // same price is ok

			if !isGreen {
				redCounter = redCounter + 1
				redKlineIsMoreThan1 := redCounter > 1
				klineIs5PercentDown := 0.995 > every15MinClose/lastIndexOpen

				if redKlineIsMoreThan1 || klineIs5PercentDown {
					return false
				}
			}

			vol15MinX2 := ((counter15thMin * 0.25 * 2) / 24)
			usdtVolYesterdayRatio := accumUsdtVol / usdtYesterday
			isAccum2xRateOfYesterday := usdtVolYesterdayRatio >= vol15MinX2
			is16PercentOfUsdtVolYesterday := (usdtVolYesterdayRatio >= 0.16) && (isAccum2xRateOfYesterday)
			isUp7Percent := (i <= 90) && (latestKlineClose/lastIndexOpen >= 1.07)

			if is16PercentOfUsdtVolYesterday || isUp7Percent {
				return true
			}
		}
	}

	return false
}

// 2% up, 3% of yesterday's volume, 70kUSDT
func SurgingMinutes(lastIndex int, k []*binance.Kline, yesterdayUsdtVol float64) bool {
	accumUsdtVol := 0.0
	latestKlineClose, _ := strconv.ParseFloat(k[lastIndex].Close, 64)

	for i := lastIndex; i >= 0; i-- {
		kline := k[i]
		open, _ := strconv.ParseFloat(kline.Open, 64)
		close, _ := strconv.ParseFloat(kline.Close, 64)
		usdtVol, _ := strconv.ParseFloat(kline.QuoteAssetVolume, 64)
		accumUsdtVol = accumUsdtVol + usdtVol
		isGreen := close >= open
		is2PercentUp := latestKlineClose/open >= 1.02
		isAccumUsdtVol70k := accumUsdtVol >= 70000.0
		is3PercentOfYesterdayUsdtVol := accumUsdtVol/yesterdayUsdtVol >= 0.03

		if !isGreen {
			return false
		}

		if is2PercentUp && isAccumUsdtVol70k && is3PercentOfYesterdayUsdtVol {
			return true
		}
	}

	return false
}
