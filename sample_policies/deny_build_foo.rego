package docker.build

allow {
   not hasFooImage
}

hasFooImage {
	images[_] = "foo"
} {
	true
}

## Get all FROM lines ##
images[output] {
  line := input.Dockerfile[_]
  startswith(line, "FROM ")
  output = substring(line, 5, -1)
}