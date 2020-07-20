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
	Method       string         `json:"method"`
	Pattern      string         `json:"pattern"`
	AppendFilter []AppendFilter `json:"append_filter"`
	CheckParam   []CheckParam   `json:"check_param"`
	CheckJSON    []CheckJSON    `json:"check_json"`
}

// AppendFilter ... struct with API filter to check and
// an array of allowed values
type AppendFilter struct {
	FilterKey string        `json:"filter_key"`
	Values    []interface{} `json:"values"`
}

// CheckParam ... struct with URL params to check and
// an array of allowed values
type CheckParam struct {
	Param         string        `json:"param"`
	AllowedValues []interface{} `json:"allowed_values"`
}

// CheckJSON ... struct with keys and
// an array of allowed values to check for in posted JSONs
type CheckJSON struct {
	Key           string        `json:"key"`
	AllowedValues []interface{} `json:"allowed_values"`
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
