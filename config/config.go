package config

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

// RoutesAllowed ... array of routes
type RoutesAllowed struct {
	Routes []route `json:"routes_allowed"`
}

type route struct {
	Method    string      `json:"method"`
	Pattern   string      `json:"pattern"`
	CheckJSON []CheckJSON `json:"check_json"`
}

type allowedType interface{}

// CheckJSON ... struct with keys and allowed values to check for in posted JSONs
type CheckJSON struct {
	Key           string        `json:"key"`
	AllowedValues []allowedType `json:"allowed_values"`
}

// RoutesConfig ... reads routes that should be available from json file
func RoutesConfig(fptr string) RoutesAllowed {
	// read json file
	data, err := ioutil.ReadFile(fptr)
	if err != nil {
		log.Fatal("Error reading file:", err)
	}

	// json data
	var routes RoutesAllowed

	// unmarshall it
	err = json.Unmarshal(data, &routes)
	if err != nil {
		log.Fatal("Error unmarshalling json:", err)
	}

	return routes
}
