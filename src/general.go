package main

import (
	"_nate/CalcBandwidth/src/myetcd"
	"fmt"
	"log"
	"math"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/nrm21/support"
	"golang.org/x/sys/windows/registry"
	"gopkg.in/yaml.v2"
)

// Config struct
type Config struct {
	Etcd struct {
		// var name has to be uppercase here or it won't work
		Endpoints      []string      `yaml:"endpoints"`
		BaseKeyToWrite string        `yaml:"baseKeyToWrite"`
		Timeout        int           `yaml:"timeout"`
		SleepSeconds   time.Duration `yaml:"sleepSeconds"`
		CertPath       string        `yaml:"certpath"`
	}
}

// Unmarshals the config contents from file into memory
func getConfigContentsFromYaml(filename string) (Config, error) {
	var conf Config
	file, err := support.ReadConfigFileContents(filename)
	if err != nil {
		return conf, err
	}
	err = yaml.Unmarshal(file, &conf)
	if err != nil {
		return conf, err
	}

	return conf, nil
}

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
	const bwLimitGBs float64 = 1229
	currentYear := time.Now().UTC().Year()
	currentMonth := time.Now().UTC().Month()

	// find the number of days since the first of the month (excluding today)
	hoursSinceMonthStart := time.Since(time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, time.UTC)).Hours()
	totalDaysInMonth := calcMonthDays(currentMonth, currentYear)
	gbPerDay := bwLimitGBs / totalDaysInMonth
	gbAllowedSoFar := math.Round(bwLimitGBs / totalDaysInMonth * (hoursSinceMonthStart / 24))
	gbLeftToUse := bwLimitGBs - bwCurrentUsed
	daysLeftInMonth = totalDaysInMonth - (hoursSinceMonthStart / 24)
	gbPerDayLeft = gbLeftToUse / daysLeftInMonth
	bwDifferential := gbAllowedSoFar - bwCurrentUsed

	output := fmt.Sprintf("Fractional days left in month:      %.3f         (Days this month:  %d)\r\n", daysLeftInMonth, int(totalDaysInMonth))
	output += fmt.Sprintf("Bandwidth allowed up to today:   %.0f GB    (Used / Differential / Left:  %.0f / %.0f / %d GB)\r\n",
		gbAllowedSoFar, bwCurrentUsed, bwDifferential, int(gbLeftToUse))
	output += fmt.Sprintf("Bandwidth per day remaining:     %.3f GB  (Daily average:  %.3f GB)\r\n", gbPerDayLeft, gbPerDay)

	return output
}

// Calculates and outputs the necessary text to populate the main window
func setToRegAndCalc() {
	var err error
	numWithoutSpace := strings.TrimSpace(bwTextBox.Text())
	bwCurrentUsed, err = strconv.ParseFloat(numWithoutSpace, 64)
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

// Get a string value from the registry
func GetRegStringValue(regStr string) string {
	value, _, err := key.GetStringValue(regStr)
	if err != nil {
		log.Fatalf("Error reading registry value %s, exiting program", regStr)
	}

	return value
}

// Checks a socket connection and returns bool of if open or not
func testSockConnect(host string, port string) bool {
	conn, _ := net.DialTimeout("tcp", net.JoinHostPort(host, port), 500*time.Millisecond)
	if conn != nil {
		defer conn.Close()

		return true
	} else {
		return false
	}
}

// Things to perform just before exit
func closingFunctions(config *Config) {
	// do closing writes to DB
	if doEtcd {
		myetcd.WriteToEtcd(&config.Etcd.CertPath, &config.Etcd.Endpoints,
			config.Etcd.BaseKeyToWrite+"/"+regValue1, fmt.Sprintf("%.0f", bwCurrentUsed))
		myetcd.WriteToEtcd(&config.Etcd.CertPath, &config.Etcd.Endpoints,
			config.Etcd.BaseKeyToWrite+"/"+regValue2, fmt.Sprintf("%.3f", gbPerDayLeft))
		myetcd.WriteToEtcd(&config.Etcd.CertPath, &config.Etcd.Endpoints,
			config.Etcd.BaseKeyToWrite+"/"+regValue3, fmt.Sprintf("%.3f", daysLeftInMonth))
	} else {
		setSingleRegKeyValue(regValue1, fmt.Sprintf("%.0f", bwCurrentUsed))
		setSingleRegKeyValue(regValue2, fmt.Sprintf("%.3f", gbPerDayLeft))
		setSingleRegKeyValue(regValue3, fmt.Sprintf("%.3f", daysLeftInMonth))
	}
}
