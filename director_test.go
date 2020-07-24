package dockerguard

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"reflect"
	"strings"
	"testing"
)

func TestFindJSONKey(t *testing.T) {
	// read file
	data, err := ioutil.ReadFile("test_jsons/create_service.json")
	if err != nil {
		fmt.Print(err)
	}

	// read JSON string and decode it to map
	var decoded map[string]interface{}
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		panic(err)
	}

	// simple non-nested
	testFindNested(t, decoded, []string{"Name"}, "web")

	// nested
	testFindNested(t, decoded, []string{"TaskTemplate", "LogDriver", "Options", "max-size"}, "10M")
	testFindNested(t, decoded, []string{"TaskTemplate", "ContainerSpec", "Image"}, "nginx:alpine")

	// 'Labels'
	testFindNested(t, decoded, []string{"Labels", "foo"}, "bar")

	expectedJSONString := []byte(`[
		{
		  "Protocol": "tcp",
		  "PublishedPort": 8080,
		  "TargetPort": 80
		}
	      ]`)
	expectedJSON := make([]map[string]interface{}, 0)
	err = json.Unmarshal(expectedJSONString, &expectedJSON)
	if err != nil {
		panic(err)
	}
	// this yields a type mismatch ([]interface vs []map[string]interface{})
	testFindNested(t, decoded, []string{"EndpointSpec", "Ports"}, expectedJSON)
}

func testFindNested(t *testing.T, json map[string]interface{}, keys []string, expected interface{}) {
	found, val := findNested(json, keys)
	if !found {
		t.Errorf("Key '%s' was not found \n", strings.Join(keys, "."))
	}
	if reflect.TypeOf(val) != reflect.TypeOf(expected) {
		t.Errorf("Types of val and expected are different %s != %s", reflect.TypeOf(val), reflect.TypeOf(expected))
	}
	if val != expected {
		t.Errorf("Value was incorrect, got '%s', want '%s'", prettyPrint(val), prettyPrint(expected))
	}
}

func TestIsAllowed(t *testing.T) {
	value := []byte(`{"Source": "/mnt/scratch/", "Target": 10, "ReadOnly": true, "Labels": {"com.example.something": "something-value"}}`)
	allowed := []byte(`[{"Source": "^/mnt/scratch", "Target": 10, "ReadOnly": true}]`)

	var decodedValue interface{}
	err := json.Unmarshal(value, &decodedValue)
	if err != nil {
		panic(err)
	}

	var decodedAllowed []interface{}
	err = json.Unmarshal(allowed, &decodedAllowed)
	if err != nil {
		panic(err)
	}

	ok := isAllowed(decodedValue, decodedAllowed)
	if !ok {
		t.Errorf("Value %v not matching %v", decodedValue, decodedAllowed)
	}
}
