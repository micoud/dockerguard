package dockerguard

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"

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
			if route.Method == "POST" && req.Header.Get("Content-Type") == "application/json" {
				return r.handleJSON(l, req, upstream)
			}
			return upstream
		}
	}

	return errorHandler(req.Method+" "+req.URL.Path+" Endpoint not allowed", http.StatusForbidden)
}

func (r *RulesDirector) handleJSON(l socketproxy.Logger, req *http.Request, upstream http.Handler) http.Handler {
	if r.Debug {
		fmt.Println("Called handleJSON()")
	}
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var decoded map[string]interface{}
		if err := json.NewDecoder(req.Body).Decode(&decoded); err != nil {
			writeError(w, err.Error(), http.StatusBadRequest)
			return
		}

		if r.Debug {
			fmt.Printf("%s \n", prettyPrint(decoded))
		}

		encoded, err := json.Marshal(decoded)
		if err != nil {
			writeError(w, err.Error(), http.StatusBadRequest)
			return
		}

		// reset it so that upstream can read it again
		req.ContentLength = int64(len(encoded))
		req.Body = ioutil.NopCloser(bytes.NewReader(encoded))

		upstream.ServeHTTP(w, req)
	})
}

// aux function to pretty print json
func prettyPrint(i interface{}) string {
	s, _ := json.MarshalIndent(i, "", "\t")
	return string(s)
}
