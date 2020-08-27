package purdue_api

import (
	"encoding/json"
	"fmt"
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

// DiningInfo contains information about one location's meals for one day.
type DiningInfo struct {
	Notes string `json:"notes"`
	// Add date maybe? String or time?
	Location string          `json:"location"`
	Meals    map[string]Meal `json:"meals"`
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

// More specific errors for different purposes
var InvalidLocationErr = errors.Wrap(GenericParameterErr, "invalid location")

// Valid dining options are retrieved from https://dining.purdue.edu/menus/
var validDining = []string{
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
// Accepts date as a string of format YYYY-MM-DD or a time.Time which is parsed into the former.
// Returns a populated pointer if successful, otherwise returns an error (refer to above errors and comments)
func GetDining(location string, date interface{}) (*DiningInfo, error) {
	var err error
	client := fasthttp.Client{} // TODO maybe X clients per config?

	// check whether valid dining location passed
	if !stringArrContains(validDining, location) {
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
		Notes:    rawDining.Notes,
		Location: rawDining.Location,
	}
	diningInfo.Meals = map[string]Meal{}

	// parse individual meals, get name, type, hours, and status (open or closed)
	for _, rawMeal := range rawDining.Meals {
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
