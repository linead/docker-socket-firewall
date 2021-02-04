package main

import (
	"bytes"
	"io"
	"net/http"
	"testing"
)

func Test_serveReverseProxy(t *testing.T) {
	type args struct {
		w   http.ResponseWriter
		req *http.Request
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serveReverseProxy(tt.args.w, tt.args.req)
		})
	}
}

func Test_hijack(t *testing.T) {
	type args struct {
		req *http.Request
		w   http.ResponseWriter
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hijack(tt.args.req, tt.args.w)
		})
	}
}

func Test_copyBuffer(t *testing.T) {
	type args struct {
		src io.Reader
	}
	tests := []struct {
		name    string
		args    args
		want    int64
		wantDst string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dst := &bytes.Buffer{}
			got, err := copyBuffer(dst, tt.args.src)
			if (err != nil) != tt.wantErr {
				t.Errorf("copyBuffer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("copyBuffer() = %v, want %v", got, tt.want)
			}
			if gotDst := dst.String(); gotDst != tt.wantDst {
				t.Errorf("copyBuffer() = %v, want %v", gotDst, tt.wantDst)
			}
		})
	}
}

func Test_copyHeader(t *testing.T) {
	type args struct {
		dst http.Header
		src http.Header
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			copyHeader(tt.args.dst, tt.args.src)
		})
	}
}

func Test_verifyBuildInstruction(t *testing.T) {
	type args struct {
		req *http.Request
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := verifyBuildInstruction(tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("verifyBuildInstruction() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("verifyBuildInstruction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_handleRequestAndRedirect(t *testing.T) {
	type args struct {
		res http.ResponseWriter
		req *http.Request
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handleRequestAndRedirect(tt.args.res, tt.args.req)
		})
	}
}

func Test_listenAndServe(t *testing.T) {
	type args struct {
		sockPath string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := listenAndServe(tt.args.sockPath); (err != nil) != tt.wantErr {
				t.Errorf("listenAndServe() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_flushResponse(t *testing.T) {
	tests := []struct {
		name  string
		wantW string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			flushResponse(w)
			if gotW := w.String(); gotW != tt.wantW {
				t.Errorf("flushResponse() = %v, want %v", gotW, tt.wantW)
			}
		})
	}
}

func Test_main(t *testing.T) {
	tests := []struct {
		name string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			main()
		})
	}
}
