package docker.authz

allow {
  not input.Method = "POST"
}