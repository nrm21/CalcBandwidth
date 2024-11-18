package main

import (
	"errors"
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"MyLibs/myetcd"
	"MyLibs/mysupport"

	"github.com/lxn/walk"
	"golang.org/x/sys/windows/registry"
)

// Config struct
type Config struct {
	Etcd struct {
		// var name has to be uppercase here or it won't work
		Endpoints      []string `yaml:"endpoints"`
		BaseKeyToWrite string   `yaml:"baseKeyToWrite"`
		Timeout        int      `yaml:"timeout"`
		CertPath       string   `yaml:"certpath"`
	}
	dbValues                                  map[string][]byte
	bwCurrentUsed, gbPerDayLeft, bwMin, bwMax float64
}

// Returns the number of days in the month
func (mw *MainWin) calcMonthDays(month time.Month, year int) float64 {
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

// Simply returns the lower of 2 numbers
func lower(x, y float64) float64 {
	if x < y {
		return x
	} else {
		return y
	}
}

// This does all the calculations that are shown on the screen and returns the string to be printed
func (mw *MainWin) calculateBandwidth() string {
	const bwLimitGBs float64 = 1229
	var err error

	currentYear := time.Now().Year()
	currentMonth := time.Now().Month()
	if mw.bwTextBox != nil { // will be nil on initial run of func at opening of program
		mw.config.bwCurrentUsed, err = strconv.ParseFloat(strings.TrimSpace(mw.bwTextBox.Text()), 64)
	}
	if err != nil {
		log.Println("Invalid characters detected, please use integers only")
		return ""
	} else {
		// find the number of days since the first of the month (excluding today)
		hoursSinceMonthStart := time.Since(time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, time.Local)).Hours()
		totalDaysInMonth := mw.calcMonthDays(currentMonth, currentYear)
		gbPerDay := bwLimitGBs / totalDaysInMonth
		gbAllowedSoFar := math.Round(gbPerDay*(hoursSinceMonthStart/24)*100) / 100 // gets number to 2 decimals
		gbLeftToUse := bwLimitGBs - mw.config.bwCurrentUsed
		daysLeftInMonth := totalDaysInMonth - (hoursSinceMonthStart / 24)
		mw.config.gbPerDayLeft = lower(gbLeftToUse/daysLeftInMonth, gbLeftToUse)
		bwDifferential := gbAllowedSoFar - mw.config.bwCurrentUsed

		output := fmt.Sprintf("Fractional days left in month:     %.3f               (Days this month:  %d)\r\n",
			daysLeftInMonth, int(totalDaysInMonth))
		output += fmt.Sprintf("Bandwidth allowed up to today:  %.2f GB    (Difference from used / Left: %.2f / %d GB)\r\n",
			gbAllowedSoFar, bwDifferential, int(gbLeftToUse))
		output += fmt.Sprintf("Bandwidth per day remaining:    %.2f GB       (Daily average:  %.2f GB)\r\n", mw.config.gbPerDayLeft, gbPerDay)

		return output
	}
}

