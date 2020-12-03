package main

import (
	"encoding/json"
	"fmt"
	"locallibs/support"
	"math"
	"time"
)

type config struct {
	// var name has to be uppercase here or it won't work
	BandwidthTotal float64 `json:"bandwidthTotal"`
	BandwidthLeft  float64 `json:"bandwidthLeft"`
}

// CalcMonthDays returns the number of days in the month
func CalcMonthDays(month time.Month, year int) float64 {
	var days float64

	switch int(month) {
	case 1: // january
	case 3: // march
	case 5: // may
	case 7: // july
	case 8: // august
	case 10: // october
	case 12: // december
		days = 31
	default: // other months
		days = 30
	}

	if month == 2 { // february
		days = 28

		if year%4 == 0 {
			days = 29 // february leap year
		}
	}

	return days
}

func main() {
	support.SetupCloseHandler() // ctrl + c handler

	var config config
	stringConf := support.ReadConfigFileContents("config.json")
	err := json.Unmarshal([]byte(stringConf), &config)
	if err != nil {
		fmt.Printf("There was an error decoding the json. err = %s\n", err)
		return
	}

	currentYear := time.Now().UTC().Year()
	currentMonth := time.Now().UTC().Month()
	totalDaysInMonth := CalcMonthDays(currentMonth, currentYear)
	bwLimitGBs := config.BandwidthTotal
	bwCurrentUsed := config.BandwidthLeft
	// find the number of days since the first of the month (excluding today)
	hoursSince := time.Since(time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, time.UTC)).Hours()

	// do calcs
	gbPerDay := bwLimitGBs / totalDaysInMonth
	gbAllowedSoFar := math.Round(bwLimitGBs / totalDaysInMonth * (hoursSince / 24))
	gbLeftToUse := bwLimitGBs - bwCurrentUsed
	daysLeftInMonth := totalDaysInMonth - (hoursSince / 24)
	gbPerDayLeft := gbLeftToUse / daysLeftInMonth

	fmt.Println()
	fmt.Printf("Fractional days left in month:              %.3f   (Days this month:   %d)\n", daysLeftInMonth, int(totalDaysInMonth))
	fmt.Printf("Cumulative bandwidth allowed up to today:   %.0f GB   (Used / Left:   %.0f / %d GB)\n", gbAllowedSoFar, bwCurrentUsed, int(gbLeftToUse))
	fmt.Printf("Bandwidth per day remaining:                %.2f GB   (Daily average:   %.2f GB)\n", gbPerDayLeft, gbPerDay)
	fmt.Printf("\nPress ctrl + c to exit\n")

	// now run until user exits with ctrl + c  this allows command line to
	// remain open as long as user likes (will close after being open an hour)
	time.Sleep(time.Hour)
}
