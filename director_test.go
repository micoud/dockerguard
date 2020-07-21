package dockerguard

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
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

	// different 'Labels'
	testFindNested(t, decoded, []string{"Labels", "foo"}, "bar")
	testFindNested(t, decoded, []string{"TaskTemplate", "ContainerSpec", "Mounts", "VolumenOptions", "Labels", "com.example.something"}, "bar")
}

func testFindNested(t *testing.T, json map[string]interface{}, keys []string, expected string) {
	found, val := findNested(json, keys)
	if !found {
		t.Errorf("Key '%s' was not found \n", strings.Join(keys, "."))
	}
	if val != expected {
		t.Errorf("Value was incorrect, got %s, want %s", val, expected)
	}
}
