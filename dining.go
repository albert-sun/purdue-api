package purdue_api

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/remeh/sizedwaitgroup"
	"github.com/valyala/fasthttp"
)

// TODO maybe mirror the "name" field or whatnot in struct?

// Item represents a single served item
type Item struct {
	Name       string   `json:"Name"`
	Vegetarian bool     `json:"Vegetarian"`
	Allergens  []string `json:"Allergens"`
}

// Station represents a single serving station containing items
type Station struct {
	Name    string `json:"Name"`
	Items   []Item `json:"Items"`
	IconURL string `json:"IconURL"`
}

// Meal contains meal information including opening hours and station dishes (meal name is key in map)
type Meal struct {
	Name          string             `json:"Name"` // just in case
	Open          bool               `json:"Open"`
	Type          string             `json:"Type"`
	StartingHours string             `json:"StartingHours"` // sick of parsing...
	EndingHours   string             `json:"EndingHours"`   // sick of parsing...
	Stations      map[string]Station `json:"Stations"`
}

// DiningInfo contains information about one location's meals for one day.
type DiningInfo struct {
	Notes     string          `json:"Notes"`
	Available bool            `json:"Available"` // whether has menu info
	Location  string          `json:"Location"`
	Meals     map[string]Meal `json:"Meals"`
}

// All the raw info no-one likes dealing with...
// Who even designed this JSON response anyway?
type rawDiningInfo struct {
	Location string `json:"Location"`
	Notes    string `json:"Notes"` // needed?
	Meals    []struct {
		Name   string `json:"Name"` // different, holiday meals?
		Type   string `json:"Type"`
		Status string `json:"Status"` // open or closed
		Hours  struct {
			StartTime string `json:"StartTime"` // HH:MM:SS 24hr
			EndTime   string `json:"EndTime"`   // HH:MM:SS 24hr
		} `json:"Hours"`
		Stations []struct {
			Name    string `json:"Name"`
			IconURL string `json:"IconUrl"`
			Items   []struct {
				Name       string `json:"Name"`
				Vegetarian bool   `json:"IsVegetarian"`
				Allergens  []struct {
					Name  string `json:"Name"`
					Value bool   `json:"Value"`
				} `json:"Allergens"`
			}
		}
	} `json:"Meals"`
}

// Generic errors wrapped to make others
var GenericParameterErr = errors.New("invalid parameter")
var GenericRequestErr = errors.New("error performing request") // wrap??
var GenericParsingErr = errors.New("error parsing")
var GenericConfigErr = errors.New("invalid config") // checks len(locations) != 0

// More specific errors for different purposes
var InvalidLocationErr = errors.Wrap(GenericParameterErr, "invalid location")
var InvalidDayRangeErr = errors.Wrap(GenericParameterErr, "invalid day range")
var UninitializedConfigErr = errors.Wrap(GenericConfigErr, "uninitialized config")

// Valid dining options are retrieved from https://dining.purdue.edu/menus/
var DiningLocations = []string{
	"Earhart", "Ford", "Hillenbrand", "Wiley", "Windsor", // dining courts
	"1Bowl", "All American Dining Room", "Pete's Za", "The Gathering Place", // other meal-swipe dining
	"Earhart On-the-GO!", "Ford On-the-GO!", "Knoy On-the-GO!", "Lawson On-the-GO!", "Lilly On-the-GO!",
	"Windsor On-the-GO!", // on-the-go (seriously, who even eats this?)
}

// Random constants not declared inside functions
const menuAPIURL = "https://api.hfs.purdue.edu/menus/v2/locations" // just in case
const dateLayout = "2006-01-02"                                    // YYYY-MM-DD
var diningHeaders = map[string]string{
	"Accept": "application/json",
}

