package opa

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/open-policy-agent/opa/rego"
)

type dockerOpaHandler struct {
	proxyPolicyFile      string
	dockerfilePolicyFile string
}

const authAllowPath string = "data.docker.authz.allow"
const buildAllowPath string = "data.docker.build.allow"

func NewDockerOpaHandler(pPolicy string, dPolicy string) dockerOpaHandler {
	return dockerOpaHandler{
		proxyPolicyFile:      pPolicy,
		dockerfilePolicyFile: dPolicy,
	}
}

func (p dockerOpaHandler) ProxyRequest(r *http.Request) (bool, error) {
	if _, err := os.Stat(p.proxyPolicyFile); os.IsNotExist(err) {
		log.Printf("OPA proxy policy file %s does not exist, failing open and allowing request", p.proxyPolicyFile)
		return true, err
	}

	input, err := makeInput(r)
	if err != nil {
		return false, err
	}

	allowed, err := processPolicy(input, p.proxyPolicyFile, authAllowPath, r.Context())

	return allowed, err
}

func (p dockerOpaHandler) ValidateDockerFile(r *http.Request, dockerFile string) (bool, error) {
	if _, err := os.Stat(p.dockerfilePolicyFile); os.IsNotExist(err) {
		log.Printf("OPA dockerfile policy file %s does not exist, failing open and allowing request", p.dockerfilePolicyFile)
		return true, err
	}

	input, err := makeDockerfileInput(r, strings.Split(dockerFile, "\n"))
	if err != nil {
		return false, err
	}

	allowed, err := processPolicy(input, p.dockerfilePolicyFile, buildAllowPath, r.Context())

	return allowed, err
}

func processPolicy(input interface{}, policyFile string, policyPath string, ctx context.Context) (bool, error) {

	bs, err := ioutil.ReadFile(policyFile)
	if err != nil {
		return false, err
	}

	pretty, _ := json.MarshalIndent(input, "", "  ")
	log.Printf("Querying OPA policy %v. Input: %s", policyPath, pretty)
	allowed, err := func() (bool, error) {

		eval := rego.New(
			rego.Query(policyPath),
			rego.Input(input),
			rego.Module(policyFile, string(bs)),
		)

		rs, err := eval.Eval(ctx)
		if err != nil {
			return false, err
		}

		if len(rs) == 0 {
			// Decision is undefined. Fallback to deny.
			return false, nil
		}

		allowed, ok := rs[0].Expressions[0].Value.(bool)
		if !ok {
			return false, fmt.Errorf("administrative policy decision invalid")
		}

		return allowed, nil

	}()
	if err != nil {
		log.Printf("Returning OPA policy decision: %v (error: %v)", allowed, err)
		return allowed, err
	} else {
		log.Printf("Returning OPA policy decision: %v", allowed)
	}
	return allowed, nil
}

func makeInput(r *http.Request) (interface{}, error) {

	var body interface{}

	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") && r.ContentLength > 0 {
		if err := json.NewDecoder(peekBody(r)).Decode(&body); err != nil {
			return nil, err
		}
	}

	input := map[string]interface{}{
		"Headers": flattenHeaders(r.Header),
		"Path":    r.URL.Path + "?" + r.URL.RawQuery,
		"Method":  r.Method,
		"Body":    body,
	}

	return input, nil
}

func makeDockerfileInput(r *http.Request, dockerfile []string) (interface{}, error) {

	var body interface{}

	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") && r.ContentLength > 0 {
		if err := json.NewDecoder(peekBody(r)).Decode(&body); err != nil {
			return nil, err
		}
	}

	input := map[string]interface{}{
		"Headers": 		flattenHeaders(r.Header),
		"Path":    		r.URL.Path + "?" + r.URL.RawQuery,
		"Method":  		r.Method,
		"Dockerfile": 	dockerfile,
	}

	return input, nil
}

func flattenHeaders(src http.Header) map[string]string {
	var headers = make(map[string]string)
	for k, vv := range src {
		headers[k] = strings.Join(vv, ", ")
	}
	return headers
}


func peekBody(req *http.Request) io.Reader {
	var buf bytes.Buffer
	b := req.Body
	var err error

	if _, err = buf.ReadFrom(b); err != nil {
		//TODO: Error handling
	}

	if err = b.Close(); err != nil {
		//TODO: Error handling
	}

	b1, b2 := ioutil.NopCloser(&buf), ioutil.NopCloser(bytes.NewReader(buf.Bytes()))

	req.Body = b2

	return b1

}