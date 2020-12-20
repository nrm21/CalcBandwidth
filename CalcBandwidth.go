package main

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

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

func calculate(bwCurrentUsed float64) string {
	currentYear := time.Now().UTC().Year()
	currentMonth := time.Now().UTC().Month()
	totalDaysInMonth := CalcMonthDays(currentMonth, currentYear)
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
	var bwCurrentUsed float64
	bwCurrentUsed = 1
	output := calculate(bwCurrentUsed)

	var resultMsgBox *walk.TextEdit
	var bwTextBox *walk.LineEdit

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
								bwCurrentUsed, _ = strconv.ParseFloat(bwTextBox.Text(), 64)
								resultMsgBox.SetText(calculate(bwCurrentUsed))
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
