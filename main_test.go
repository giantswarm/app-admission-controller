package main

import (
	"encoding/base64"
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
	r, err := os.ReadFile(path + "/tls-b64.crt")
	if err != nil {
		return err
	}

	w, err := os.Create("testdata/certs/current/tls.crt")
	if err != nil {
		return err
	}

	crt, err := base64.StdEncoding.DecodeString(string(r))
	if err != nil {
		return err
	}

	_, err = w.Write(crt)
	if err != nil {
		return err
	}

	err = w.Close()
	if err != nil {
		return err
	}

	r, err = os.ReadFile(path + "/tls-b64.key")
	if err != nil {
		return err
	}

	key, err := base64.StdEncoding.DecodeString(string(r))
	if err != nil {
		return err
	}

	w, err = os.Create("testdata/certs/current/tls.key")
	if err != nil {
		return err
	}

	_, err = w.Write(key)
	if err != nil {
		return err
	}

	err = w.Close()
	if err != nil {
		return err
	}

	return nil
}
