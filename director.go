package dockerguard

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"regexp"
	"strings"

	"github.com/micoud/dockerguard/config"
	"github.com/micoud/dockerguard/socketproxy"
)

var (
	versionRegex = regexp.MustCompile(`^/v\d\.\d+\b`)
)

// RulesDirector ... struct that contains a http client additional fields needed
// for handling / manipulating requests
type RulesDirector struct {
	Client        *http.Client
	RoutesAllowed *config.RoutesAllowed
	Debug         bool
}

func writeError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message": msg,
	})
}

// Direct ... fn to handle incoming requests, it forwards allowed requests to upstream
// or returns an error if the request is not allowed
func (r *RulesDirector) Direct(l socketproxy.Logger, req *http.Request, upstream http.Handler) http.Handler {
	var match = func(method string, pattern string) bool {
		if method != "*" && method != req.Method {
			return false
		}
		path := req.URL.Path
		if versionRegex.MatchString(path) {
			path = versionRegex.ReplaceAllString(path, "")
		}
		re := regexp.MustCompile(pattern)
		return re.MatchString(path)
	}

	var errorHandler = func(msg string, code int) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			l.Printf("Handler returned error %q", msg)
			writeError(w, msg, code)
			return
		})
	}

	// match default routes
	switch {
	case match(`GET`, `^/(_ping|version|info)$`):
		return upstream
	case match(`HEAD`, `^/_ping$`):
		return upstream
	}

	// match routes defined in json files
	for _, route := range r.RoutesAllowed.Routes {
		if match(route.Method, route.Pattern) {
			// do request checking
			if (route.Method == "POST" && req.Header.Get("Content-Type") == "application/json" && route.CheckJSON != nil) ||
				(route.CheckParam != nil) ||
				(route.AppendFilter != nil) {
				return r.checkRequest(l, req, upstream, route.CheckJSON, route.CheckParam, route.AppendFilter)
			}

			return upstream
		}
	}

	return errorHandler(req.Method+" "+req.URL.Path+" Endpoint not allowed", http.StatusForbidden)
}

func (r *RulesDirector) checkRequest(l socketproxy.Logger, req *http.Request, upstream http.Handler, checkJSON []config.CheckJSON, checkParam []config.CheckParam, appendFilter []config.AppendFilter) http.Handler {
	if r.Debug {
		fmt.Println("Called checkRequest()")
	}
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var q = req.URL.Query()
		// check URL params
		if checkParam != nil {
			for _, c := range checkParam {
				if qf := q.Get(c.Param); qf != "" {
					fmt.Printf("Param found %s\n", qf)
					if !isAllowed(qf, c.AllowedValues) {
						errString := fmt.Sprintf("Found forbidden value: %v for param %s", qf, c.Param)
						fmt.Println(errString)
						writeError(w, errString, http.StatusUnauthorized)
						return
					}
				}
			}
		}

		// append labels to filters
		if appendFilter != nil {
			var filters = map[string][]interface{}{}
			// parse existing filters from querystring
			if qf := q.Get("filters"); qf != "" {
				var existing map[string]interface{}

				if err := json.NewDecoder(strings.NewReader(qf)).Decode(&existing); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}

				// different docker implementations send us different data structures
				for k, v := range existing {
					switch tv := v.(type) {
					// sometimes we get a map of value=true
					case map[string]interface{}:
						for mk := range tv {
							filters[k] = append(filters[k], mk)
						}
					// sometimes we get a slice of values (from docker-compose)
					case []interface{}:
						filters[k] = append(filters[k], tv...)
					default:
						http.Error(w, fmt.Sprintf("Unhandled filter type of %T", v), http.StatusBadRequest)
						return
					}
				}
			}
			// add fields to filter
			for _, f := range appendFilter {
				if _, exists := filters[f.FilterKey]; !exists {
					filters[f.FilterKey] = []interface{}{}
				}

				// add values to filter param
				for _, v := range f.Values {
					l.Printf("Adding '%v' to filter '%v'", v, f.FilterKey)
					filters[f.FilterKey] = append(filters[f.FilterKey], v)
				}
			}

			// encode back into json
			encoded, err := json.Marshal(filters)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			q.Set("filters", string(encoded))
			req.URL.RawQuery = q.Encode()
		}

		// check JSON
		if checkJSON != nil {
			fmt.Println("checkRequest() - JSON checking")
			var decoded map[string]interface{}
			if err := json.NewDecoder(req.Body).Decode(&decoded); err != nil {
				writeError(w, err.Error(), http.StatusBadRequest)
				return
			}

			if r.Debug {
				fmt.Printf("%s \n", prettyPrint(decoded))
			}

			for _, c := range checkJSON {
				found, val := findNested(decoded, c.Key)
				if found {
					switch vt := val.(type) {
					// if val is an array
					case []interface{}:
						for _, v := range vt {
							if !isAllowed(v, c.AllowedValues) {
								errString := fmt.Sprintf("Found forbidden value: %v for key %s", v, c.Key)
								fmt.Println(errString)
								writeError(w, errString, http.StatusUnauthorized)
								return
							}
						}
					// if val is a single object
					case interface{}:
						if !isAllowed(val, c.AllowedValues) {
							errString := fmt.Sprintf("Found forbidden value: %v for key %s", val, c.Key)
							fmt.Println(errString)
							writeError(w, errString, http.StatusUnauthorized)
							return
						}
					}
				} else {
					// TODO: this should trigger notice, that routes*.json is not configured well
					fmt.Printf("Key '%s' not found\n", strings.Join(c.Key, "."))
				}
			}

			encoded, err := json.Marshal(decoded)
			if err != nil {
				writeError(w, err.Error(), http.StatusBadRequest)
				return
			}

			// reset it so that upstream can read it again
			req.ContentLength = int64(len(encoded))
			req.Body = ioutil.NopCloser(bytes.NewReader(encoded))
		}

		upstream.ServeHTTP(w, req)
	})
}

