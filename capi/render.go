package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"software.drimotta.dev/invoices"
)

const (
	codeOK       = 0
	codeInput    = 1
	codeInternal = 99
)

func errorResponse(code int, name string, err error) (int, string) {
	body, _ := json.Marshal(map[string]any{"error": map[string]string{"code": name, "message": err.Error()}})
	return code, string(body)
}

// renderJSON renders the params JSON and returns the status and the response
// JSON, as the C API hands them out.
func renderJSON(request string) (code int, response string) {
	defer func() {
		if r := recover(); r != nil {
			code, response = errorResponse(codeInternal, "internal", fmt.Errorf("panic: %v", r))
		}
	}()
	dec := json.NewDecoder(bytes.NewReader([]byte(request)))
	dec.DisallowUnknownFields()
	var p invoices.Params
	if err := dec.Decode(&p); err != nil {
		return errorResponse(codeInput, "input", fmt.Errorf("bad JSON: %w", err))
	}
	pdf, err := invoices.Render(&p)
	if err != nil {
		return errorResponse(codeInput, "input", err)
	}
	body, err := json.Marshal(map[string]string{"pdf_b64": base64.StdEncoding.EncodeToString(pdf)})
	if err != nil {
		return errorResponse(codeInternal, "internal", err)
	}
	return codeOK, string(body)
}
