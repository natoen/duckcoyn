package main

func main() {
	var (
		// binanceApiKey    = "SjtKWLrEyswIwTvbGj4bpUAYLP4LjdZb02aMBcI0xOzMzbOsN17SVUbYH0b9rhMA"
		// binanceSecretKey = "13JtnIW1pYLlRm3fWAVY3p6CzCQiwVTgEPZpccQwokClvEVd9VlIbEaiclLTm5H9"
		slackToken = "xoxb-1953607810134-2082368693729-5ORkYiqyztdZsQAvijlMquRE"
	)

	// bc := binance.NewClient(binanceApiKey, binanceSecretKey)

	// ti := time.Unix(0, 1718852340000*int64(time.Millisecond))
	// hour := ti.Hour()
	// fmt.Println(ti.Hour(), ti.Minute())

	// if hour < 9 {
	// 	hour = hour + 15
	// } else {
	// 	hour = hour - 9
	// }

	// dayMinutesRatio := float64(hour*60+ti.Minute()+1) / 1440.0
	// volRate := (78589 + 115784 + 52057) / (881834 * dayMinutesRatio)
	// isNxVolRate := 1.49 <= volRate
	// fmt.Println(isNxVolRate, volRate, dayMinutesRatio)

	// HOW TO CHECK FOR SURGING MINUTES
	// minuteKlines := helpers.GetKlines(bc, "ONGUSDT", "1m", 1000, 1718852340000)
	// isSurgingMinutes, isSurgingMinutesStr := helpers.SurgingMinutes(999, minuteKlines, 881834)
	// fmt.Print(isSurgingMinutes, isSurgingMinutesStr)

	// 1710668667000 18:44
	// 1710671367000 19:29

	// klines := helpers.GetKlines(bc, "NULSUSDT", "1m", 240, 1710780240000)

	// latestKline := klines[239]

	// latestKlineClose, err := strconv.ParseFloat(latestKline.Close, 64)
	// if err != nil {
	// 	fmt.Println("ParseFloat latestKlineClose error:", latestKline, err)
	// 	return
	// }

	// fmt.Println(helpers.IsAHigherKlineOpenExists(239, klines, latestKlineClose))

	// fmt.Println(helpers.Surging15Min(239, klines, 1184000))
	// t := time.Now()
	// hour := t.Hour()

	// if hour < 9 {
	// 	hour = hour + 15
	// } else {
	// 	hour = hour - 9
	// 		}

	// dayMinutesRatio := float64(hour*60+t.Minute()+1) / 1440.0

	// t := time.Now().Add(-1 * time.Minute)
	// t1 := time.Now()

	// fmt.Println(t1.After(t))

}