// aux function to pretty print json
func prettyPrint(i interface{}) string {
	s, _ := json.MarshalIndent(i, "", "\t")
	return string(s)
}

// aux function to find nested key 'key' in map
func findNested(m map[string]interface{}, keys []string) (bool, interface{}) {
	// fmt.Printf("keys: %s, len(keys) %d\n", keys, len(keys))
	// already reached the last level
	if keys != nil {
		for k, v := range m {
			if k == keys[0] && len(keys) == 1 {
				// fmt.Printf("%s %s %v\n", keys[0], k, v)
				return true, v
			}
		}

		for _, v := range m {
			nm, ok := v.(map[string]interface{})
			if ok && len(keys) > 1 {
				found, val := findNested(nm, keys[1:])
				if found {
					return found, val
				}
			}
		}
	}

	// not found at all
	return false, nil
}

// aux function to match allowed_values with values in json / param
func isAllowed(value interface{}, allowedValues []interface{}) bool {
	var matchString = func(v string, a string) bool {
		fmt.Printf("Check allowed string: '%s' against '%s'\n", v, a)
		re := regexp.MustCompile(a)
		return re.MatchString(v)
	}

	var matchFloat = func(v float64, a float64) bool {
		fmt.Printf("Check allowed number: '%f' against '%f'\n", v, a)
		return v == a
	}

	var matchBool = func(v bool, a bool) bool {
		fmt.Printf("Check allowed bool: '%t' against '%t'\n", v, a)
		return v == a
	}

	var matchJSON = func(v map[string]interface{}, a map[string]interface{}) bool {
		fmt.Printf("Check allowed JSON: '%v' against '%v'\n", v, a)
		for kv, vv := range v {
			for ka, va := range a {
				if kv == ka {
					// fmt.Printf("Check key:%s - val '%v' (%v) vs allowed '%v' (%v)\n", kv, vv, reflect.TypeOf(vv), va, reflect.TypeOf(va))
					if reflect.TypeOf(vv) == reflect.TypeOf(va) {
						switch vt := vv.(type) {
						case bool:
							if va, ok := va.(bool); ok {
								fmt.Printf("Check allowed bool: '%t' against '%t'\n", vt, va)
								if vt != va {
									return false
								}
							}
						case float64:
							if va, ok := va.(float64); ok {
								fmt.Printf("Check allowed number: '%f' against '%f'\n", vt, va)
								if vt != va {
									return false
								}
							}
						case string:
							if va, ok := va.(string); ok {
								fmt.Printf("Check allowed string: '%s' against '%s'\n", vt, va)
								re := regexp.MustCompile(va)
								if !re.MatchString(vt) {
									return false
								}
							}
						default:
							return false
						}
					} else {
						fmt.Printf("Types do not match! Not allowed.\n")
						return false
					}
				}
			}
		}
		return true
	}

	for _, a := range allowedValues {
		if reflect.TypeOf(value) == reflect.TypeOf(a) {
			switch vt := value.(type) {
			case bool:
				if a, ok := a.(bool); ok {
					if matchBool(vt, a) {
						return true
					}
				}
			case float64:
				if a, ok := a.(float64); ok {
					if matchFloat(vt, a) {
						return true
					}
				}
			case string:
				if a, ok := a.(string); ok {
					if matchString(vt, a) {
						return true
					}
				}
			case map[string]interface{}:
				if a, ok := a.(map[string]interface{}); ok {
					if matchJSON(vt, a) {
						return true
					}
				}
			}
		} else {
			return false
		}
	}
	return false
}