// TODO deal with locations being case sensitive, seriously Purdue?
// GetDining retrieves the day's dining for one dining "location" (just meal courts?).
// Returns a populated pointer if successful, otherwise returns an error (refer to above errors for errors.Is)
func (config *DiningConfig) GetDining(location string, date time.Time) (*DiningInfo, error) {
	var err error

	// check whether config has locations
	if len(config.Locations) == 0 {
		return nil, UninitializedConfigErr
	}

	// check whether valid dining location passed
	if !stringArrContains(DiningLocations, location) {
		return nil, InvalidLocationErr
	}

	// create URL, perform actual request, TODO for now return error if request error
	client := fasthttp.Client{} // TODO maybe X clients per config?
	dateInput := date.Format(dateLayout)
	menuURL := fmt.Sprintf("%s/%s/%s", menuAPIURL, location, dateInput)
	response, err := compactGET(&client, menuURL, fastHeaders(diningHeaders)) // make sure to release
	if err != nil {
		return nil, errors.Wrap(GenericRequestErr, err.Error())
	}

	// unmarshal into JSON, return if error
	var rawDining rawDiningInfo
	err = json.Unmarshal(response.Body(), &rawDining)
	if err != nil {
		return nil, errors.Wrap(GenericParsingErr, "invalid json format")
	}
	fasthttp.ReleaseResponse(response)

	// assume invalid location if not populated
	if rawDining.Location == "" {
		return nil, InvalidLocationErr
	}

	// begin painstaking parsing process, UGH
	diningInfo := DiningInfo{
		Notes:     rawDining.Notes,
		Available: true,
		Location:  rawDining.Location,
	}
	diningInfo.Meals = map[string]Meal{}

	// parse individual meals, get name, type, hours, and status (open or closed)
	for _, rawMeal := range rawDining.Meals {
		// flag if unavailable but keep parsing
		if rawMeal.Status == "Unavailable" {
			diningInfo.Available = true
		}

		meal := Meal{
			Name:          rawMeal.Name,
			Open:          rawMeal.Status == "Open",
			Type:          rawMeal.Type,
			StartingHours: rawMeal.Hours.StartTime,
			EndingHours:   rawMeal.Hours.EndTime,
			Stations:      map[string]Station{},
		}

		// parse individual stations for name, items, and iconURL (?)
		for _, rawStation := range rawMeal.Stations {
			station := Station{
				Name:    rawStation.Name,
				IconURL: rawStation.IconURL,
			}

			// parse individual items for name and allergens (in []string format)
			for _, rawItem := range rawStation.Items {
				item := Item{
					Name:       rawItem.Name,
					Vegetarian: rawItem.Vegetarian,
				}

				// parse "true" allergens into array of strings
				for _, allergen := range rawItem.Allergens {
					if allergen.Value {
						item.Allergens = append(item.Allergens, allergen.Name)
					}
				}

				station.Items = append(station.Items, item)
			}

			meal.Stations[rawStation.Name] = station
		}

		diningInfo.Meals[rawMeal.Name] = meal
	}

	return &diningInfo, nil
}

// TODO maybe use config for controlling number of concurrent goroutines?
// GetDiningDays gets dining info for one location over a range of dates (positive or negative number of days).
// Returns a populated map[int]*DiningInfo if successful (where int represents date range), else returns err != nil.
// For concurrency, the last error found is returned if any is found.
func (config *DiningConfig) GetDiningDays(location string, date time.Time, dayStart int, dayEnd int) (map[int]*DiningInfo, error) {
	// check whether config has locations
	if len(config.Locations) == 0 {
		return nil, UninitializedConfigErr
	}

	// check whether day range is valid (start <= end)
	if dayEnd < dayStart {
		return nil, InvalidDayRangeErr
	}

	// get array of times for processing
	var index int
	dates := make([]time.Time, dayEnd-dayStart+1)
	for i := dayStart; i <= dayEnd; i++ {
		dates[index] = date.AddDate(0, 0, i)
		index++
	}

	// concurrent goroutines calling GetDining
	var swgErr error // better way?
	swg := sizedwaitgroup.New(config.Concurrent)
	diningInfos := map[int]*DiningInfo{}
	for j := 0; j < len(dates); j++ {
		swg.Add()
		go func(loc string, k int, d time.Time) { // goroutine for swg
			defer swg.Done()

			diningInfo, err := config.GetDining(loc, d)
			if err != nil { // log MOST RECENT error found
				swgErr = err
				return
			}

			diningInfos[k+dayStart] = diningInfo // okay because no concurrent read and write
		}(location, j, dates[j])
	}

	swg.Wait()

	if swgErr != nil {
		return nil, swgErr
	}

	return diningInfos, nil
}

// GetDiningLocations gets all dining info (from what's been implemented) for a specific day.
// Returns a populated map[string]*DiningInfo if successful (where string represents location), else returns err !- nil.
// For concurrency, the last error found is returned if any is found.
func (config *DiningConfig) GetDiningLocations(date time.Time) (map[string]*DiningInfo, error) {
	// check whether config has locations
	if len(config.Locations) == 0 {
		return nil, UninitializedConfigErr
	}

	diningInfos := map[string]*DiningInfo{}

	// concurrent goroutines calling GetDining
	var swgErr error // better way?
	swg := sizedwaitgroup.New(config.Concurrent)
	for j := 0; j < len(DiningLocations); j++ {
		swg.Add()
		go func(loc string, k int, d time.Time) { // goroutine for swg
			defer swg.Done()

			diningInfo, err := config.GetDining(loc, d)
			if err != nil { // log MOST RECENT error found
				swgErr = err
				return
			}

			diningInfos[diningInfo.Location] = diningInfo // okay because no concurrent read and write
		}(DiningLocations[j], j, date)
	}

	swg.Wait()

	if swgErr != nil {
		return nil, swgErr
	}

	return diningInfos, nil
}
