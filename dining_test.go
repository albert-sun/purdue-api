package purdue_api

import (
	"errors"
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
func TestGetDiningSuccess(test *testing.T) {
	now := time.Now()
	for _, location := range validDining { // test for each
		diningInfo, err := GetDining(location, now)
		if err != nil { // generic error
			test.Errorf("error checking %s: %s", location, err.Error())
		}

		if missing := checkDiningFields(diningInfo); missing != "" {
			test.Errorf("error checking %s: missing field %s", location, missing)
		}
	}
} // test valid request with all locations and current date
func TestGetDiningParameterLocation(test *testing.T) {
	_, err := GetDining("foo", time.Now())
	if !errors.Is(err, GenericParameterErr) {
		test.Errorf("should error: generic parameter")
	}
} // test invalid location not within array

// GetDiningRange
func TestGetDiningRangeSuccess(test *testing.T) {
	now := time.Now()
	diningInfos, err := GetDiningRange("Earhart", now, 0, 5)
	if err != nil { // generic error
		test.Errorf("error checking Earhart: %s", err.Error())
	}

	if len(diningInfos) == 0 {
		test.Errorf("error checking Earhart: zero-length map (expected 6)")
	} else if missing := checkDiningFields(diningInfos[0]); missing != "" {
		test.Errorf("error checking Earhart: missing field %s", missing)
	}
} // test valid request with Earhart and 5-day ahead range
