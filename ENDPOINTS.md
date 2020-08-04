# DOCKER API ENDPOINTS

Here the most important docker calls for the usage in a CI setup (e.g. from a Jenkins swarm-agent) are listed.

## Containers

* **list containers** `docker ps`
  * `GET - /v1.40/container/json`
  * add filter to e.g. only list containers that are attached to a certain network
* **inspect a container** `docker inspect <id|name>`
  * `GET - /v1.40/containers/<id|name>/json`
* **exec a command** inside a container `docker exec [flags] <container-name|id> <cmd>`
  * `GET - /v1.40/containers/<container-name>/json`
  * `POST - /v1.40/containers/<container-name>/exec`
  * `POST - /v1.40/exec/<id>/start` the id is a random exec-id instance
  * when called as a TTY session (*flag -t*) `POST - /v1.40/exec/<id>/resize?h=50&w=190` is called with terminal parameters
  * `GET - /v1.40/exec/<id>/json` the same id used


## Images

* **build an image** `docker build -t <image:tag> .`
  * e.g. `POST - /v1.40/build?buildargs=%7B%7D&cachefrom=%5B%5D&cgroupparent=&cpuperiod=0&cpuquota=0&cpusetcpus=&cpusetmems=&cpushares=0&dockerfile=Dockerfile&labels=%7B%7D&memory=0&memswap=0&networkmode=default&rm=1&shmsize=0&t=<image:tag>&target=&ulimits=null&version=1`
* **push an image** to the registry `docker push <image:tag>`
  * `POST - /v1.40/images/<image>/push?tag=<tag>`


## Services

* **list swarm services** `docker service ls`
  * `GET - /v1.40/services` (add filter to restrict which services are visible)
  * `GET - /v1.40/tasks?filters=%7B%22service%22%3A%7B%22<id>%22%3Atrue%2C%22<id2>%22%3Atrue%7D%7D` (the filters contain the ids that the former api call has returned)
  * `GET - /v1.40/nodes`
* **start a stack** defined in a compose file `docker stack deploy -c <stack.yml> <stack>`
  * `GET - /v1.40/networks/<network name requested>` (the network requested could be the one to which CI containers are detached)
  * `GET - /v1.40/networks?filters=%7B%22label%22%3A%7B%22com.docker.stack.namespace%3D<stack-name>%22%3Atrue%7D%7D` (the filter com.docker.stack.namespace=stack-name is added by the client) - an additional filter for network names might be added
  * `GET - /v1.40/services?filters=%7B%22label%22%3A%7B%22com.docker.stack.namespace%3D<stack-name>%22%3Atrue%7D%7D` - additional filters might be added
  * `GET - /v1.40/distribution/<registry-name>/<image-name>/json` - registry-name and image-name should be defined by regular expressions to match allowed images
  * `POST - /v1.40/services/create` criterias should be defined for the posted JSON, e.g.
    * name of services
    * network to connect to
    * allowed host-directories to be mounted
* **list processes/tasks that belong to a service** `docker service ps <service_name>`
  * `GET - /v1.40/services?filters=%7B%22id%22%3A%7B%22<service-name>%22%3Atrue%7D%7D` (additional filters might be configured, see above)
  * `GET - /v1.40/services?filters=%7B%22name%22%3A%7B%22<service-name>%22%3Atrue%7D%7D` same call again but this time with name-filter instead of id-filter
  * `GET - /v1.40/tasks?filters=%7B%22service%22%3A%7B%22<service-id>%22%3Atrue%7D%7D` id is the one that was returned by the last request
  * `GET - /v1.40/services/<service-id>?insertDefaults=false` - the id is the one that was returned by the last request
  * `GET - /v1.40/nodes/<node-id>`
* **stop a stack** `docker stack rm <stack-name>`
  * `GET - /v1.40/services?filters=%7B%22label%22%3A%7B%22com.docker.stack.namespace%3D<stack-name>%22%3Atrue%7D%7D`
  * `GET - /v1.40/networks?filters=%7B%22label%22%3A%7B%22com.docker.stack.namespace%3D<stack-name>%22%3Atrue%7D%7D`
  * `GET - /v1.40/secrets?filters=%7B%22label%22%3A%7B%22com.docker.stack.namespace%3D<stack-name>%22%3Atrue%7D%7D`
  * `GET - /v1.40/configs?filters=%7B%22label%22%3A%7B%22com.docker.stack.namespace%3D<stack-name>%22%3Atrue%7D%7D`
  * `DELETE - /v1.40/services/<service-id>` this endpoint is called for every service that should be stopped, the ids come from previous calls


## Exec command in service

* **execute a command in a service container**

```bash
> docker service ps --filter 'desired-state=running' $SERVICE_NAME -q                      # -> TASK_ID
> docker inspect --format '{{ .NodeID }}' $TASK_ID                                         # -> NODE_ID
> docker inspect --format '{{ .Status.ContainerStatus.ContainerID }}' $TASK_ID             # -> CONTAINER_ID
> docker node inspect --format '{{ .Description.Hostname }}' $NODE_ID | cut -d '.' -f 1    # -> NODE_HOST
```

calls made by this commands

* `GET - /v1.40/services?filters=%7B%22id%22%3A%7B%22<service_name>%22%3Atrue%7D%7D`
* `GET - /v1.40/services?filters=%7B%22name%22%3A%7B%22<service_name>%22%3Atrue%7D%7D`
* `GET - /v1.40/tasks?filters=%7B%22desired-state%22%3A%7B%22running%22%3Atrue%7D%2C%22service%22%3A%7B%22<service-id>%22%3Atrue%7D%7D` (task id returned from previous request)
* `GET - /v1.40/services/<task-id>?insertDefaults=false`
* `GET - /v1.40/nodes/<node-id>`
* `GET - /v1.40/containers/<task-id>/json`
* `GET - /v1.40/images/<task-id>/json`

Calls for subsequent `docker exec` calls are listed above.

## Notes

* filters in URLs are encoded as JSONs and processed as such by the proxy
* for some endpoints it can be useful to append filters and as well to check filters:
  * e.g. `GET /services` is used for different CLI actions and uses filters in different ways
