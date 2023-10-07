package main

import (
	"os"
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

	t.Run("GetConfigContentsFromYaml: Get config from correct YAML path", func(t *testing.T) {
		tests := []struct {
			field    string
			expected string
		}{
			{"Endpoints", "10.150.30.17:2379"},
			{"BaseKeyToWrite", "/nate/CalcBandwidth"},
			{"CertPath", "E:\\Documents\\_Nate\\Computer Related\\Private keys\\Etcd Certs"},
		}

		exePath, _ := os.Getwd()
		if exePath[len(exePath)-4:] == "\\bin" || exePath[len(exePath)-4:] == "\\cmd" || exePath[len(exePath)-4:] == "\\src" {
			exePath = exePath[:len(exePath)-4]
		}
		exePath = exePath + "\\config.yml"
		mw.getConfigContentsFromYaml(exePath)

		for _, test := range tests {
			switch test.field {
			case "BaseKeyToWrite":
				if test.expected != mw.config.Etcd.BaseKeyToWrite {
					t.Errorf("ERROR: Expected: %q got: %q", test.expected, mw.config.Etcd.BaseKeyToWrite)
				}
			case "CertPath":
				if test.expected != mw.config.Etcd.CertPath {
					t.Errorf("ERROR: Expected: %q got: %q", test.expected, mw.config.Etcd.CertPath)
				}
			case "Endpoints":
				if test.expected != mw.config.Etcd.Endpoints[0] {
					t.Errorf("ERROR: Expected: %q got: %q", test.expected, mw.config.Etcd.CertPath)
				}
			}
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
