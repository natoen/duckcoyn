package helpers

import "testing"

func TestCheckIfCloseIsHigher(t *testing.T) {
	result := CheckIfCloseIsHigher(2, 2.02, 0.01)

	if !result {
		t.Errorf("CheckIfCloseIsHigher was incorrect. Expected true. Received false")
	}

	result = CheckIfCloseIsHigher(2.02, 2, 0.01)

	if result {
		t.Errorf("CheckIfCloseIsHigher was incorrect. Expected false. Received true")
	}
}

func TestCheckIfUsdtVolIsHigher(t *testing.T) {
	givenUsdtVol := 20000.0

	result := CheckIfUsdtVolIsHigher(20000.0, givenUsdtVol)

	if !result {
		t.Errorf("CheckIfUsdtVolIsHigher was incorrect. Expected true. Received false")
	}

	result = CheckIfUsdtVolIsHigher(19990.0, givenUsdtVol)

	if result {
		t.Errorf("CheckIfUsdtVolIsHigher was incorrect. Expected false. Received true")
	}
}

// &{1683461760000 8.18700000 8.20600000 8.18700000 8.19500000 458.30000000 1683461819999 3755.83310000 46 302.40000000 2478.45290000}
// opentime openprice highprice lowprice closeprice volumecoin closetime volumeusdt tradenum? takerbuycoin? takerbuyusdt?
