package purdue_api

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"
)

// checkDiningFields checks whether specific response fields are populated.
// Returns a non-empty string specifying the missing field if any exist, otherwise returns an empty string.
func checkDiningFields(diningInfo *DiningInfo) string {
	if diningInfo.Location == "" { // manual
		return "location"
	} else if len(diningInfo.Meals) == 0 { // automatic
		return "meals"
	}

	return ""
}

// GetDining
func TestDiningSuccess(test *testing.T) {
	now := time.Now()
	for _, location := range DiningLocations { // test for each
		diningInfo, err := GetDining(location, now)
		if err != nil { // generic error
			test.Errorf("error for %s: %s", location, err.Error())
		}

		if location == "Earhart" {
			str, _ := json.Marshal(diningInfo)
			fmt.Println(string(str))
		}

		if missing := checkDiningFields(diningInfo); missing != "" {
			test.Errorf("error for %s: missing field %s", location, missing)
		}
	}
} // test valid request with all locations and current date
func TestDiningInvalidLocation(test *testing.T) {
	_, err := GetDining("foo", time.Now())
	if !errors.Is(err, GenericParameterErr) {
		test.Errorf("should error: generic parameter")
	}
} // test invalid location not within array

// GetDiningDays
func TestDiningDaysSuccess(test *testing.T) {
	now := time.Now()
	diningInfos, err := GetDiningDays("Earhart", now, 0, 5)
	if err != nil { // generic error
		test.Errorf("error: %s", err.Error())
	}

	if len(diningInfos) == 0 {
		test.Errorf("error: zero-length map (expected 6)")
	} else if missing := checkDiningFields(diningInfos[0]); missing != "" {
		test.Errorf("error: missing field %s", missing)
	}
} // test valid request with Earhart and 5-day ahead range
func TestDiningDaysInvalidDays(test *testing.T) {
	now := time.Now()
	_, err := GetDiningDays("Earhart", now, 5, 0)
	if !errors.Is(err, InvalidDayRangeErr) {
		test.Errorf("should error: invalid day range")
	}
} // test start integer after end integer

// GetDiningLocations
func TestDiningAllSuccess(test *testing.T) {
	now := time.Now()
	diningInfos, err := GetDiningLocations(now)
	if err != nil { // generic error
		test.Errorf("error: %s", err.Error())
	}

	if len(diningInfos) == 0 {
		test.Errorf("error: zero-length map (expected %d)", len(DiningLocations))
	} else if missing := checkDiningFields(diningInfos["Earhart"]); missing != "" {
		test.Errorf("error: missing field %s", missing)
	}
} // test valid request with current date
