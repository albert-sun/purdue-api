package purdue_api

import (
	"strings"
	"time"

	"github.com/pkg/errors"
)

// Date contains year, month, day of date, pretty self-explanatory.
type Date struct {
	Year  int `json:"year"`
	Month int `json:"month"`
	Day   int `json:"day"`
}

// Purdue Dining API - Dining court opening times, menu options, and other information (?)
// Information about each location each day is structured into one struct containing meals within a map[string]string.
type DiningDayInfo struct {
	Location string            `json:"location"`
	Date     `json:"date"`     // easy parsing
	Meals    map[string]string `json:"meals"`
}

var InvalidLocationErr = errors.New("invalid dining location")
var InvalidDateErr = errors.New("invalid date format") // for string passed date

// Valid dining options are retrieved from https://dining.purdue.edu/menus/
var validDining = []string{
	"earhart", "ford", "hillenbrand", "wiley", "windsor", // dining courts
	"1bowl", "all american dining room", "pete's za", "the gathering place", // other meal-swipe dining
	"earhart on-the-go!", "ford on-the-go!", "knoy on-the-go!", "lawson on-the-go!", "lilly on-the-go!",
	"windsor on-the-go!", // on-the-go (seriously, who even eats this?)
}

// Random constants not declared inside functions
const dateLayout = "2006-01-02" // YYYY-MM-DD

// DiningGetDay retrieves the day's dining for one dining "location" (just meal courts?).
// Accepts date as a string of format YYYY-MM-DD or a time.Time which is parsed into the former.
// Returns a populated pointer if successful, otherwise returns an error (refer to above errors and comments)
func DiningGetDay(location string, date interface{}) (*DiningDayInfo, error) {
	var err error

	// check whether valid dining location passed
	if stringArrContains(validDining, strings.ToLower(location)) {
		return nil, InvalidLocationErr
	}

	// check date formatting and convert to time.Time if needed
	var parsedDate time.Time
	switch date.(type) {
	case string:
		parsedDate, err = time.Parse(dateLayout, date.(string))
		if err != nil {
			return nil, InvalidDateErr
		}
		// can't fallthrough...
	case time.Time:
		parsedDate = date.(time.Time) // for convenience purposes
	}

	// prepare actual info struct before performing requests
	diningInfo := DiningDayInfo{
		Date: Date{
			Year:  parsedDate.Year(),
			Month: int(parsedDate.Month()), // one-indexed
			Day:   parsedDate.Day(),
		},
	} // don't want another struct just for date

	return &diningInfo, nil
}
