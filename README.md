# Docker Socket Firewall

The docker socket firewall provides a way to protect the docker socket using OPA rego policies. For more information  on the rego format see: [Open Policy Agent](https://www.openpolicyagent.org/)

When running the firewall opens a unix socket that proxies all requests through to the real docker socket, you can defined policies that lock down:
- The source of docker images
- Running of containers in privileged mode
- Defining a whitelist of capabilities that containers can use

These are just some examples, the policies should allow you to lock down any API call through to docker and anything found in a dockerfile being built.

## Todo
Things I should add to this:
- Bring basic URL parsing and some basic dockerfile variables into golang to avoid complicating the policies

## Usage

Usage of docker-socket-firewall:

Argument | Type | Description
---------|------|-------------
host|string|The docker socket to listen on (default ```/var/run/protected-docker.sock```)
policyDir|string|The directory containing the OPA policies (default ```/etc/docker```)
target|string|The docker socket to connect to (default ```/var/run/docker.sock```)
usage| |Print usage information
verbose| |Print debug logging

With docker running I can execute with verbose mode and see how the firewall works

```sudo ./docker-socket-firewall --verbose```

```INFO[0000] Docker Firewall: /var/run/docker.sock -> /var/run/protected-docker.sock, Policy Dir: /etc/docker```

Now I know that the firewall is up and running on ```/var/run/protected-docker.sock```

We need to configure docker to point to this socket
```export DOCKER_HOST=unix:///var/run/protected-docker.sock```

And try out a basic command ```docker ps```

```
DEBU[0170] Received Request: /v1.39/containers/json     
DEBU[0170] Querying OPA policy data.docker.authz.allow. Input: {
  "Body": null,
  "Headers": {
    "User-Agent": "Docker-Client/18.09.1 (darwin)"
  },
  "Method": "GET",
  "Path": "/v1.39/containers/json?"
} 
DEBU[0170] Returning OPA policy decision: true 
```

We can see three log lines printed, each with important information on it
- The request path that we received
- The OPA policy path ```data.docker.authz.allow``` (from authz.rego) being invoked and the input being passed to it
- Finally the policy decision, in this case we accepted the request and returned true

When building a container the usage is similar:

In this example we want to build a simple docker file

```dockerfile
FROM ubuntu:18.04
COPY . /app
```

```
DEBU[0675] Received Request: /v1.39/build               
DEBU[0675] Querying OPA policy data.docker.build.allow. Input: {
  "Dockerfile": [
    "FROM ubuntu:18.04",
    "COPY . /app",
    ""
  ],
  "Headers": {
    "Content-Type": "application/x-tar",
    "User-Agent": "Docker-Client/18.09.1 (darwin)",
    "X-Registry-Config": "bnVsbA=="
  },
  "Method": "POST",
  "Path": "/v1.39/build?buildargs=%7B%7D\u0026cachefrom=%5B%5D\u0026cgroupparent=\u0026cpuperiod=0\u0026cpuquota=0\u0026cpusetcpus=\u0026cpusetmems=\u0026cpushares=0\u0026dockerfile=Dockerfile\u0026labels=%7B%7D\u0026memory=0\u0026memswap=0\u0026networkmode=default\u0026rm=1\u0026shmsize=0\u0026target=\u0026ulimits=null\u0026version=1"
} 
DEBU[0675] Returning OPA policy decision: true 
```

The notable differences in building and other commands are:
- Different policy path and file ```data.docker.build.allow``` (from build.rego) 
- The input contains the dockerfile steps in a list

## Policy examples

### Build Policies - build.rego

Disable dockerfiles containing the line "FROM node"

```
package docker.build
  
allow {
  not nodeImage
}

nodeImage {
  images[_] = "node"
} {
  true
}

## Get all FROM lines ##
images[output] {
  line := input.Dockerfile[_]
  startswith(line, "FROM")
  output = substring(line, 5, -1)
}
```

### Run policies - authz.rego

Stop creation of docker networks

```
package docker.authz

allow {
  not createNetwork
}

createNetwork {
  startswith(path, "/networks/create")
}

## Parsing Path, Allowing for versioned and non versioned
versioned = output {
  output = re_match("/v\\d+.*", input.Path)
}

path = output {
  not versioned
  index := indexof(input.Path, "?")
  output = substring(input.Path, 0, index)
}

path = output {
  versioned
  path := substring(input.Path, 2, -1)
  end := indexof(path, "?")
  start := indexof(path, "/")
  output = substring(path, start, end)
}
```

## Building

You can build the docker-socket-firewall executable with make, type 

- ```make local``` - builds the local executable for your environment

- ```make ci``` - builds both mac and linux executables