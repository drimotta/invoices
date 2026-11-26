// Package invoices renders invoice PDFs from JSON-shaped params. It reads no
// files or environment: the CLI in cmd/invoices and the C API in capi do.
package invoices

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Item struct {
	Name       string `json:"name"`
	PriceCents int64  `json:"price_cents"`
	Qty        int64  `json:"qty"`
}

type Currency struct {
	Symbol           string `json:"symbol"`
	Grouper          string `json:"grouper"`
	GroupSize        int    `json:"group_size"`
	DecimalSeparator string `json:"decimal_separator"`
	DecimalSize      int    `json:"decimal_size"`
}

type Params struct {
	Items    []Item `json:"items"`
	Customer struct {
		Name    string `json:"name"`
		Details string `json:"details"`
	} `json:"customer"`
	Invoice struct {
		Number int    `json:"number"`
		Due    string `json:"due"` // YYYY-MM-DD
	} `json:"invoice"`
	Company struct {
		Name    string `json:"name"`
		Details string `json:"details"`
		Banking string `json:"banking"`
		// Logo is an inline SVG document; anything else is ignored.
		Logo string `json:"logo"`
	} `json:"company"`
	Currency *Currency `json:"currency"`
}

var DefaultCurrency = Currency{Symbol: "USD", Grouper: ",", GroupSize: 3, DecimalSeparator: ".", DecimalSize: 2}

// EffectiveCurrency is the params' currency, or USD.
func (p *Params) EffectiveCurrency() Currency {
	if p.Currency != nil && p.Currency.Symbol != "" {
		return *p.Currency
	}
	return DefaultCurrency
}

// Validate reports every problem, each prefixed with its field.
func (p *Params) Validate() error {
	if p == nil {
		return errors.New("no params")
	}
	var errs []error
	bad := func(field, format string, args ...any) {
		errs = append(errs, fmt.Errorf("%s: "+format, append([]any{field}, args...)...))
	}
	if p.Invoice.Number <= 0 {
		bad("invoice.number", "must be positive, got %d", p.Invoice.Number)
	}
	if _, err := time.Parse("2006-01-02", p.Invoice.Due); err != nil {
		bad("invoice.due", "must be YYYY-MM-DD, got %q", p.Invoice.Due)
	}
	if len(p.Items) == 0 {
		bad("items", "at least one is required")
	}
	for i, it := range p.Items {
		f := "items[" + strconv.Itoa(i) + "]"
		if strings.TrimSpace(it.Name) == "" {
			bad(f+".name", "is required")
		}
		if it.Qty <= 0 {
			bad(f+".qty", "must be positive, got %d", it.Qty)
		}
		if it.PriceCents < 0 {
			bad(f+".price_cents", "cannot be negative, got %d", it.PriceCents)
		}
	}
	if strings.TrimSpace(p.Customer.Name) == "" {
		bad("customer.name", "is required")
	}
	if strings.TrimSpace(p.Company.Name) == "" {
		bad("company.name", "is required")
	}
	return errors.Join(errs...)
}

// Render validates the params and returns the invoice PDF.
func Render(p *Params) ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return buildInvoice(p)
}

// LogoSVG returns the logo when it is an inline SVG document, else nil.
func LogoSVG(logo string) []byte {
	if !strings.Contains(logo, "<svg") {
		return nil
	}
	return []byte(logo)
}

// FormatCurrency follows the spec'd reverse-walk algorithm.
func FormatCurrency(c Currency, cents int64) string {
	neg := cents < 0
	if neg {
		cents = -cents
	}
	rs := []rune(strconv.FormatInt(cents, 10))
	for i, j := 0, len(rs)-1; i < j; i, j = i+1, j-1 {
		rs[i], rs[j] = rs[j], rs[i]
	}
	for len(rs) < c.DecimalSize+1 { // pad so 5 cents => 0.05
		rs = append(rs, '0')
	}
	var out []rune
	decimals, group := 0, 0
	for i, r := range rs {
		if decimals < c.DecimalSize {
			out = append(out, r)
			decimals++
			if decimals == c.DecimalSize {
				out = append(out, []rune(c.DecimalSeparator)...)
			}
			continue
		}
		if c.GroupSize > 0 && group == c.GroupSize && i < len(rs) {
			out = append(out, []rune(c.Grouper)...)
			group = 0
		}
		out = append(out, r)
		group++
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	s := c.Symbol + string(out)
	if neg {
		s = "-" + s
	}
	return s
}
