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

	"github.com/lxn/walk"
	"github.com/nrm21/support"
	"github.com/wcharczuk/go-chart"
	"golang.org/x/sys/windows/registry"
	"gopkg.in/yaml.v2"
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
}

// Unmarshals the config contents from file into memory
func (mw *MainWin) getConfigContentsFromYaml(filename string) error {
	file, err := support.ReadConfigFileContents(filename)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(file, &mw.config)
	if err != nil {
		return err
	}

	return nil
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

// This does all the calculations that are shown on the screen and returns the string to be printed
func (mw *MainWin) calculateBandwidth() string {
	const bwLimitGBs float64 = 1229
	currentYear := time.Now().Year()
	currentMonth := time.Now().Month()

	// find the number of days since the first of the month (excluding today)
	hoursSinceMonthStart := time.Since(time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, time.Local)).Hours()
	totalDaysInMonth := mw.calcMonthDays(currentMonth, currentYear)
	gbPerDay := bwLimitGBs / totalDaysInMonth
	gbAllowedSoFar := math.Round(bwLimitGBs / totalDaysInMonth * (hoursSinceMonthStart / 24))
	gbLeftToUse := bwLimitGBs - mw.bwCurrentUsed
	daysLeftInMonth := totalDaysInMonth - (hoursSinceMonthStart / 24)
	mw.gbPerDayLeft = gbLeftToUse / daysLeftInMonth
	bwDifferential := gbAllowedSoFar - mw.bwCurrentUsed

	output := fmt.Sprintf("Fractional days left in month:      %.3f         (Days this month:  %d)\r\n",
		daysLeftInMonth, int(totalDaysInMonth))
	output += fmt.Sprintf("Bandwidth allowed up to today:   %.0f GB    (Used / Differential / Left:  %.0f / %.0f / %d GB)\r\n",
		gbAllowedSoFar, mw.bwCurrentUsed, bwDifferential, int(gbLeftToUse))
	output += fmt.Sprintf("Bandwidth per day remaining:     %.3f GB  (Daily average:  %.3f GB)\r\n", mw.gbPerDayLeft, gbPerDay)

	return output
}

