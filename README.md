# Dockerguard

Sets up a custom proxy to guard a docker socket to filter security relevant calls.

## Credits

This is very much inspired by the excellent [sockguard](https://github.com/buildkite/sockguard) project, where the basic structure and the proxy functionality comes from.

## Build and run

```bash
go build -o dockerguard ./cmd/dockerguard
./dockerguard [-debug] [-port <port number>] [-upstream <docker-socket>] [-config </path/to/routes.json>]
```

### Commandline flags

* `-debug`: get detailed logging of request and response bodies, should only be used for debugging, default is `false`
* `-port`: local port number that is listened on , default is `2375`
* `-upstream`: docker-socket to guard/to forward allowed requests to, default is `/var/run/docker.sock`
* `-config`: specifies the file to read routes config from, default is `routes.json`


## Docker container

Build the container

```bash
docker build -t dockerguard:latest .
```

Note: to use it the docker-socket has to be mounted. To override the standard (dummy) routes configuration mount your desired configuration to /routes.json or mount it to another location and override the command by appending `/go/bin/dockerguard <config-params>`. Port and upstream docker-socket can be overriden, likewise.

E.g. to use `examples/routes_create_container.json` and enable debugging, run

```bash
docker run -v /var/run/docker.sock:/var/run/docker.sock -v /path/to/examples/routes_create_container.json:/routes.json -p 2375:2375 dockerguard /go/bin/dockerguard -debug=true
```

### Example docker-compose-file

Find an example for a docker stack to deploy to a swarm with a dockerguard container and clients with an encrypted network in `docker-compose.yml`.

## Use as proxy for Docker daemon

To use it set env. variable `export DOCKER_HOST='DOCKER_HOST=tcp://localhost:<port>'`.



## Configuration of allowed routes

Routes that should be allowed are defined in json files with the following structure:

```json
{
  "routes_allowed": [
    {
      "method": "GET",
      "pattern": "^/containers/json$"
    }
  ]
}
```

where method can be `GET`, `POST` or both matched by `*`. The pattern is a [golang regular expression](https://golang.org/pkg/regexp/syntax/) pattern.

Regular expressions can be used, e.g., to allow only container names that match 'mariadb'.

```json
{
  "method": "*",
  "pattern": "^/containers/(.*mariadb.*)/(json|start|stop)$"
}
```

If no config-file is specified `routes.json` is used, that just enables a listing of running containers via `docker ps`.

Find example route definitions in `./examples`.


Note: to learn about Docker API endpoints, consult the [documentation](https://docs.docker.com/engine/api/v1.40/).

## TODOs

* [x] dockerize
* [x] Jenkinsfile
* [x] mechanism to check posted JSONs
* [x] mechanism to check URL parameters
* [ ] finish implementation of label adding
* [ ] review which HTTP statuscodes should be used where
* [ ] add mechanism to manipulate jsons in request bodies (e.g. for services)
* [ ] add tests
