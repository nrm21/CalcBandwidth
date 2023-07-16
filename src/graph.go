package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/lxn/walk"
	"github.com/wcharczuk/go-chart"
)

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
				Top:    10,
				Left:   -2,
				Bottom: 23,
				Right:  10,
			},
		},
		DPI:      1200,
		Width:    initialWinWidth + 75,
		Height:   graphImgHeight + 15,
		BarWidth: 35,
		XAxis: chart.Style{
			Show:     true,
			FontSize: 1.5,
		},
		YAxis: chart.YAxis{
			Range: &chart.ContinuousRange{
				Min: mw.bwMin,
				Max: mw.bwMax,
			},
			Style: chart.Style{
				Show:     true,
				FontSize: 1.5,
			},
		},
		Bars: bars,
	}
	if err = graph.Render(chart.PNG, f); err != nil {
		log.Fatal("Chart could not be rendered")
	}
}

// Gets the image from file and puts into walk.image struct so we can use
func (mw *MainWin) getImageFromFile() walk.Image {
	img, err := walk.NewImageFromFile("graph.png")
	if err != nil {
		log.Fatal("Cannot load new image")
	}

	return img
}

// Sets (and resets) the widget that holds the graph so we can refresh it in program
func (mw *MainWin) refreshImage() {
	// create new image from recalculations
	imageView, err := walk.NewImageView(mw.graphImage.Parent())
	if err != nil {
		log.Fatal("Cannot create new imageview")
	}
	imageView.SetImage(mw.getImageFromFile())
	imageView.SetMinMaxSize(walk.Size{initialWinWidth, graphImgHeight},
		walk.Size{initialWinWidth, graphImgHeight})
	imageView.SetMargin(4)
	imageView.SetMode(walk.ImageViewModeZoom)

	// and dispose old image widget, and reassign the new one
	mw.graphImage.Dispose()
	mw.graphImage = imageView
}