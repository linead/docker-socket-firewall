package opa

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"path/filepath"
	"runtime"
	"testing"
)

var (
	_, b, _, _ = runtime.Caller(0)
	basepath   = filepath.Dir(b)
)

func TestCreateNetworkDenied(t *testing.T) {

	req, _ := http.NewRequest("GET", "/v1.39/networks/create?", nil)

	opaHandler := &DockerOpaHandler{ basepath+"/../../sample_policies/deny_network_create.rego", "build.rego"}
	res, _ := opaHandler.ValidateRequest(req);

	assert.False(t, res, "Network creation should be rejected");

}

func TestDenyPost(t *testing.T) {

	postReq, _ := http.NewRequest("POST", "/test", nil)

	opaHandler := &DockerOpaHandler{ basepath+"/../../sample_policies/deny_post.rego", "build.rego"}
	res, _ := opaHandler.ValidateRequest(postReq);

	assert.False(t, res, "POST request should be rejected");

	getReq, _ := http.NewRequest("GET", "/test", nil)

	res, _ = opaHandler.ValidateRequest(getReq);

	assert.True(t, res, "GET request should be allowed");

}

func TestDenyBasedOnHeader(t *testing.T) {
	reqWithHeader, _ := http.NewRequest("POST", "/test", nil)

	reqWithHeader.Header.Set("X-foo", "bar")

	opaHandler := &DockerOpaHandler{ basepath+"/../../sample_policies/deny_header.rego", "build.rego"}
	res, _ := opaHandler.ValidateRequest(reqWithHeader);

	assert.True(t, res, "request with X-foo header should be allowed");

	reqWithoutHeader, _ := http.NewRequest("POST", "/test", nil)

	res, _ = opaHandler.ValidateRequest(reqWithoutHeader);

	assert.False(t, res, "request without X-foo header should be rejected");

}

func TestBuildDockerfileFromFoo(t *testing.T) {

	req, _ := http.NewRequest("GET", "/test", nil)

	opaHandler := &DockerOpaHandler{ "", basepath+"/../../sample_policies/deny_build_foo.rego"}
	res, _ := opaHandler.ValidateDockerFile(req, "FROM foo");

	assert.False(t, res, "FROM foo should be rejected");

	res, _ = opaHandler.ValidateDockerFile(req, "FROM bar");

	assert.False(t, res, "FROM bar should be allowed");

}


