package main

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"context"
	"flag"
	"github.com/docker/go-connections/sockets"
	"github.com/h2non/filetype"
	"github.com/linead/docker-socket-firewall/pkg/opa"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/xi2/xz"
	"golang.org/x/net/context/ctxhttp"
	"gopkg.in/h2non/filetype.v1/matchers"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"regexp"
	"time"
)

var opaHandler opa.DockerHandler
var targetSocket string

/*
	Reverse Proxy Logic
*/

// Serve a reverse proxy for a given url
func serveReverseProxy(w http.ResponseWriter, req *http.Request) {
	transport := new(http.Transport)
	sockets.ConfigureTransport(transport, "unix", targetSocket)
	client := &http.Client{
		Transport: transport,
	}

	req.Proto = "http"
	req.URL.Scheme = "http"
	req.URL.Host = targetSocket
	req.RequestURI = ""
	req.Close = true

	if req.Header.Get("Connection") == "Upgrade" {
		if req.Header.Get("Upgrade") != "tcp" && req.Header.Get("Upgrade") != "h2c" {
			http.Error(w, "Unsupported upgrade protocol: "+req.Header.Get("Protocol"), http.StatusInternalServerError)
			return
		}
		log.Debug("Connection upgrading")
		hijack(req, w)
	} else {
		resp, err := ctxhttp.Do(req.Context(), client, req)

		if err != nil {
			log.Warnf("Error %v", err)
			return
		}

		defer resp.Body.Close()

		copyHeader(w.Header(), resp.Header)

		//If we're looking at a raw stream and we're not sending a value fo TE golang tries
		//to chunk the response, which can break clients.
		if resp.Header.Get("Content-Type") == "application/vnd.docker.raw-stream" {
			if resp.Header.Get("Transfer-Encoding") == "" {
				w.Header().Set("Transfer-Encoding", "identity")
			}
		}
		w.WriteHeader(resp.StatusCode)

		flushResponse(w)
		copyBuffer(w, resp.Body)
	}
}

func hijack(req *http.Request, w http.ResponseWriter) {
	inConn, err := net.Dial("unix", targetSocket)

	if err != nil {
		log.Warnf("Error in connection %v", err)
	}

	if tcpConn, ok := inConn.(*net.TCPConn); ok {
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(30 * time.Second)
	}

	clientconn := httputil.NewClientConn(inConn, nil)

	// Server hijacks the connection, error 'connection closed' expected
	resp, err := clientconn.Do(req)
	if err != httputil.ErrPersistEOF {
		if err != nil {
			log.Errorf("error upgrading: %v", err)
		}
		if resp.StatusCode != http.StatusSwitchingProtocols {
			resp.Body.Close()
			log.Errorf("unable to upgrade to %s, received %d", "tcp", resp.StatusCode)
		}
	}

	log.Debugf("Response: %v", resp)
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)

	flushResponse(w)

	c, br := clientconn.Hijack()

	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "webserver doesn't support hijacking", http.StatusInternalServerError)
		return
	}
	outConn, _, err := hj.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if br.Buffered() > 0 {
		log.Debugf("Found buffered bytes")
		var bs = make([]byte, br.Buffered())
		br.Read(bs)
		outConn.Write(bs)
	}

	errClient := make(chan error, 1)
	errBackend := make(chan error, 1)

	streamFn := func(dst, src net.Conn, errc chan error, desc string) {
		log.Debugf("%s Streaming connections", desc)
		written, err := copyBuffer(dst, src)
		log.Debugf("%s wrote %v, err: %v", desc, written, err)
		errc <- err
	}

	go streamFn(outConn, c, errClient, "docker -> client")
	go streamFn(c, outConn, errBackend, "client -> docker")

	select {
	case err = <-errClient:
		if err != nil {
			log.Error("hijack: Error when copying from docker to client")
		} else {
			log.Debugf("Closed connection by docker")
		}
	case err = <-errBackend:
		if err != nil {
			log.Debugf("hijack: Error when copying from docker to client", err)
		} else {
			log.Debug("Closed connection by docker")
		}
	}

	c.Close()
	outConn.Close()
	clientconn.Close()
	inConn.Close()
}

func copyBuffer(dst io.Writer, src io.Reader) (int64, error) {
	var buf = make([]byte, 100)
	var written int64
	for {
		nr, rerr := src.Read(buf)
		if rerr != nil && rerr != io.EOF && rerr != context.Canceled {
			log.Debugf("read error during body copy: %v", rerr)
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
			flushResponse(dst);
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

	b1, b2 := bufio.NewReader(&buf), ioutil.NopCloser(bytes.NewReader(buf.Bytes()))

	head, _ := b1.Peek(262)

	var tr *tar.Reader

	if(filetype.IsType(head, matchers.TypeGz)) {
		gzip_reader, _ := gzip.NewReader(b1)
		tr = tar.NewReader(gzip_reader)
	} else if(filetype.IsType(head, matchers.TypeBz2)) {
		bz2_reader := bzip2.NewReader(b1)
		tr = tar.NewReader(bz2_reader)
	} else if(filetype.IsType(head, matchers.TypeXz)) {
		xz_reader, _ := xz.NewReader(b1, 0)
		tr = tar.NewReader(xz_reader)
	} else if(filetype.IsType(head, matchers.TypeTar)) {
		tr = tar.NewReader(b1)
	}

	dockerfileLoc := req.URL.Query().Get("dockerfile")

	if dockerfileLoc == "" {
		dockerfileLoc = "Dockerfile";
	}

	var valid = false

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			log.Fatal(err)
		}
		if hdr.Name == dockerfileLoc {
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
	log.Debugf("Received Request: %s", req.URL.Path)
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

func flushResponse(w io.Writer) {
	flusher, ok := w.(http.Flusher)
	if ok {
		flusher.Flush()
	}
}

/*
	Entry
*/

func main() {

	var hostSocket string
	var policyDir string

	flag.StringVar(&targetSocket, "target", "/var/run/docker.sock", "The docker socket to connect to")
	flag.StringVar(&hostSocket, "host", "/var/run/protected-docker.sock", "The docker socket to listen on")
	flag.StringVar(&policyDir, "policyDir", "/etc/docker", "The directory containing the OPA policies")
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
