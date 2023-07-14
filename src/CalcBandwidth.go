package main

import (
	"os"
	"strconv"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"golang.org/x/sys/windows/registry"
)

const regKeyBranch = `SOFTWARE\NateMorrison\CalcBandwidth`
const regValue1 = "bwCurrentUsed"
const regValue2 = "bwPerDayRemaining"
const regValue3 = "dayOfMonth"
const regValue4 = "monthOfYear"
const regValue5 = "bwMin"
const regValue6 = "bwMax"
const initialWinWidth = 950
const initialWinHeight = 975

// Global pointers
var useEtcd bool
var mainWin *walk.MainWindow
var resultMsgBox, barGraphBox *walk.TextEdit
var bwTextBox, lowerTextBox, upperTextBox *walk.LineEdit
var pushButton *walk.PushButton
var key *registry.Key
var bwCurrentUsed, gbPerDayLeft, bwMin, bwMax float64
var dayOfMonth int

func main() {
	var config Config

	// first get values from conf
	exePath, _ := os.Getwd()
	if exePath[len(exePath)-4:] == "\\src" || exePath[len(exePath)-4:] == "\\bin" {
		exePath = exePath[:len(exePath)-4]
	}
	dbValues := getConfigAndDBValues(&config, exePath+"\\config.yml")

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
								Text:     strconv.FormatFloat(bwCurrentUsed, 'f', -1, 64),
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
									// write values to db, reload them and update gui
									setToRegAndCalc()
									writeClosingValuesToDB(&config)
									dbValues = getConfigAndDBValues(&config, exePath+"\\config.yml")
									barGraphBox.SetText(populateGraph(&config, dbValues))
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
						MinSize:  Size{initialWinWidth, 60},
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
			HSplitter{
				Children: []Widget{
					ScrollView{
						Layout: HBox{
							MarginsZero: true,
						},
						Children: []Widget{
							Label{
								Text: "Lower graph range:",
							},
							LineEdit{
								AssignTo: &lowerTextBox,
								Text:     string(dbValues[config.Etcd.BaseKeyToWrite+"/"+regValue5]),
							},
							Label{
								Text: "Upper graph range:",
							},
							LineEdit{
								AssignTo: &upperTextBox,
								Text:     string(dbValues[config.Etcd.BaseKeyToWrite+"/"+regValue6]),
							},
						},
					},
				},
			},
			HSplitter{
				Children: []Widget{
					TextEdit{
						AssignTo: &barGraphBox,
						MinSize:  Size{initialWinWidth, 750},
						ReadOnly: true,
						Font: Font{
							Family:    "Ariel",
							PointSize: 17,
						},
						Text: populateGraph(&config, dbValues),
						OnBoundsChanged: func() {
							barGraphBox.SetWidth(mainWin.Width() - 35)
						},
					},
				},
			},
		},
	}.Run()

	writeClosingValuesToDB(&config)
}
