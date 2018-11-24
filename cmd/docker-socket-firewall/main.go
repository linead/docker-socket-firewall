package main

import (
	"archive/tar"
	"bytes"
	"context"
	"flag"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/linead/docker-socket-firewall/pkg/opa"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/tv42/httpunix"
)

var opaHandler *opa.DockerOpaHandler
var targetSocket string

/*
	Reverse Proxy Logic
*/

// Serve a reverse proxy for a given url
func serveReverseProxy(w http.ResponseWriter, req *http.Request) {
	u := &httpunix.Transport{
		DialTimeout:           100 * time.Millisecond,
		RequestTimeout:        10 * time.Second,
		ResponseHeaderTimeout: 0 * time.Second,
	}
	u.RegisterLocation("docker-socket", targetSocket)

	req.URL.Scheme = "http+unix"
	req.URL.Host = "docker-socket"

	resp, err := u.RoundTrip(req)

	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	defer resp.Body.Close()
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	copyBuffer(w, resp.Body)
}

func copyBuffer(dst io.Writer, src io.Reader) (int64, error) {
	var buf = make([]byte, 32*1024)
	var written int64
	for {
		nr, rerr := src.Read(buf)
		if rerr != nil && rerr != io.EOF && rerr != context.Canceled {
			log.Warnf("read error during body copy: %v", rerr)
		}
		if nr > 0 {
			nw, werr := dst.Write(buf[:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if werr != nil {
				return written, werr
			}
			if nr != nw {
				return written, io.ErrShortWrite
			}
		}
		if rerr != nil {
			if rerr == io.EOF {
				rerr = nil
			}
			return written, rerr
		}
	}
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func verifyBuildInstruction(req *http.Request) (bool, error) {
	//preserve original request if we want to still send it (Dockerfile is clean)
	var buf bytes.Buffer
	b := req.Body
	var err error

	if _, err = buf.ReadFrom(b); err != nil {
		return false, err
	}

	if err = b.Close(); err != nil {
		return false, err
	}

	b1, b2 := ioutil.NopCloser(&buf), ioutil.NopCloser(bytes.NewReader(buf.Bytes()))

	tr := tar.NewReader(b1)

	dockerfileLoc := req.URL.Query()["dockerfile"]

	var valid = false

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			log.Fatal(err)
		}
		if hdr.Name == dockerfileLoc[0] {
			df, _ := ioutil.ReadAll(tr)
			valid, _ = opaHandler.ValidateDockerFile(req, string(df))
		}
	}

	if valid {
		req.Body = b2
	}

	return valid, nil

}

// Given a request send it to the appropriate url if it validates
func handleRequestAndRedirect(res http.ResponseWriter, req *http.Request) {
	matched, _ := regexp.MatchString("^(/v[\\d\\.]+)?/build$", req.URL.Path)

	var err error
	var allowed bool

	if matched {
		allowed, err = verifyBuildInstruction(req)
	} else {
		allowed, err = opaHandler.ValidateRequest(req)
	}

	if err != nil {
		http.Error(res, "Authorization failure", http.StatusInternalServerError)
		return
	}

	if allowed {
		serveReverseProxy(res, req)
	} else {
		http.Error(res, "Authorization denied", http.StatusForbidden)
	}
}

func listenAndServe(sockPath string) error {
	http.HandleFunc("/", handleRequestAndRedirect)
	l, err := net.Listen("unix", sockPath)
	if err != nil {
		return errors.Wrap(err, "failed to listen")
	}

	os.Chmod(sockPath, 0777)

	return http.Serve(l, nil)
}

/*
	Entry
*/

func main() {

	targetSocket = *flag.String("target", "/var/run/docker.sock", "The docker socket to connect to")
	hostSocket := *flag.String("host", "/var/run/protected-docker.sock", "The docker socket to listen on")
	policyDir := *flag.String("policyDir", "/etc/docker", "The directory containing the OPA policies")
	printUsage := flag.Bool("usage", false, "Print usage information")
	verbose := flag.Bool("verbose", false, "Print debug logging")

	flag.Parse()

	if *printUsage {
		flag.Usage()
		os.Exit(0)
	}

	if *verbose {
		log.SetLevel(log.DebugLevel)
	}

	// clean up old sockets
	os.Remove(hostSocket)

	opaHandler = &opa.DockerOpaHandler{
		policyDir + "/authz.rego",
		policyDir + "/build.rego"}

	log.Infof("Docker Firewall: %s -> %s, Policy Dir: %s", targetSocket, hostSocket, policyDir)

	// start server
	if err := listenAndServe(hostSocket); err != nil {
		log.Fatal("Unable to start firewalled socket")
	}
}
