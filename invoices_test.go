package invoices

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func example(t *testing.T) *Params {
	t.Helper()
	data, err := os.ReadFile("example.json")
	if err != nil {
		t.Fatal(err)
	}
	var p Params
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatal(err)
	}
	return &p
}

func TestFormatCurrency(t *testing.T) {
	cases := []struct {
		c    Currency
		in   int64
		want string
	}{
		{Currency{"JPY", "", 0, "", 0}, 12345, "JPY12345"},
		{Currency{"BRL", ".", 3, ",", 2}, 1234567, "BRL12.345,67"},
		{Currency{"USD", ",", 3, ".", 2}, 1234567, "USD12,345.67"},
		{Currency{"USD", ",", 3, ".", 2}, 5, "USD0.05"},
		{Currency{"USD", ",", 3, ".", 2}, 123456, "USD1,234.56"},
	}
	for _, tc := range cases {
		if got := FormatCurrency(tc.c, tc.in); got != tc.want {
			t.Errorf("%v %d: got %q want %q", tc.c, tc.in, got, tc.want)
		}
	}
}

func TestRenderGivesAPDF(t *testing.T) {
	pdf, err := Render(example(t))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatalf("not a PDF: %q", pdf[:min(len(pdf), 16)])
	}
}

func TestRenderDefaultsToUSD(t *testing.T) {
	p := example(t)
	p.Currency = nil
	if _, err := Render(p); err != nil {
		t.Fatal(err)
	}
	if got := p.EffectiveCurrency(); got != DefaultCurrency {
		t.Errorf("got %v", got)
	}
}

func TestInvalidParamsAreRefused(t *testing.T) {
	cases := map[string]func(*Params){
		"invoice.number":       func(p *Params) { p.Invoice.Number = 0 },
		"invoice.due":          func(p *Params) { p.Invoice.Due = "01/11/2001" },
		"items":                func(p *Params) { p.Items = nil },
		"items[0].qty":         func(p *Params) { p.Items[0].Qty = 0 },
		"items[1].price_cents": func(p *Params) { p.Items[1].PriceCents = -1 },
		"items[0].name":        func(p *Params) { p.Items[0].Name = " " },
		"customer.name":        func(p *Params) { p.Customer.Name = "" },
		"company.name":         func(p *Params) { p.Company.Name = "" },
	}
	for field, breakIt := range cases {
		p := example(t)
		breakIt(p)
		_, err := Render(p)
		if err == nil || !strings.Contains(err.Error(), field) {
			t.Errorf("%s: got %v", field, err)
		}
	}
	if _, err := Render(nil); err == nil {
		t.Error("nil params accepted")
	}
}

func TestLogoMustBeInlineSVG(t *testing.T) {
	if LogoSVG("example.json") != nil {
		t.Error("a path was read as a logo")
	}
	if LogoSVG("<svg></svg>") == nil {
		t.Error("inline SVG refused")
	}
	if LogoSVG("") != nil {
		t.Error("empty logo")
	}
}
