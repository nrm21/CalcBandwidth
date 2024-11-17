package main

import (
	"testing"
	"time"
)

func TestGeneral(t *testing.T) {
	mw := new(MainWin)

	t.Run("GetConfigContentsFromYaml: Get config from empty path", func(t *testing.T) {
		expected := ""
		if err := mw.getConfigContentsFromYaml(""); err == nil {
			t.Errorf("ERROR: Expected: an fs.PathError but got none")
		}
		if expected != mw.config.Etcd.BaseKeyToWrite {
			t.Errorf("ERROR: Expected: %q got: %q", expected, mw.config.Etcd.BaseKeyToWrite)
		}
	})
}

func TestCalcMonthDays(t *testing.T) {
	mw := new(MainWin)

	tests := []struct {
		name     string
		month    time.Month
		year     int
		expected float64
	}{
		{"Check 31 days in Dec", 12, 2023, 31},
		{"Check 28 days in Feb", 2, 2023, 28},
		{"Check 29 days in Feb of leap year", 2, 2024, 29},
		{"Check 30 days in Jun", 6, 2023, 30},
		{"Check negative month", -6, 2023, 30},
		{"Check negative year", 6, -2023, 30},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			results := mw.calcMonthDays(test.month, test.year)
			if results != test.expected {
				t.Errorf("ERROR: Expected: %f got: %f", test.expected, results)
			}
		})
	}
}

func TestGetMinAndMaxOf(t *testing.T) {
	mw := new(MainWin)

	tests := []struct {
		name   string
		array  []float64
		expMin float64
		expMax float64
	}{
		{"Check normal array", []float64{1.456, 6345.6546, 8.324525}, 1.456, 6345.6546},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			min, max := mw.getMinAndMaxOf(test.array)
			if min != test.expMin && max != test.expMax {
				t.Errorf("ERROR: Expected: %f, %f got: %f, %f", min, max, test.expMin, test.expMax)
			}
		})
	}
}
