package main

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"golang.org/x/sys/windows/registry"
)

const regKeyBranch = `SOFTWARE\NateMorrison\CalcBandwidth`
const regValue1 = "bwCurrentUsed"
const regValue2 = "prevBwCurrentUsed"
const regValue3 = "daysLeftInMonth"
const regValue4 = "prevDaysLeftInMonth"
const initialWinWidth = 920
const initialWinHeight = 200

// Global pointers
var mainWin *walk.MainWindow
var resultMsgBox *walk.TextEdit
var bwTextBox *walk.LineEdit
var pushButton *walk.PushButton
var key *registry.Key
var bwCurrentUsed, prevBwCurrentUsed, daysLeftInMonth, prevDaysLeftInMonth, prevBwAtProgStart, prevDaysAtProgStart *float64

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

// This does all the calculations that are shown on the screen and returns the string to be printed
func calculateBandwidth() string {
	currentYear := time.Now().UTC().Year()
	currentMonth := time.Now().UTC().Month()
	totalDaysInMonth := calcMonthDays(currentMonth, currentYear)
	var bwLimitGBs float64
	bwLimitGBs = 1229
	// find the number of days since the first of the month (excluding today)
	hoursSinceMonthStart := time.Since(time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, time.UTC)).Hours()

	gbPerDay := bwLimitGBs / totalDaysInMonth
	gbAllowedSoFar := math.Round(bwLimitGBs / totalDaysInMonth * (hoursSinceMonthStart / 24))
	gbLeftToUse := bwLimitGBs - *bwCurrentUsed
	*daysLeftInMonth = totalDaysInMonth - (hoursSinceMonthStart / 24)
	gbPerDayLeft := gbLeftToUse / *daysLeftInMonth

	// find estimate for usage based on bw diff and time between last run
	bwDifference := *bwCurrentUsed - *prevBwAtProgStart
	timeSinceFraction := *prevDaysLeftInMonth - *daysLeftInMonth
	dailyUsageSincePrev := bwDifference / timeSinceFraction

	// find out how many hours and minutes since we last ran the program
	minsSincePrev := int(1440 * (*prevDaysAtProgStart - *daysLeftInMonth))
	hoursSincePrev := (minsSincePrev - (minsSincePrev % 60)) / 60
	minsSincePrev = minsSincePrev % 60

	// now determine if showing the estimate is worth it, it's higly inaccurate if under 144 minutes (.1 of a day)
	strDailyUsageSincePrev := ""
	if timeSinceFraction > .1 {
		strDailyUsageSincePrev = fmt.Sprintf("%.1f GB", dailyUsageSincePrev)
	} else {
		strDailyUsageSincePrev = "N/A      "
	}

	output := fmt.Sprintf("Fractional days left in month:                         %.3f         (Days this month:  %d)\r\n", *daysLeftInMonth, int(totalDaysInMonth))
	output += fmt.Sprintf("Cumulative bandwidth allowed up to today:   %.0f GB        (Used / Left:  %.0f / %d GB)\r\n", gbAllowedSoFar, *bwCurrentUsed, int(gbLeftToUse))
	output += fmt.Sprintf("Bandwidth per day remaining:                        %.3f GB  (Daily average:  %.3f GB)\r\n", gbPerDayLeft, gbPerDay)
	//output += fmt.Sprintf("Previous Bandwidth Difference and Days since:  %.0f GB        %.3f\r\n", bwDifference, timeSinceFraction)
	output += fmt.Sprintf("Daily usage estimate since last time:            %s        (%d hours %d minutes ago)\r\n", strDailyUsageSincePrev, hoursSincePrev, minsSincePrev)

	return output
}

// Calculates and outputs the necessary text to populate the main window
func setToRegAndCalc() {
	var err error
	numWithoutSpace := strings.TrimSpace(bwTextBox.Text())
	*bwCurrentUsed, err = strconv.ParseFloat(numWithoutSpace, 64)
	if err != nil {
		log.Println("Invalid characters detected, please use integers only")
	} else {
		resultMsgBox.SetText(calculateBandwidth())
	}
}

