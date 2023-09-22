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
const initialWinWidth = 850
const initialWinHeight = 1000
const graphImgHeight = 750

type MainWin struct {
	*walk.MainWindow
	resultMsgBox, barGraphBox                 *walk.TextEdit
	bwTextBox, lowerTextBox, upperTextBox     *walk.LineEdit
	fillPrevDaysCheckBox                      *walk.CheckBox
	graphImage                                *walk.ImageView
	key                                       *registry.Key
	config                                    Config
	dbValues                                  map[string][]byte
	useEtcd                                   bool
	bwCurrentUsed, gbPerDayLeft, bwMin, bwMax float64
}

func main() {
	var appIcon, _ = walk.NewIconFromResourceId(2) // number 2 is resource ID printed by rsrc.exe when using v0.10+
	mw := new(MainWin)

	// first get values from conf
	exePath, _ := os.Getwd()
	if exePath[len(exePath)-4:] == "\\cmd" || exePath[len(exePath)-4:] == "\\bin" {
		exePath = exePath[:len(exePath)-4]
	}
	mw.getConfigAndDBValues(exePath + "\\config.yml")

	MainWindow{
		AssignTo: &mw.MainWindow,
		Icon:     appIcon,
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
								AssignTo: &mw.bwTextBox,
								Text:     strconv.FormatFloat(mw.bwCurrentUsed, 'f', -1, 64),
								// OnKeyPress event fires before we get the number, need to use OnKeyUp
								OnKeyUp: func(keystroke walk.Key) {
									if keystroke >= walk.Key0 && keystroke <= walk.Key9 { // if a digit key pressed
										mw.resultMsgBox.SetText(mw.calculateBandwidth())
									}
								},
							},
							PushButton{
								Text: "        Press to calculate        ",
								OnClicked: func() {
									// write values to db, reload them and update gui
									mw.resultMsgBox.SetText(mw.calculateBandwidth())
									mw.writeValuesToDB()
									mw.getConfigAndDBValues(exePath + "\\config.yml")
									mw.makeChart()
									mw.refreshImage()
								},
							},
						},
					},
				},
			},
			HSplitter{
				Children: []Widget{
					TextEdit{
						AssignTo: &mw.resultMsgBox,
						MinSize:  Size{initialWinWidth, 70},
						ReadOnly: true,
						Font: Font{
							Family:    "Ariel",
							PointSize: 17,
						},
						Text: mw.calculateBandwidth(),
						OnBoundsChanged: func() {
							mw.resultMsgBox.SetWidth(mw.Width() - 35)
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
								AssignTo: &mw.lowerTextBox,
								Text:     string(mw.dbValues[mw.config.Etcd.BaseKeyToWrite+"/"+regValue5]),
							},
							Label{
								Text: "Upper graph range:",
							},
							LineEdit{
								AssignTo: &mw.upperTextBox,
								Text:     string(mw.dbValues[mw.config.Etcd.BaseKeyToWrite+"/"+regValue6]),
							},
							Label{
								Text: "Fill empty previous days:",
							},
							CheckBox{
								AssignTo: &mw.fillPrevDaysCheckBox,
								Checked:  true,
							},
						},
					},
				},
			},
			HSplitter{
				Children: []Widget{
					ImageView{
						AssignTo: &mw.graphImage,
					},
				},
			},
		},
	}.Create()

	// make the bar graph
	mw.makeChart()
	mw.refreshImage()
	mw.Run()

	mw.writeValuesToDB()
}
