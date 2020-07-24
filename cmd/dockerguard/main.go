package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/micoud/dockerguard"
	"github.com/micoud/dockerguard/config"
	"github.com/micoud/dockerguard/socketproxy"
)

var (
	debug bool
)

func main() {
	// read cmdline flags
	flag.BoolVar(&debug, "debug", false, "Show debugging logging for the socket")
	configfile := flag.String("config", "routes.json", "json-file to read routes config from")
	upstream := flag.String("upstream", "/var/run/docker.sock", "The path to docker socket")
	port := flag.Int("port", 2375, "port to listen on")
	flag.Parse()

	if debug {
		socketproxy.Debug = true
	}

	// read the routes config from file
	routesAllowed := config.RoutesConfig(*configfile)
	for _, r := range routesAllowed.Routes {
		fmt.Printf("Route allowed: %s, %s \n", r.Method, r.Pattern)
		if r.CheckJSON != nil {
			fmt.Printf("\t JSON key to check: %v\n", r.CheckJSON)
		}
		if r.CheckParam != nil {
			fmt.Printf("\t URL Param to check: %v\n", r.CheckParam)
		}
		if r.AppendFilter != nil {
			fmt.Printf("\t Filters to append: %v\n", r.AppendFilter)
		}
	}

	// dial upstreamproxy
	proxyHTTPClient := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				debugf("Dialing directly")
				return net.Dial("unix", *upstream)
			},
		},
	}

	proxy := socketproxy.New(*upstream, &dockerguard.RulesDirector{
		Client:        &proxyHTTPClient,
		RoutesAllowed: &routesAllowed,
		Debug:         debug,
	})

	listener, err := net.Listen("tcp", ":"+strconv.Itoa(*port))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Listening on port " + strconv.Itoa(*port) + "...\n")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, os.Kill, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		debugf("Caught signal %s: shutting down.", sig)
		_ = listener.Close()
		os.Exit(0)
	}()

	if err = http.Serve(listener, proxy); err != nil {
		log.Fatal(err)
	}
}

func debugf(format string, v ...interface{}) {
	if debug {
		fmt.Printf(format+"\n", v...)
	}
}
