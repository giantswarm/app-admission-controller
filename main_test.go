package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/dyson/certman"
)

func Test_healthCheck(t *testing.T) {
	// Initially create certificate
	err := copyCertificate("testdata/certs/old")
	if err != nil {
		t.Fatalf("error == %#v, want nil", err.Error())
	}

	// Load certificates with the certman
	cm, err := certman.New("testdata/certs/current/tls.crt", "testdata/certs/current/tls.key")
	if err != nil {
		t.Fatalf("error == %#v, want nil", err.Error())
	}

	if err := cm.Watch(); err != nil {
		t.Fatalf("error == %#v, want nil", err.Error())
	}

	req, err := http.NewRequest("GET", "/healthz", nil)
	if err != nil {
		t.Fatalf("error == %#v, want nil", err.Error())
	}

	// Check health endpoint
	rec := httptest.NewRecorder()
	healthCheck(rec, req, cm, "testdata/certs/current/tls.crt", "testdata/certs/current/tls.key")
	if rec.Code != 200 {
		t.Fatalf("got %v HTTP response, want 200", rec.Code)
	}

	// Certificates are good, now reload them
	err = copyCertificate("testdata/certs/new")
	if err != nil {
		t.Fatalf("error == %#v, want nil", err.Error())
	}

	// Check health endpoint again
	rec = httptest.NewRecorder()
	healthCheck(rec, req, cm, "testdata/certs/current/tls.crt", "testdata/certs/current/tls.key")
	if rec.Code != 503 {
		t.Fatalf("got %v HTTP response, want 503", rec.Code)
	}
}

func copyCertificate(path string) error {
	r, err := os.Open(path + "/tls.crt")
	if err != nil {
		return err
	}

	w, err := os.Create("testdata/certs/current/tls.crt")
	if err != nil {
		return err
	}

	_, err = io.Copy(w, r)
	if err != nil {
		return err
	}

	err = r.Close()
	if err != nil {
		return err
	}

	err = w.Close()
	if err != nil {
		return err
	}

	r, err = os.Open(path + "/tls.key")
	if err != nil {
		return err
	}

	w, err = os.Create("testdata/certs/current/tls.key")
	if err != nil {
		return err
	}

	_, err = io.Copy(w, r)
	if err != nil {
		return err
	}

	err = r.Close()
	if err != nil {
		return err
	}

	err = w.Close()
	if err != nil {
		return err
	}

	return nil
}