// Calculates and outputs the necessary text to populate the main window
func (mw *MainWin) setToRegAndCalc() {
	var err error
	numWithoutSpace := strings.TrimSpace(mw.bwTextBox.Text())
	mw.bwCurrentUsed, err = strconv.ParseFloat(numWithoutSpace, 64)
	if err != nil {
		log.Println("Invalid characters detected, please use integers only")
	} else {
		mw.resultMsgBox.SetText(mw.calculateBandwidth())
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
func (mw *MainWin) deleteIfNewMonth(etcdValues *(map[string][]byte)) {
	dbMonth, _ := strconv.ParseInt(string((*etcdValues)[mw.config.Etcd.BaseKeyToWrite+"/"+regValue4]), 10, 64)
	if dbMonth != int64(time.Now().Month()) {
		// Have a msg box here notifying the user of deleting keys
		walk.MsgBox(nil, "Info", "New month, will delete all daily keys now", walk.MsgBoxIconInformation)

		// If key for the first day of month exists, we should assume all days do
		// and delete all 31 days one by one (silent error for days that don't exist)
		dayOfMonthSubkey := mw.config.Etcd.BaseKeyToWrite + "/" + regValue3
		if _, ok := (*etcdValues)[dayOfMonthSubkey+"/01"]; ok {
			for day := 1; day <= 31; day++ {
				var strDay string
				if day < 10 {
					strDay = "0" + fmt.Sprint(day)
				} else {
					strDay = fmt.Sprint(day)
				}

				support.DeleteFromEtcd(&mw.config.Etcd.CertPath, &mw.config.Etcd.Endpoints,
					dayOfMonthSubkey+"/"+strDay)
			}
		}
	}
}

// Things to perform before showing GUI
func (mw *MainWin) getConfigAndDBValues(exePath string) {
	var err error

	mw.useEtcd = false
	if err = mw.getConfigContentsFromYaml(exePath); err != nil {
		log.Fatal("Cannot get contents from yaml")
	}

	for _, address := range mw.config.Etcd.Endpoints {
		ipAndPort := strings.Split(address, ":")
		if mw.testSockConnect(ipAndPort[0], ipAndPort[1]) { // etcd exists lets use that for settings
			mw.useEtcd = true

			// if the cert path doesnt exist
			if _, err = os.Stat(mw.config.Etcd.CertPath); errors.Is(err, os.ErrNotExist) {
				walk.MsgBox(nil, "Fatal Error", "Fatal: "+err.Error(), walk.MsgBoxIconError)
			}

			mw.dbValues, err = support.ReadFromEtcd(&mw.config.Etcd.CertPath, &mw.config.Etcd.Endpoints, mw.config.Etcd.BaseKeyToWrite)
			if err != nil {
				walk.MsgBox(nil, "Fatal Error", "Fatal: "+err.Error()+"\nPossible authentication failure", walk.MsgBoxIconError)
				log.Fatal(err.Error())
			}
			mw.bwCurrentUsed, _ = strconv.ParseFloat(string(mw.dbValues[mw.config.Etcd.BaseKeyToWrite+"/"+regValue1]), 64)

			mw.deleteIfNewMonth(&mw.dbValues)
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
		mw.bwCurrentUsed, _ = strconv.ParseFloat(mw.GetRegStringValue(regValue1), 64)
	}
}

// Return the min and max numbers of the slice
func (mw *MainWin) getMinAndMaxOf(vals []float64) (float64, float64) {
	min, max := 0.0, 0.0
	for i, e := range vals {
		if i == 0 || e < min {
			min = e
		}
		if i == 0 || e > max {
			max = e
		}
	}

	return min, max
}

// Creates the bar graph png file
func (mw *MainWin) makeChart() {
	allValues := []float64{}
	bars := []chart.Value{}

	f, err := os.OpenFile("graph.png", os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		log.Fatal("Chart file could not be opened")
	}
	defer f.Close()

	for i := 1; i < 32; i++ {
		strNum := fmt.Sprint(i)
		if i < 10 {
			strNum = "0" + fmt.Sprint(i)
		}
		val, ok := mw.dbValues[mw.config.Etcd.BaseKeyToWrite+"/"+regValue3+"/"+strNum]
		if ok { // if we hit values that doesnt exist we have no more in the map (since they will always be ordered)
			fVal, _ := strconv.ParseFloat(string(val[:]), 64)
			allValues = append(allValues, fVal)
			bars = append(bars, chart.Value{Label: strNum, Value: fVal})
		}
	}
	min, max := mw.getMinAndMaxOf(allValues)

	if mw.lowerTextBox != nil {
		if bwMinFromText, _ := strconv.ParseFloat(mw.lowerTextBox.Text(), 64); bwMinFromText <= min {
			mw.bwMin = bwMinFromText
		} else {
			// if our value is higher than the minimum number ignore it and set it back to the minimum,
			// this helps avoid out of range, divide by zero and other nasty errors
			mw.lowerTextBox.SetText(fmt.Sprintf("%.3f", min))
			mw.bwMin = min
		}
	} else {
		mw.bwMin, _ = strconv.ParseFloat(string(mw.dbValues[mw.config.Etcd.BaseKeyToWrite+"/"+regValue5]), 64)
	}
	if mw.upperTextBox != nil {
		if bwMaxFromText, _ := strconv.ParseFloat(mw.upperTextBox.Text(), 64); bwMaxFromText >= max {
			mw.bwMax = bwMaxFromText
		} else {
			// if our value is lower than the maximum number ignore it and set it back to the maximum,
			// this helps avoid out of range, divide by zero and other nasty errors
			mw.upperTextBox.SetText(fmt.Sprintf("%.3f", max))
			mw.bwMax = max
		}
	} else {
		mw.bwMax, _ = strconv.ParseFloat(string(mw.dbValues[mw.config.Etcd.BaseKeyToWrite+"/"+regValue6]), 64)
	}

	graph := chart.BarChart{
		Background: chart.Style{
			Padding: chart.Box{
				Top:    3,
				Left:   0,
				Bottom: 18,
				Right:  8,
			},
		},
		DPI:    600,
		Width:  initialWinWidth,
		Height: graphImgHeight + 15,
		XAxis: chart.Style{
			Show:     true,
			FontSize: 2.0,
		},
		YAxis: chart.YAxis{
			Range: &chart.ContinuousRange{
				Min: mw.bwMin,
				Max: mw.bwMax,
			},
			Style: chart.Style{
				Show:     true,
				FontSize: 2.0,
			},
		},
		Bars: bars,
	}
	if err = graph.Render(chart.PNG, f); err != nil {
		log.Fatal("Chart could not be rendered")
	}
}

// Writes the characters for the graph
// func (mw *MainWin) populateGraph() string {
// 	var everything []string
// 	var everystring string
// 	var allValues []float64

// 	for i := 1; i < 32; i++ {
// 		strNum := fmt.Sprint(i)
// 		if i < 10 {
// 			strNum = "0" + fmt.Sprint(i)
// 		}
// 		val, ok := mw.dbValues[mw.config.Etcd.BaseKeyToWrite+"/"+regValue3+"/"+strNum]
// 		if ok { // if we hit values that doesnt exist we have no more in the map (since they will always be ordered)
// 			fVal, _ := strconv.ParseFloat(string(val[:]), 64)
// 			allValues = append(allValues, fVal)
// 			everything = append(everything, regValue3+"/"+strNum+":\t"+string(val))
// 		}
// 	}

// 	min, max := mw.getMinAndMaxOf(allValues)

// 	if mw.lowerTextBox != nil {
// 		if bwMinFromText, _ := strconv.ParseFloat(mw.lowerTextBox.Text(), 64); bwMinFromText <= min {
// 			mw.bwMin = bwMinFromText
// 		} else {
// 			// if our value is higher than the minimum number ignore it and set it back to the minimum,
// 			// this helps avoid out of range, divide by zero and other nasty errors
// 			mw.lowerTextBox.SetText(fmt.Sprintf("%.3f", min))
// 			mw.bwMin = min
// 		}
// 	} else {
// 		mw.bwMin, _ = strconv.ParseFloat(string(mw.dbValues[mw.config.Etcd.BaseKeyToWrite+"/"+regValue5]), 64)
// 	}
// 	if mw.upperTextBox != nil {
// 		if bwMaxFromText, _ := strconv.ParseFloat(mw.upperTextBox.Text(), 64); bwMaxFromText >= max {
// 			mw.bwMax = bwMaxFromText
// 		} else {
// 			// if our value is lower than the maximum number ignore it and set it back to the maximum,
// 			// this helps avoid out of range, divide by zero and other nasty errors
// 			mw.upperTextBox.SetText(fmt.Sprintf("%.3f", max))
// 			mw.bwMax = max
// 		}
// 	} else {
// 		mw.bwMax, _ = strconv.ParseFloat(string(mw.dbValues[mw.config.Etcd.BaseKeyToWrite+"/"+regValue6]), 64)
// 	}

// 	for i, line := range everything {
// 		if mw.bwMax > 0 {
// 			pct := (allValues[i] - mw.bwMin) / (mw.bwMax - mw.bwMin) * 100 // get percentage
// 			everystring += line + "\t" + strings.Repeat("|", int(pct)) + "\r\n"
// 		} else {
// 			everystring += line + "\t" + "\r\n"
// 		}
// 	}

// 	return everystring
// }

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

// Writes the final values before exiting program
func (mw *MainWin) writeClosingValuesToDB() {
	if mw.useEtcd {
		// Add leading zero to single digit days
		var strDayOfMonth string
		if time.Now().Day() < 10 {
			strDayOfMonth = "0" + fmt.Sprintf("%d", time.Now().Day())
		} else {
			strDayOfMonth = fmt.Sprintf("%d", time.Now().Day())
		}

		// then write to etcd
		support.WriteToEtcd(&mw.config.Etcd.CertPath, &mw.config.Etcd.Endpoints,
			mw.config.Etcd.BaseKeyToWrite+"/"+regValue1, fmt.Sprintf("%.0f", mw.bwCurrentUsed))
		support.WriteToEtcd(&mw.config.Etcd.CertPath, &mw.config.Etcd.Endpoints,
			mw.config.Etcd.BaseKeyToWrite+"/"+regValue2, fmt.Sprintf("%.3f", mw.gbPerDayLeft))
		support.WriteToEtcd(&mw.config.Etcd.CertPath, &mw.config.Etcd.Endpoints,
			mw.config.Etcd.BaseKeyToWrite+"/"+regValue3+"/"+strDayOfMonth, fmt.Sprintf("%.3f", mw.gbPerDayLeft))
		support.WriteToEtcd(&mw.config.Etcd.CertPath, &mw.config.Etcd.Endpoints,
			mw.config.Etcd.BaseKeyToWrite+"/"+regValue4, fmt.Sprintf("%d", int(time.Now().Month())))
		support.WriteToEtcd(&mw.config.Etcd.CertPath, &mw.config.Etcd.Endpoints,
			mw.config.Etcd.BaseKeyToWrite+"/"+regValue5, fmt.Sprintf("%.3f", mw.bwMin))
		support.WriteToEtcd(&mw.config.Etcd.CertPath, &mw.config.Etcd.Endpoints,
			mw.config.Etcd.BaseKeyToWrite+"/"+regValue6, fmt.Sprintf("%.3f", mw.bwMax))
	} else {
		// or write to registry if no etcd
		mw.setSingleRegKeyValue(regValue1, fmt.Sprintf("%.0f", mw.bwCurrentUsed))
		mw.setSingleRegKeyValue(regValue2, fmt.Sprintf("%.3f", mw.gbPerDayLeft))
	}
}
