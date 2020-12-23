package main

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"golang.org/x/sys/windows/registry"
)

// Returns the number of days in the month
func calcMonthDays(month time.Month, year int) float64 {
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

func calculateBandwidth(bwCurrentUsed float64) string {
	currentYear := time.Now().UTC().Year()
	currentMonth := time.Now().UTC().Month()
	totalDaysInMonth := calcMonthDays(currentMonth, currentYear)
	var bwLimitGBs float64
	bwLimitGBs = 1229
	// find the number of days since the first of the month (excluding today)
	hoursSince := time.Since(time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, time.UTC)).Hours()

	// do calcs
	gbPerDay := bwLimitGBs / totalDaysInMonth
	gbAllowedSoFar := math.Round(bwLimitGBs / totalDaysInMonth * (hoursSince / 24))
	gbLeftToUse := bwLimitGBs - bwCurrentUsed
	daysLeftInMonth := totalDaysInMonth - (hoursSince / 24)
	gbPerDayLeft := gbLeftToUse / daysLeftInMonth

	output := fmt.Sprintf("Fractional days left in month:                      %.3f       (Days this month:  %d)\r\n", daysLeftInMonth, int(totalDaysInMonth))
	output += fmt.Sprintf("Cumulative bandwidth allowed up to today:  %.0f GB     (Used / Left:  %.0f / %d GB)\r\n", gbAllowedSoFar, bwCurrentUsed, int(gbLeftToUse))
	output += fmt.Sprintf("Bandwidth per day remaining:                     %.2f GB  (Daily average:  %.2f GB)\r\n", gbPerDayLeft, gbPerDay)

	return output
}

func main() {
	var resultMsgBox *walk.TextEdit
	var bwTextBox *walk.LineEdit
	var bwCurrentUsed float64
	bwCurrentUsed = 0
	regKey := `SOFTWARE\NateMorrison\CalcBandwidth`
	regValue := "bwCurrentUsed"

	// now attempt to read last known value of bwCurrentUsed from registry
	k, err := registry.OpenKey(registry.CURRENT_USER, regKey, registry.QUERY_VALUE)
	if err != nil {
		// key dones't exist lets create it
		k, _, err = registry.CreateKey(registry.CURRENT_USER, regKey, registry.QUERY_VALUE|registry.SET_VALUE)
		if err != nil {
			log.Fatalf("Error creating registry key, exiting program")
		}
		// then write current value to the key
		k, err := registry.OpenKey(registry.CURRENT_USER, regKey, registry.QUERY_VALUE|registry.SET_VALUE)
		if err != nil {
			log.Fatalf("Unable to open key to write too")
		}
		k.SetStringValue(regValue, strconv.FormatFloat(bwCurrentUsed, 'f', -1, 64))
	}
	s, _, err := k.GetStringValue(regValue)
	bwCurrentUsed, _ = strconv.ParseFloat(s, 64)

	output := calculateBandwidth(bwCurrentUsed)

	MainWindow{
		Title:  "Bandwidth Calculator",
		Size:   Size{800, 200},
		Layout: VBox{},
		Children: []Widget{
			HSplitter{
				Children: []Widget{
					Label{Text: "Bandwidth Used:"},
					LineEdit{AssignTo: &bwTextBox, Text: strconv.FormatFloat(bwCurrentUsed, 'f', -1, 64)},
					PushButton{
						MaxSize: Size{30, 20},
						Text:    "Press to calculate",
						OnClicked: func() {
							go func() {
								// get new bandwidth value entered by user and write to registry before calculating
								bwCurrentUsed, _ = strconv.ParseFloat(bwTextBox.Text(), 64)

								k, err := registry.OpenKey(registry.CURRENT_USER, regKey, registry.QUERY_VALUE|registry.SET_VALUE)
								if err != nil {
									log.Fatalf("Unable to open key to write too")
								}
								k.SetStringValue(regValue, strconv.FormatFloat(bwCurrentUsed, 'f', -1, 64))

								resultMsgBox.SetText(calculateBandwidth(bwCurrentUsed))
							}()
						},
					},
				},
			},
			HSplitter{
				Children: []Widget{
					TextEdit{
						AssignTo: &resultMsgBox,
						ReadOnly: true,
						Font: Font{
							Family:    "Ariel",
							PointSize: 15,
							//Bold:      true,
						},
						Text: output,
					},
				},
			},
		},
	}.Run()
}
