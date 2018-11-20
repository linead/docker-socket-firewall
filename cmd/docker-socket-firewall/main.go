package main

import (
	"archive/tar"
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/linead/docker-socket-firewall/pkg/opa"
	"github.com/pkg/errors"
	"github.com/tv42/httpunix"
)

/*
	Utilities
*/

// Get env var or default
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

/*
	Getters
*/

// Get the port to listen on
func getListenAddress() string {
	socketAddr := getEnv("SOCKET_ADDR", "/tmp/docker")
	return socketAddr
}

/*
	Logging
*/

func logSetup() {
	log.Printf("Server will run on socket: %s\n", getListenAddress())
}

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
	u.RegisterLocation("docker-socket", "/var/run/docker.sock")

	req.URL.Scheme = "http+unix"
	req.URL.Host = "docker-socket"

	resp, err := u.RoundTrip(req)

	if err != nil {
		//http.Error(w, err.Error(), http.StatusServiceUnavailable)
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
			log.Printf("read error during body copy: %v", rerr)
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

func verifyProxyCall(req *http.Request) {
	p := opa.NewDockerOpaHandler("/tmp/auth.rego", "/tmp/dockerfile.rego")
	p.ProxyRequest(req)

}

func verifyBuildInstruction(req *http.Request) {
	//preserve original request if we want to still send it (Dockerfile is clean)
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

	tr := tar.NewReader(b1)

	dockerfileLoc := req.URL.Query()["dockerfile"]

	var valid = true

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			log.Fatal(err)
		}
		if hdr.Name == dockerfileLoc[0] {
			p := opa.NewDockerOpaHandler("/tmp/auth.rego", "/tmp/dockerfile.rego")
			df, _ := ioutil.ReadAll(tr)
			p.ValidateDockerFile(req, string(df))
		}
	}

	if valid {
		req.Body = b2
	}

}

// Given a request send it to the appropriate url
func handleRequestAndRedirect(res http.ResponseWriter, req *http.Request) {
	// detect build requests
	matched, err := regexp.MatchString("/v\\d\\.\\d+/build", req.URL.Path)

	if err != nil {
		panic(err)
	} else if matched {
		verifyBuildInstruction(req)
	} else {
		verifyProxyCall(req)
	}

	serveReverseProxy(res, req)
}

// ListenAndServe Listen for all requests on the given socket
func ListenAndServe(sockPath string) error {
	http.HandleFunc("/", handleRequestAndRedirect)
	l, err := net.Listen("unix", "/tmp/docker")
	if err != nil {
		return errors.Wrap(err, "failed to listen")
	}
	return http.Serve(l, nil)
}

/*
	Entry
*/

func main() {
	// Log setup values
	logSetup()

	// clean up old sockets
	os.Remove("/tmp/docker")

	// start server
	if err := ListenAndServe(getListenAddress()); err != nil {
		panic(err)
	}
}
