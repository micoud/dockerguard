package dockerguard

import (
	"encoding/json"
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

	for _, route := range r.RoutesAllowed.Routes {
		if match(route.Method, route.Pattern) {
			return upstream
		}
	}
	// case match(`GET`, `^/events$`):
	// 	return r.addLabelsToQueryStringFilters(l, req, upstream)
	// // Container related endpoints
	// case match(`POST`, `^/containers/create$`):
	// 	return r.handleContainerCreate(l, req, upstream)
	// case match(`POST`, `^/containers/prune$`):
	// 	return r.addLabelsToQueryStringFilters(l, req, upstream)
	// case match(`GET`, `^/containers/json$`):
	// 	return r.addLabelsToQueryStringFilters(l, req, upstream)
	// case match(`*`, `^/(containers|exec)/(\w+)\b`):
	// 	if ok, err := r.checkOwner(l, "containers", false, req); ok {
	// 		return upstream
	// 	} else if err == errInspectNotFound {
	// 		l.Printf("Container not found, allowing")
	// 		return upstream
	// 	} else if err != nil {
	// 		return errorHandler(err.Error(), http.StatusInternalServerError)
	// 	}
	// 	return errorHandler("Unauthorized access to container", http.StatusUnauthorized)

	// // Build related endpoints
	// case match(`POST`, `^/build$`):
	// 	return r.handleBuild(l, req, upstream)

	// // Image related endpoints
	// case match(`GET`, `^/images/json$`):
	// 	return r.addLabelsToQueryStringFilters(l, req, upstream)
	// case match(`POST`, `^/images/create$`):
	// 	return upstream
	// case match(`POST`, `^/images/(create|search|get|load)$`):
	// 	break
	// case match(`POST`, `^/images/prune$`):
	// 	return r.addLabelsToQueryStringFilters(l, req, upstream)
	// case match(`*`, `^/images/(\w+)\b`):
	// 	if ok, err := r.checkOwner(l, "images", true, req); ok {
	// 		return upstream
	// 	} else if err == errInspectNotFound {
	// 		l.Printf("Image not found, allowing")
	// 		return upstream
	// 	} else if err != nil {
	// 		return errorHandler(err.Error(), http.StatusInternalServerError)
	// 	}
	// 	return errorHandler("Unauthorized access to image", http.StatusUnauthorized)

	// // Network related endpoints
	// case match(`GET`, `^/networks$`):
	// 	return r.addLabelsToQueryStringFilters(l, req, upstream)
	// case match(`POST`, `^/networks/create$`):
	// 	return r.handleNetworkCreate(l, req, upstream)
	// case match(`POST`, `^/networks/prune$`):
	// 	return r.addLabelsToQueryStringFilters(l, req, upstream)
	// case match(`DELETE`, `^/networks/(.+)$`):
	// 	return r.handleNetworkDelete(l, req, upstream)
	// case match(`GET`, `^/networks/(.+)$`),
	// 	match(`POST`, `^/networks/(.+)/(connect|disconnect)$`):
	// 	if ok, err := r.checkOwner(l, "networks", true, req); ok {
	// 		return upstream
	// 	} else if err == errInspectNotFound {
	// 		l.Printf("Network not found, allowing")
	// 		return upstream
	// 	} else if err != nil {
	// 		return errorHandler(err.Error(), http.StatusInternalServerError)
	// 	}
	// 	return errorHandler("Unauthorized access to network", http.StatusUnauthorized)

	// // Volumes related endpoints
	// case match(`GET`, `^/volumes$`):
	// 	return r.addLabelsToQueryStringFilters(l, req, upstream)
	// case match(`POST`, `^/volumes/create$`):
	// 	return r.addLabelsToBody(l, req, upstream)
	// case match(`POST`, `^/volumes/prune$`):
	// 	return r.addLabelsToQueryStringFilters(l, req, upstream)
	// case match(`GET`, `^/volumes/([-\w]+)$`), match(`DELETE`, `^/volumes/(-\w+)$`):
	// 	if ok, err := r.checkOwner(l, "volumes", true, req); ok {
	// 		return upstream
	// 	} else if err == errInspectNotFound {
	// 		l.Printf("Volume not found, allowing")
	// 		return upstream
	// 	} else if err != nil {
	// 		return errorHandler(err.Error(), http.StatusInternalServerError)
	// 	}
	// 	return errorHandler("Unauthorized access to volume", http.StatusUnauthorized)

	return errorHandler(req.Method+" "+req.URL.Path+" Endpoint not allowed", http.StatusForbidden)
}
