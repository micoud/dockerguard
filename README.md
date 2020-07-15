# Dockerguard

Sets up a custom proxy to guard a docker socket to filter security relevant calls.

## Credits

This is very much inspired by the excellent [sockguard](https://github.com/buildkite/sockguard) project, where the basic structure and the proxy functionality comes from.

## Build and run

```bash
go build -o dockerguard ./cmd/dockerguard
./dockerguard [-debug] [-port <port number>] [-upstream <docker-socket>]
```
