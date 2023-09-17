package main

import (
	"os"
	"testing"
	"time"
)

func TestGetConfigContentsFromYaml(t *testing.T) {
	mw := new(MainWin)

	t.Run("Get config from empty path", func(t *testing.T) {
		expected := ""
		if err := mw.getConfigContentsFromYaml(""); err == nil {
			t.Errorf("ERROR: Expected: an fs.PathError but got none")
		}
		if expected != mw.config.Etcd.BaseKeyToWrite {
			t.Errorf("ERROR: Expected: %q got: %q", expected, mw.config)
		}
	})

	t.Run("Get config from correct YAML path", func(t *testing.T) {
		type testresults struct {
			field    string
			expected string
		}
		tests := []testresults{
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

	type testresults struct {
		InputMonth time.Month
		InputYear  int
		Expected   float64
	}
	tests := []testresults{
		{12, 2022, 31},
		{2, 2022, 28},
		{2, 2024, 29},
		{6, 2022, 30},
	}

	for _, test := range tests {
		results := mw.calcMonthDays(test.InputMonth, test.InputYear)
		if results != test.Expected {
			t.Errorf("ERROR: Expected: %f got: %f", test.Expected, results)
		} else {
			t.Logf("PASS: Got: %f", results)
		}
	}
}