// Attempts to read last known values of program from registry (stored from last run)
func getRegKeyValues() registry.Key {
	// attempt to create key (won't delete if existing)
	k, exists, err := registry.CreateKey(registry.CURRENT_USER, regKeyBranch, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		log.Fatalf("Error creating registry key, exiting program")
	}
	if exists {
		log.Println("Registry key already existed")
	}

	return k
}

// Attempts to write a value to a registry string
func setSingleRegKeyValue(regName, regValue string) {
	err := key.SetStringValue(regName, regValue)
	if err != nil {
		log.Fatalf("Error writing registry value %s, exiting program", regName)
	}
}

// Things to perform just before exit
func closingFunctions() {
	setSingleRegKeyValue(regValue1, fmt.Sprintf("%.0f", *bwCurrentUsed))
	setSingleRegKeyValue(regValue2, fmt.Sprintf("%.0f", *prevBwAtProgStart))
	setSingleRegKeyValue(regValue3, fmt.Sprintf("%.4f", *daysLeftInMonth))
	setSingleRegKeyValue(regValue4, fmt.Sprintf("%.4f", *prevDaysAtProgStart))
}

// Get a string value from the registry
func GetRegStringValue(regStr string) string {
	value, _, err := key.GetStringValue(regStr)
	if err != nil {
		log.Fatalf("Error reading registry value %s, exiting program", regStr)
	}

	return value
}

func main() {
	key = new(registry.Key)
	*key = getRegKeyValues()

	bwCurrentUsed, prevBwCurrentUsed, daysLeftInMonth, prevDaysLeftInMonth = new(float64), new(float64), new(float64), new(float64)
	*bwCurrentUsed, _ = strconv.ParseFloat(GetRegStringValue(regValue1), 64)
	*prevBwCurrentUsed, _ = strconv.ParseFloat(GetRegStringValue(regValue2), 64)
	*daysLeftInMonth, _ = strconv.ParseFloat(GetRegStringValue(regValue3), 64)
	*prevDaysLeftInMonth, _ = strconv.ParseFloat(GetRegStringValue(regValue4), 64)

	// record previous settings now since these values might change a few times at runtime
	prevBwAtProgStart, prevDaysAtProgStart = new(float64), new(float64)
	*prevBwAtProgStart = *bwCurrentUsed     // copy value of pointer into new pointer, NOT point to the same object (since that num will change)
	*prevDaysAtProgStart = *daysLeftInMonth // copy value of pointer into new pointer, NOT point to the same object (since that num will change)

	MainWindow{
		AssignTo: &mainWin,
		Title:    "Bandwidth Calculator",
		Size:     Size{initialWinWidth, initialWinHeight},
		MinSize:  Size{500, 200},
		Layout:   VBox{},
		Children: []Widget{
			HSplitter{
				Children: []Widget{
					ScrollView{
						Layout: HBox{
							MarginsZero: true,
						},
						Children: []Widget{
							Label{
								Text: "Bandwidth Used:",
							},
							LineEdit{
								AssignTo: &bwTextBox,
								Text:     strconv.FormatFloat(*bwCurrentUsed, 'f', -1, 64),
								OnKeyPress: func(keystroke walk.Key) {
									if keystroke >= walk.Key0 && keystroke <= walk.Key9 { // if a digit key pressed
										go setToRegAndCalc()
									}
								},
							},
							PushButton{
								AssignTo: &pushButton,
								Text:     "        Press to calculate        ",
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
						MinSize:  Size{initialWinWidth / 2, initialWinHeight - 115},
						ReadOnly: true,
						Font: Font{
							Family:    "Ariel",
							PointSize: 17,
						},
						Text: calculateBandwidth(),
						OnBoundsChanged: func() {
							resultMsgBox.SetWidth(mainWin.Width() - 35)
						},
					},
				},
			},
		},
	}.Run()

	closingFunctions()
}
