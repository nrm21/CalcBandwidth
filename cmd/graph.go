package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/lxn/walk"
	"github.com/wcharczuk/go-chart"
)

const graphFilename = "graph.png"

// iterates through all possible numbers 1 to 31 to see if the exist in the DB
// to determine if there is bar data for each day in the map structure
func getBarsData(mw *MainWin) ([]float64, []chart.Value) {
	allValues := []float64{}
	bars := []chart.Value{}

	for i := 1; i < 32; i++ {
		strNum := fmt.Sprint(i)
		if i < 10 {
			strNum = "0" + fmt.Sprint(i)
		}
		val, ok := mw.config.dbValues[mw.config.Etcd.BaseKeyToWrite+"/"+regValue3+"/"+strNum]
		if ok { // if we hit values that doesnt exist we have no more in the map (since they will always be ordered)
			fVal, _ := strconv.ParseFloat(string(val[:]), 64)
			allValues = append(allValues, fVal)
			bars = append(bars, chart.Value{Label: strNum, Value: fVal})
		}
	}

	return allValues, bars
}

// Return the min and max numbers of the slice in one iteration of the array
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

func setGraphUpperLowerExtents(mw *MainWin, min, max float64) {
	if mw.lowerTextBox != nil {
		if bwMinFromText, _ := strconv.ParseFloat(mw.lowerTextBox.Text(), 64); bwMinFromText <= min {
			if bwMinFromText < 0 { // if min below zero just set to zero
				mw.lowerTextBox.SetText("0")
				mw.config.bwMin = 0
			} else {
				mw.config.bwMin = bwMinFromText
			}
		} else {
			// if our value is higher than the minimum number ignore it and set it back to the minimum,
			// this helps avoid out of range, divide by zero and other nasty errors
			mw.lowerTextBox.SetText(fmt.Sprintf("%.3f", min))
			mw.config.bwMin = min
		}
	} else {
		mw.config.bwMin, _ = strconv.ParseFloat(string(mw.config.dbValues[mw.config.Etcd.BaseKeyToWrite+"/"+regValue5]), 64)
	}
	if mw.upperTextBox != nil {
		if bwMaxFromText, _ := strconv.ParseFloat(mw.upperTextBox.Text(), 64); bwMaxFromText >= max {
			mw.config.bwMax = bwMaxFromText
		} else {
			// if our value is lower than the maximum number ignore it and set it back to the maximum,
			// this helps avoid out of range, divide by zero and other nasty errors
			mw.upperTextBox.SetText(fmt.Sprintf("%.3f", max))
			mw.config.bwMax = max
		}
	} else {
		mw.config.bwMax, _ = strconv.ParseFloat(string(mw.config.dbValues[mw.config.Etcd.BaseKeyToWrite+"/"+regValue6]), 64)
	}
}

// Creates the bar graph png file
func (mw *MainWin) makeChart() {
	allValues, bars := getBarsData(mw)

	// only render new graph if we have a dataset, otherwise just use the previously rendered png file
	if len(bars) != 0 {
		// find the smallest and largest graph bar so know what extents to use for our graph
		min, max := mw.getMinAndMaxOf(allValues)

		// Instead of setting min and max exactly, lets just round to the nearest
		// integer below and above respectively to keep the graph somewhat pretty
		min = float64(int(min))
		max = float64(int(max)) + 1
		setGraphUpperLowerExtents(mw, min, max)

		// setup the values for the y axis based on min and max we are looking at
		yaxisticks := []chart.Tick{}
		f := 0.0
		for f <= mw.config.bwMax {
			if f >= mw.config.bwMin {
				yaxisticks = append(yaxisticks, chart.Tick{Value: f, Label: fmt.Sprintf("%.1f", f)})
			}
			// if range is small/big enough change step size
			if mw.config.bwMax-mw.config.bwMin <= 5 {
				f += .5
			} else if mw.config.bwMax-mw.config.bwMin >= 25 {
				f += 5
			} else {
				f += 1
			}
		}

		graph := chart.BarChart{
			Background: chart.Style{
				Padding: chart.Box{
					Top:    10,
					Left:   -2,
					Bottom: 23,
					Right:  10,
				},
			},
			DPI:      1200,
			Width:    initialWinWidth + 75,
			Height:   graphImgHeight + 15,
			BarWidth: 30,
			XAxis: chart.Style{
				Show:     true,
				FontSize: 1.2,
			},
			YAxis: chart.YAxis{
				Ticks: yaxisticks,
				Range: &chart.ContinuousRange{
					Min: mw.config.bwMin,
					Max: mw.config.bwMax,
				},
				Style: chart.Style{
					Show:     true,
					FontSize: 1.2,
				},
				ValueFormatter: chart.FloatValueFormatter,
			},
			Bars: bars,
		}
		// open the file we will write too
		file, err := os.OpenFile(graphFilename, os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			log.Fatal("Chart file could not be opened")
		}
		defer file.Close()

		if err = graph.Render(chart.PNG, file); err != nil {
			log.Fatal("Chart could not be rendered")
		}
	}
}

// Gets the image from file and puts into walk.image struct so we can use
func (mw *MainWin) getImageFromFile() walk.Image {
	img, err := walk.NewImageFromFileForDPI(graphFilename, 600)
	if err != nil {
		log.Fatal("Cannot load new image")
	}

	return img
}

// Sets (and resets) the widget that holds the graph so we can refresh it in program
func (mw *MainWin) refreshImage() {
	// create new image from recalculations
	if imageView, err := walk.NewImageView(mw.graphImage.Parent()); err == nil {
		imageView.SetImage(mw.getImageFromFile())
		imageView.SetMinMaxSize(walk.Size{initialWinWidth, graphImgHeight},
			walk.Size{initialWinWidth, graphImgHeight})
		imageView.SetMargin(4)
		imageView.SetMode(walk.ImageViewModeZoom)

		// and dispose old image widget, and reassign the new one
		mw.graphImage.Dispose()
		mw.graphImage = imageView
	} else {
		log.Fatal("Cannot create new imageview")
	}
}
