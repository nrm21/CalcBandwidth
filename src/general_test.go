package main

import (
	"testing"
	"time"
)

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
