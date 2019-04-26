package docker.authz

allow {
   hasXfooHeader
}

hasXfooHeader {
	input.Headers["X-Foo"]
}