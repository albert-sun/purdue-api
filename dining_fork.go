package purdue_api

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"
)

// Location holds location information for simple
type Location struct {
	Name string // name, case sensitive
}

// Contains every single field for unmarshalling purposes,
// Changes to underlying JSOn structure:
// - Date field changed from string to time.Time for easy parsing
type forkDiningInfo struct {
	Location string
	Date     time.Time // CHANGED, originally string
	Notes    string    // almost always empty?
	Meals    []struct {
		ID     string
		Name   string // same, except for custom holiday
		Type   string
		Order  int
		Status string // Open, Closed, Unavailable
		Hours  struct {
			StartTime string // HH:MM:SS (24hr format)
			EndTime   string // HH:MM:SS (24hr format)
		}
		Stations []struct {
			Name  string
			Items []struct {
				ID           string // maybe for caching or searching?
				Name         string
				IsVegetarian bool
				Allergens    []struct { // why like this...
					Name  string
					Value bool
				}
			}
		}
	}
	Available bool // custom field, whether any meals available that day
}

// GetDiningLocation retrieves dining information for the passed date.
// Returns a struct pointer containing dining information, and error if any is encountered.
// TODO maybe keep HTTP client static within instance config for proxy support and whatnot
func GetDining(location *Location, date time.Time) (*forkDiningInfo, error) {
	// init http client prematurely
	httpClient := fasthttp.Client{}

	dateInput := date.Format(dateLayout) // convert to string
	menuURI := fmt.Sprintf("%s/%s/%s", menuAPIURL, location.Name, dateInput)
	response, err := compactGET(&httpClient, menuURI, fastHeaders(diningHeaders)) // make sure released
	if err != nil {                                                               // error performing request
		return nil, errors.Wrap(GenericRequestErr, err.Error())
	}
	defer fasthttp.ReleaseResponse(response)

	// unmarshal straight into struct, maybe convert allergens but lazy
	var diningInfo forkDiningInfo
	if err = json.Unmarshal(response.Body(), &diningInfo); err != nil {
		return nil, errors.Wrap(GenericParsingErr, "invalid dining json")
	}

	// populate custom fields
	diningInfo.Date = date
	for _, meal := range diningInfo.Meals {
		if meal.Status != "Unavailable" {
			diningInfo.Available = true
			break
		}
	}

	return &diningInfo, nil
}

// GetDiningRange retrieves dining information for a location within a range of days (positive or negative integer).
// Automatically stops checking once a date is marked as unavailable (meaning they haven't written the menus that far).
// Returns a slice of dining infos for each date ranged over, and the first error if any is encountered.
// TODO add functional options for ignoring or stopping at unavailable dining days (closed or too far ahead).
// TODO use goroutines for concurrency, temporarily leaving synchronous because of potential rate-limiting.
// TODO find better way to wrap errors
func GetDiningRange(location *Location, date time.Time, rng int) (map[time.Time]*forkDiningInfo, error) {
	diningInfos := make(map[time.Time]*forkDiningInfo, rng+1)

	// populate dates slice for iteration
	dates := make([]time.Time, rng+1)
	isNegative := boolToInt(rng >= 0)
	for i := 0; i <= abs(rng); i++ { // quick way to range dates?
		dates[i] = date.AddDate(0, 0, isNegative*i)
	}
	if isNegative == 1 { // reverse dates if needed
		reverseDates(dates)
	}

	// iterate and get info, return if unavailable found
	for _, currDate := range dates {
		diningInfo, err := GetDining(location, currDate)
		if err != nil {
			return nil, err
		}

		// check whether unavailable
		if !diningInfo.Available {
			break
		}

		diningInfos[currDate] = diningInfo
	}

	return diningInfos, nil
}

// GetDiningFull retrieves dining information for all locations in the near future.
// For each location, checks from the current date until meals are marked as unavailable or meals are empty.
// Returns a double map containing dining info for each location, and the first error if any is encountered.
// TODO use goroutines for concurrency, temporarily leaving synchronous because of potential rate-limiting.
func GetDiningFull(locations []*Location) (map[*Location]map[time.Time]*forkDiningInfo, error) {
	locationsDiningInfos := map[*Location]map[time.Time]*forkDiningInfo{} // initialize fat map

	// iterate through each location synchronously
	now := time.Now()
	for _, location := range locations {
		diningInfos, err := GetDiningRange(location, now, 365) // ok purdue
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("retrieval error for %s", location.Name))
		}

		locationsDiningInfos[location] = diningInfos
	}

	return locationsDiningInfos, nil
}
