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

// Global vars
var resultMsgBox *walk.TextEdit
var bwTextBox *walk.LineEdit
var bwCurrentUsed, prevBwUsed float64
var daysLeftInMonth *float64
var key registry.Key
var regKey, regValue1, regValue2, regValue3, strBwCurrentUsed string

// Returns the number of days in the month
func calcMonthDays(month time.Month, year int) float64 {
	var days float64

	switch int(month) {
	case 1, 3, 5, 7, 8, 10, 12: // january march may july august october december
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

func calculateBandwidth() string {
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
	prevDaysLeftInMonth := *daysLeftInMonth
	*daysLeftInMonth = totalDaysInMonth - (hoursSince / 24)
	daysSince := *daysLeftInMonth - prevDaysLeftInMonth
	gbPerDayLeft := gbLeftToUse / *daysLeftInMonth
	bwDifference := bwCurrentUsed - prevBwUsed

	output := fmt.Sprintf("Fractional days left in month:                       %.3f         (Days this month:  %d)\r\n", *daysLeftInMonth, int(totalDaysInMonth))
	output += fmt.Sprintf("Cumulative bandwidth allowed up to today:   %.0f GB        (Used / Left:  %.0f / %d GB)\r\n", gbAllowedSoFar, bwCurrentUsed, int(gbLeftToUse))
	output += fmt.Sprintf("Bandwidth per day remaining:                      %.3f GB   (Daily average:  %.3f GB)\r\n", gbPerDayLeft, gbPerDay)
	output += fmt.Sprintf("Previous Bandwidth Difference and Days since:  %.0f GB        %.3f\r\n", bwDifference, daysSince)

	return output
}

func setToRegAndCalc() {
	bwCurrentUsed, _ = strconv.ParseFloat(bwTextBox.Text(), 64)
	strBwCurrentUsed = bwTextBox.Text()

	resultMsgBox.SetText(calculateBandwidth())
}

// Attempts to read last known values of program from registry (stored from last run)
func getRegKeyValues() registry.Key {
	// key doesn't exist lets create it
	k, exists, err := registry.CreateKey(registry.CURRENT_USER, regKey, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		log.Fatalf("Error creating registry key, exiting program")
	}
	if exists {
		log.Println("Registry key already existed")
	}

	return k
}

// Thigs to do just before exit
func closingFunctions(prevBw string) {
	// now write the current bwCurrentUsed to previous
	err := key.SetStringValue(regValue2, prevBw)
	if err != nil {
		log.Fatalf("Error writing registry value %s, exiting program", regValue2)
	}

	err = key.SetStringValue(regValue3, fmt.Sprintf("%.3f", *daysLeftInMonth))
	if err != nil {
		log.Fatalf("Error writing registry value %s, exiting program", regValue3)
	}

	// write the current bandwidth entered by user to bwCurrentUsed
	key.SetStringValue(regValue1, strBwCurrentUsed)
}

func main() {
	// init some global vars
	bwCurrentUsed = 0
	regKey = `SOFTWARE\NateMorrison\CalcBandwidth`
	regValue1 = "bwCurrentUsed"
	regValue2 = "prevBwCurrentUsed"
	regValue3 = "daysLeftInMonth"
	daysLeftInMonth = new(float64)
	var err error

	key = getRegKeyValues()

	strBwCurrentUsed, _, err = key.GetStringValue(regValue1)
	if err != nil {
		log.Fatalf("Error reading registry value %s, exiting program", regValue1)
	}
	prevBw := strBwCurrentUsed // record previous bandwidth now since this value might change a few times at runtime

	val, _, err := key.GetStringValue(regValue2)
	if err != nil {
		log.Fatalf("Error reading registry value %s, exiting program", regValue2)
	}
	prevBwUsed, _ = strconv.ParseFloat(val, 64)

	val, _, err = key.GetStringValue(regValue3)
	if err != nil {
		log.Fatalf("Error reading registry value %s, exiting program", regValue3)
	}
	*daysLeftInMonth, _ = strconv.ParseFloat(val, 64)

	bwCurrentUsed, _ = strconv.ParseFloat(strBwCurrentUsed, 64)
	output := calculateBandwidth()

	MainWindow{
		Title:  "Bandwidth Calculator",
		Size:   Size{820, 220},
		Layout: VBox{},
		Children: []Widget{
			HSplitter{
				Children: []Widget{
					ScrollView{
						Layout: HBox{MarginsZero: true},
						Children: []Widget{
							Label{Text: "Bandwidth Used:"},
							LineEdit{
								AssignTo: &bwTextBox,
								Text:     strconv.FormatFloat(bwCurrentUsed, 'f', -1, 64),
								OnKeyPress: func(key walk.Key) {
									if key >= walk.Key0 && key <= walk.Key9 { // if a digit key pressed
										go setToRegAndCalc()
									}
								},
							},
							PushButton{
								MinSize: Size{150, 20},
								MaxSize: Size{150, 20},
								Text:    "Press to calculate",
								OnClicked: func() {
									go setToRegAndCalc()
								},
							},
						},
					},
				},
			},
			HSplitter{
				Children: []Widget{
					TextEdit{
						AssignTo: &resultMsgBox,
						MinSize:  Size{800, 100},
						ReadOnly: true,
						Font: Font{
							Family:    "Ariel",
							PointSize: 15,
						},
						Text: output,
					},
				},
			},
		},
	}.Run()

	closingFunctions(prevBw)
}
