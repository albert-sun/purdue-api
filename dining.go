package purdue_api

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"
)

// TODO maybe mirror the "name" field or whatnot in struct?

// Item represents a single served item
type Item struct {
	Name       string   `json:"name"`
	Vegetarian bool     `json:"isVegetarian"`
	Allergens  []string `json:"allergens"`
}

// Station represents a single serving station containing items
type Station struct {
	Name    string `json:"name"`
	Items   []Item `json:"items"`
	IconURL string `json:"iconURL"`
}

// Meal contains meal information including opening hours and station dishes (meal name is key in map)
type Meal struct {
	Name          string             `json:"name"` // just in case
	Open          bool               `json:"open"`
	Type          string             `json:"type"`
	StartingHours string             `json:"startingHours"` // sick of parsing...
	EndingHours   string             `json:"endingHours"`   // sick of parsing...
	Stations      map[string]Station `json:"stations"`
}

// Purdue Dining API - Dining court opening times, menu options, and other information (?)
// Information about each location each day is structured into one struct containing meals within a map[string]string.
type DiningDayInfo struct {
	Notes    string          `json:"notes"`
	Location string          `json:"location"`
	Date     string          `json:"date"` // replace with time.Time?
	Meals    map[string]Meal `json:"meals"`
}

// All the raw info no-one likes dealing with...
// Who even designed this JSON response anyway?
type rawDiningDayInfo struct {
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

// More specific errors for different purposes
var InvalidLocationErr = errors.Errorf("%w: invalid location", GenericParameterErr)
var InvalidDateErr = errors.Errorf("%w: invalid date format", GenericParameterErr)

// Valid dining options are retrieved from https://dining.purdue.edu/menus/
var validDining = []string{
	"earhart", "ford", "hillenbrand", "wiley", "windsor", // dining courts
	"1bowl", "all american dining room", "pete's za", "the gathering place", // other meal-swipe dining
	"earhart on-the-go!", "ford on-the-go!", "knoy on-the-go!", "lawson on-the-go!", "lilly on-the-go!",
	"windsor on-the-go!", // on-the-go (seriously, who even eats this?)
}

// Random constants not declared inside functions
const menuAPIURL = "https://api.hfs.purdue.edu/menus/v2/locations" // just in case
const dateLayout = "2006-01-02"                                    // YYYY-MM-DD

// DiningGetDay retrieves the day's dining for one dining "location" (just meal courts?).
// Accepts date as a string of format YYYY-MM-DD or a time.Time which is parsed into the former.
// Returns a populated pointer if successful, otherwise returns an error (refer to above errors and comments)
func DiningGetDay(location string, date interface{}) (*DiningDayInfo, error) {
	var err error
	client := fasthttp.Client{} // TODO maybe X clients per config?

	// check whether valid dining location passed
	if stringArrContains(validDining, strings.ToLower(location)) {
		return nil, InvalidLocationErr
	}

	// check date formatting and convert to time.Time if needed
	var dateInput string
	switch date.(type) { // remove redundant lines?
	case string:
		dateInput = date.(string)
	case time.Time:
		parsedDate := date.(time.Time) // for convenience purposes
		dateInput = parsedDate.Format(dateLayout)
	}

	// create URL, perform actual request, TODO for now return error if request error
	menuURL := fmt.Sprintf("%s/%s/%s", menuAPIURL, location, dateInput)
	response, err := compactGET(&client, menuURL) // no need for headers, make sure to release
	if err != nil {
		return nil, errors.Errorf("%w, %w", GenericRequestErr, err) // does this work?
	}

	// unmarshal into JSON, return if error
	var rawDining rawDiningDayInfo
	err = json.Unmarshal(response.Body(), &rawDining)
	if err != nil {
		return nil, errors.Errorf("%w: invalid json format", GenericParsingErr)
	}
	fasthttp.ReleaseResponse(response)

	// begin painstaking parsing process, UGH
	diningInfo := DiningDayInfo{
		Notes:    rawDining.Notes,
		Location: rawDining.Location,
	}
	diningInfo.Meals = map[string]Meal{}
	for _, rawMeal := range rawDining.Meals { // parse meals
		meal := Meal{
			Name:          rawMeal.Name,
			Open:          rawMeal.Status == "Open",
			Type:          rawMeal.Type,
			StartingHours: rawMeal.Hours.StartTime,
			EndingHours:   rawMeal.Hours.EndTime,
			Stations:      map[string]Station{},
		}

		for _, rawStation := range rawMeal.Stations { // parse stations
			station := Station{
				Name:    rawStation.Name,
				IconURL: rawStation.IconURL,
			}

			for _, rawItem := range rawStation.Items { // parse items
				item := Item{
					Name:       rawItem.Name,
					Vegetarian: rawItem.Vegetarian,
				}

				// parse allergens into array
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
