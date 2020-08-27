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

func TestGetDiningSuccess(test *testing.T) {
	now := time.Now()
	for _, location := range validDining { // test for each
		diningInfo, err := GetDining(location, now)
		if err != nil { // generic error
			test.Errorf("error with %s: %s", location, err.Error())
		}

		if missing := checkDiningFields(diningInfo); missing != "" {
			test.Errorf("error with %s: missing field %s", location, missing)
		}
	}
} // GetDining - test valid request with all locations and current date
func TestGetDiningParameterLocation(test *testing.T) {
	dateStr := time.Now().Format(dateLayout)
	_, err := GetDining("foo", dateStr)
	if !errors.Is(err, GenericParameterErr) {
		test.Errorf("should error: generic parameter")
	}
} // GetDining - test invalid location not within array
func TestGetDiningParameterDate(test *testing.T) {
	_, err := GetDining("earhart", "")
	if !errors.Is(err, GenericParameterErr) {
		test.Errorf("should error: generic parameter")
	}
} // GetDining - test invalid date with Earhart and empty string
