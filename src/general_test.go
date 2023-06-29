package main

import (
	"os"
	"testing"
	"time"
)

func TestGetConfigContentsFromYaml(t *testing.T) {
	result := Config{}

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
	if exePath[len(exePath)-4:] == "\\src" || exePath[len(exePath)-4:] == "\\bin" {
		exePath = exePath[:len(exePath)-4]
	}
	exePath = exePath + "\\config.yml"
	result, _ = getConfigContentsFromYaml(exePath)

	for _, test := range tests {
		switch test.field {
		case "BaseKeyToWrite":
			if test.expected != result.Etcd.BaseKeyToWrite {
				t.Errorf("ERROR: Expected: %q got: %q", test.expected, result.Etcd.BaseKeyToWrite)
			}
		case "CertPath":
			if test.expected != result.Etcd.CertPath {
				t.Errorf("ERROR: Expected: %q got: %q", test.expected, result.Etcd.CertPath)
			}
		case "Endpoints":
			if test.expected != result.Etcd.Endpoints[0] {
				t.Errorf("ERROR: Expected: %q got: %q", test.expected, result.Etcd.CertPath)
			}
		}
	}
}

func TestCalcMonthDays(t *testing.T) {
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
		results := calcMonthDays(test.InputMonth, test.InputYear)
		if results != test.Expected {
			t.Errorf("ERROR: Expected: %f got: %f", test.Expected, results)
		} else {
			t.Logf("PASS: Got: %f", results)
		}
	}
}