// Attempts to read last known values of program from registry (stored from last run)
func (mw *MainWin) getRegKeyValues() registry.Key {
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
func (mw *MainWin) setSingleRegKeyValue(regStr, regValue string) {
	err := mw.key.SetStringValue(regStr, regValue)
	if err != nil {
		walk.MsgBox(nil, "Fatal Error", "Error writing registry value: "+regStr+" exiting program", walk.MsgBoxIconError)
		log.Fatalf("Error writing registry value %s, exiting program", regStr)
	}
}

// Get a string value from the registry
func (mw *MainWin) GetRegStringValue(regStr string) string {
	value, _, err := mw.key.GetStringValue(regStr)
	if err != nil {
		walk.MsgBox(nil, "Fatal Error", "Error reading registry value: "+regStr+" exiting program", walk.MsgBoxIconError)
		log.Fatalf("Error reading registry value %s, exiting program", regStr)
	}

	return value
}

// Delete all daily data if we are in new month
func (mw *MainWin) deleteIfNewMonth() {
	dbMonth, _ := strconv.ParseInt(string(mw.config.dbValues[mw.config.Etcd.BaseKeyToWrite+"/"+regValue4]), 10, 64)
	if dbMonth != int64(time.Now().Month()) {
		// Have a msg box here notifying the user of deleting keys
		walk.MsgBox(nil, "Info", "New month, will delete all daily keys now", walk.MsgBoxIconInformation)

		// If key for the first day of month exists, we should assume all days do
		// and delete all 31 days one by one (silent error for days that don't exist)
		dayOfMonthSubkey := mw.config.Etcd.BaseKeyToWrite + "/" + regValue3
		if _, ok := mw.config.dbValues[dayOfMonthSubkey+"/01"]; ok {
			for day := 1; day <= 31; day++ {
				var strDay string
				if day < 10 {
					strDay = "0" + fmt.Sprint(day)
				} else {
					strDay = fmt.Sprint(day)
				}

				myetcd.DeleteFromEtcd(&mw.config.Etcd.CertPath, &mw.config.Etcd.Endpoints,
					dayOfMonthSubkey+"/"+strDay)
			}
		}
	}
}

// Find whichever is the higest numbered key in the range (to find the latest day of data)
func findMaxKey(data map[string][]byte) string {
	highestSeen := 0

	for k, _ := range data {
		intDay, _ := strconv.Atoi(k[len(k)-2:])

		if intDay > highestSeen {
			highestSeen = intDay
		}
	}

	// if our number is single digit add a leading zero before returning
	if highestSeen < 10 {
		return "0" + strconv.Itoa(highestSeen)
	} else {
		return strconv.Itoa(highestSeen)
	}
}

// Deletes the last day of data  we have in the graph (in case the user wants to modify or
// recalc the last few days, they can press it a few times to delete the appropriate amount)
func (mw *MainWin) deleteLastDaysData() {
	dayOfMonthSubkey := mw.config.Etcd.BaseKeyToWrite + "/" + regValue3
	data, err := myetcd.ReadFromEtcd(&mw.config.Etcd.CertPath, &mw.config.Etcd.Endpoints, dayOfMonthSubkey)
	if err != nil {
		log.Print(err.Error())
	}
	// print(data)
	myetcd.DeleteFromEtcd(&mw.config.Etcd.CertPath, &mw.config.Etcd.Endpoints, dayOfMonthSubkey+"/"+findMaxKey(data))
}

// Things to perform before showing GUI
func (mw *MainWin) getConfigAndDBValues(exePath string) {
	mw.useEtcd = false
	err := mysupport.GetConfigContentsFromYaml(exePath, &mw.config)
	if err != nil {
		log.Fatal(err)
	}

	for _, address := range mw.config.Etcd.Endpoints {
		ipAndPort := strings.Split(address, ":")
		if mw.testSockConnect(ipAndPort[0], ipAndPort[1]) { // etcd exists lets use that for settings
			mw.useEtcd = true

			// if the cert path doesnt exist
			if _, err = os.Stat(mw.config.Etcd.CertPath); errors.Is(err, os.ErrNotExist) {
				walk.MsgBox(nil, "Fatal Error", "Fatal: "+err.Error(), walk.MsgBoxIconError)
			}

			mw.config.dbValues, err = myetcd.ReadFromEtcd(&mw.config.Etcd.CertPath, &mw.config.Etcd.Endpoints, mw.config.Etcd.BaseKeyToWrite)
			if err != nil {
				walk.MsgBox(nil, "Fatal Error", "Fatal: "+err.Error()+"\nPossible authentication failure", walk.MsgBoxIconError)
				log.Fatal(err.Error())
			}
			mw.config.bwCurrentUsed, _ = strconv.ParseFloat(string(mw.config.dbValues[mw.config.Etcd.BaseKeyToWrite+"/"+regValue1]), 64)

			mw.deleteIfNewMonth()
		}
		// if we have connected sucessfully to any etcd server, we don't need to connect to
		// any others servers anymore, so just break from loop
		if mw.useEtcd {
			break
		}
	}
	if !mw.useEtcd { // etcd doesnt appear to exist lets use registry for settings
		walk.MsgBox(nil, "Info", "Unable to reach Etcd servers, using registry fallback", walk.MsgBoxIconInformation)
		mw.key = new(registry.Key)
		*mw.key = mw.getRegKeyValues()
		mw.config.bwCurrentUsed, _ = strconv.ParseFloat(mw.GetRegStringValue(regValue1), 64)
	}
}

// Checks a socket connection and returns bool of if open or not
func (mw *MainWin) testSockConnect(host string, port string) bool {
	conn, _ := net.DialTimeout("tcp", net.JoinHostPort(host, port), 500*time.Millisecond)
	if conn != nil {
		defer conn.Close()
		return true
	} else {
		return false
	}
}

// Simple function to add a leading zero to single digit calendar days
func getStrDayOfMonth(todaysDate int) string {
	if todaysDate < 10 {
		return "0" + fmt.Sprintf("%d", todaysDate)
	} else {
		return fmt.Sprintf("%d", todaysDate)
	}
}

// This func checks if days are missing between the last day of data we have
// and the current day, then adds bars for each day that is between them
func addBarsToDBIfNeeded(mw *MainWin) {
	_, bars := getBarsData(mw)
	if len(bars) > 0 {
		barsLastLabel, _ := strconv.ParseInt(bars[len(bars)-1].Label, 10, 64)
		barsLastValue := bars[len(bars)-1].Value
		daysLapse := time.Now().Day() - int(barsLastLabel)

		if daysLapse > 1 {
			differenceBetweenDays := mw.config.gbPerDayLeft - barsLastValue
			differenceBetweenDays = differenceBetweenDays / float64(daysLapse)
			for i := 1; i < daysLapse; i++ {
				// there are more than zero days missing since yesterday (or possible further
				// back) appear to not be the last bars label so we should add some bars
				barsLastLabel += 1
				barsLastValue += differenceBetweenDays
				strDayOfMonth := getStrDayOfMonth(int(barsLastLabel))

				myetcd.WriteToEtcd(&mw.config.Etcd.CertPath, &mw.config.Etcd.Endpoints,
					mw.config.Etcd.BaseKeyToWrite+"/"+regValue3+"/"+strDayOfMonth,
					fmt.Sprintf("%.3f", barsLastValue))
			}
		}
	}
}

// Writes the final values before exiting program
func (mw *MainWin) writeValuesToDB() {
	if mw.useEtcd {
		// Add leading zero to single digit days
		strDayOfMonth := getStrDayOfMonth(time.Now().Day())

		// check if there are more than zero days of data missing from chart, and if so
		// extrapolate to create the remaining bars and write them to DB
		addBarsToDBIfNeeded(mw)

		// then write to etcd
		myetcd.WriteToEtcd(&mw.config.Etcd.CertPath, &mw.config.Etcd.Endpoints,
			mw.config.Etcd.BaseKeyToWrite+"/"+regValue1, fmt.Sprintf("%.0f", mw.config.bwCurrentUsed))
		myetcd.WriteToEtcd(&mw.config.Etcd.CertPath, &mw.config.Etcd.Endpoints,
			mw.config.Etcd.BaseKeyToWrite+"/"+regValue2, fmt.Sprintf("%.3f", mw.config.gbPerDayLeft))
		myetcd.WriteToEtcd(&mw.config.Etcd.CertPath, &mw.config.Etcd.Endpoints,
			mw.config.Etcd.BaseKeyToWrite+"/"+regValue3+"/"+strDayOfMonth, fmt.Sprintf("%.3f", mw.config.gbPerDayLeft))
		myetcd.WriteToEtcd(&mw.config.Etcd.CertPath, &mw.config.Etcd.Endpoints,
			mw.config.Etcd.BaseKeyToWrite+"/"+regValue4, fmt.Sprintf("%d", int(time.Now().Month())))
		myetcd.WriteToEtcd(&mw.config.Etcd.CertPath, &mw.config.Etcd.Endpoints,
			mw.config.Etcd.BaseKeyToWrite+"/"+regValue5, fmt.Sprintf("%.3f", mw.config.bwMin))
		myetcd.WriteToEtcd(&mw.config.Etcd.CertPath, &mw.config.Etcd.Endpoints,
			mw.config.Etcd.BaseKeyToWrite+"/"+regValue6, fmt.Sprintf("%.3f", mw.config.bwMax))
	} else {
		// or write to registry if no etcd
		mw.setSingleRegKeyValue(regValue1, fmt.Sprintf("%.0f", mw.config.bwCurrentUsed))
		mw.setSingleRegKeyValue(regValue2, fmt.Sprintf("%.3f", mw.config.gbPerDayLeft))
	}
}
