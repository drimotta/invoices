package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"os"
	"testing"
)

func decode(t *testing.T, s string) map[string]any {
	t.Helper()
	var v map[string]any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		t.Fatalf("response %q: %v", s, err)
	}
	return v
}

func TestRenderReturnsThePDF(t *testing.T) {
	params, err := os.ReadFile("../example.json")
	if err != nil {
		t.Fatal(err)
	}
	code, resp := renderJSON(string(params))
	if code != codeOK {
		t.Fatalf("code %d: %s", code, resp)
	}
	pdf, err := base64.StdEncoding.DecodeString(decode(t, resp)["pdf_b64"].(string))
	if err != nil || !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatalf("not a PDF: %v", err)
	}
}

func TestBadRequestsAreInputErrors(t *testing.T) {
	for _, req := range []string{"", "[1]", "{", `{"invoice":{"number":0}}`} {
		code, resp := renderJSON(req)
		if code != codeInput {
			t.Errorf("%q: code %d", req, code)
		}
		e := decode(t, resp)["error"].(map[string]any)
		if e["code"] != "input" || e["message"] == "" {
			t.Errorf("%q: %v", req, e)
		}
	}
}
