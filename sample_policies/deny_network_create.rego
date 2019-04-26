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