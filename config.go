package purdue_api

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"
)

// rawDiningLocations holds raw location info from the menu API.
type rawDiningLocations struct {
	Locations []struct {
		Name string `json:"Name"` // difference between this and FormalName?
	} `json:"Location"`
}

// DiningConfig contains fields for use within the dining hooks.
type DiningConfig struct {
	Concurrent int      // number of concurrent goroutines
	Locations  []string // retrieved from API
}

// NewDiningConfig initializes and returns a new config with the provided parameters and up-to-date location data.
func NewDiningConfig(concurrent int) (*DiningConfig, error) {
	config := &DiningConfig{
		Concurrent: concurrent,
	}

	// get dining locations through API
	client := fasthttp.Client{}
	response, err := compactGET(&client, menuAPIURL, fastHeaders(diningHeaders))
	if err != nil {
		return nil, errors.Wrap(GenericRequestErr, err.Error())
	}

	// unmarshal into raw location data
	var rawLocations rawDiningLocations
	err = json.Unmarshal(response.Body(), &rawLocations)
	if err != nil {
		return nil, errors.Wrap(GenericParsingErr, err.Error())
	}
	fasthttp.ReleaseResponse(response)

	// parse into slice of strings for location names
	config.Locations = make([]string, len(rawLocations.Locations))
	for index, rawLocation := range rawLocations.Locations {
		config.Locations[index] = rawLocation.Name
	}

	return config, nil
}
