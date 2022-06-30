package main

import (
	"_nate/CalcBandwidth/src/myetcd"
	"os"
	"strconv"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"golang.org/x/sys/windows/registry"
)

const regKeyBranch = `SOFTWARE\NateMorrison\CalcBandwidth`
const regValue1 = "bwCurrentUsed"
const regValue2 = "prevBwCurrentUsed"
const regValue3 = "daysLeftInMonth"
const regValue4 = "prevDaysLeftInMonth"
const initialWinWidth = 975
const initialWinHeight = 175

// Global pointers
var doEtcd bool
var mainWin *walk.MainWindow
var resultMsgBox *walk.TextEdit
var bwTextBox *walk.LineEdit
var pushButton *walk.PushButton
var key *registry.Key
var bwCurrentUsed, prevBwCurrentUsed, daysLeftInMonth, prevDaysLeftInMonth, prevBwAtProgStart, prevDaysAtProgStart *float64

func main() {
	var config Config
	bwCurrentUsed, prevBwCurrentUsed, daysLeftInMonth, prevDaysLeftInMonth = new(float64), new(float64), new(float64), new(float64)
	doEtcd = false

	if testSockConnect("10.150.30.18", "2379") {
		doEtcd = true

		exePath, _ := os.Getwd()
		if exePath[len(exePath)-4:] == "\\src" || exePath[len(exePath)-4:] == "\\bin" {
			exePath = exePath[:len(exePath)-4]
		}

		config, _ = getConfigContentsFromYaml(exePath + "\\config.yml")
		etcdValues, _ := myetcd.ReadFromEtcd(&config.Etcd.CertPath, &config.Etcd.Endpoints, config.Etcd.BaseKeyToWrite)
		*bwCurrentUsed, _ = strconv.ParseFloat(etcdValues[config.Etcd.BaseKeyToWrite+"/"+regValue1], 64)
		*prevBwCurrentUsed, _ = strconv.ParseFloat(etcdValues[config.Etcd.BaseKeyToWrite+"/"+regValue2], 64)
		*daysLeftInMonth, _ = strconv.ParseFloat(etcdValues[config.Etcd.BaseKeyToWrite+"/"+regValue3], 64)
		*prevDaysLeftInMonth, _ = strconv.ParseFloat(etcdValues[config.Etcd.BaseKeyToWrite+"/"+regValue4], 64)
	} else {
		key = new(registry.Key)
		*key = getRegKeyValues()
		*bwCurrentUsed, _ = strconv.ParseFloat(GetRegStringValue(regValue1), 64)
		*prevBwCurrentUsed, _ = strconv.ParseFloat(GetRegStringValue(regValue2), 64)
		*daysLeftInMonth, _ = strconv.ParseFloat(GetRegStringValue(regValue3), 64)
		*prevDaysLeftInMonth, _ = strconv.ParseFloat(GetRegStringValue(regValue4), 64)
	}

	// record previous settings now since these values might change a few times at runtime
	prevBwAtProgStart, prevDaysAtProgStart = new(float64), new(float64)
	*prevBwAtProgStart = *bwCurrentUsed     // copy value of pointer into new pointer, NOT point to the same object (since that num will change)
	*prevDaysAtProgStart = *daysLeftInMonth // copy value of pointer into new pointer, NOT point to the same object (since that num will change)

	MainWindow{
		AssignTo: &mainWin,
		Title:    "Bandwidth Calculator",
		Size:     Size{initialWinWidth, initialWinHeight},
		MinSize:  Size{initialWinWidth, initialWinHeight},
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

	closingFunctions(&config)
}
