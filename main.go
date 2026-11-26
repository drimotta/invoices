package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gpdf-dev/gpdf/document"
	"github.com/gpdf-dev/gpdf/pdf"
	"github.com/gpdf-dev/gpdf/template"
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
		Logo    string `json:"logo"`
	} `json:"company"`
	Currency *Currency `json:"currency"`
}

type Config struct {
	Write    bool
	File     string // empty = stdin
	Currency Currency
	Env      map[string]string
}

var defaultCurrency = Currency{Symbol: "USD", Grouper: ",", GroupSize: 3, DecimalSeparator: ".", DecimalSize: 2}

func main() {
	cfg, err := loadConf()
	check(err)
	p, err := loadInvoiceParams(cfg)
	check(err)
	cfg.Currency = resolveCurrency(p.Currency, cfg.Env)

	doc, page := initializePage(p)
	buildHeader(page, p)
	buildCustomer(page, p)
	buildItems(page, p, cfg.Currency)
	buildCompany(page, p)

	out, err := doc.Generate()
	check(err)
	check(outputFile(cfg, p, out))
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "invoice:", err)
		os.Exit(1)
	}
}

// loadConf reads .env and the output mode flags.
func loadConf() (*Config, error) {
	w := flag.Bool("w", false, "write to invoice-<number>.pdf instead of stdout")
	flag.Parse()
	cfg := &Config{Write: *w, Env: map[string]string{}}
	if flag.NArg() > 0 {
		cfg.File = flag.Arg(0)
	}
	if f, err := os.Open(".env"); err == nil {
		defer f.Close()
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			k, v, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			cfg.Env[strings.TrimSpace(k)] = strings.Trim(strings.TrimSpace(v), `"'`)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return cfg, nil
}

func loadInvoiceParams(cfg *Config) (*Params, error) {
	var r io.Reader = os.Stdin
	if cfg.File != "" {
		f, err := os.Open(cfg.File)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		r = f
	}
	var p Params
	if err := json.NewDecoder(r).Decode(&p); err != nil {
		return nil, fmt.Errorf("parsing params: %w", err)
	}
	return &p, nil
}

// resolveCurrency: params first, then .env, then USD default.
func resolveCurrency(fromParams *Currency, env map[string]string) Currency {
	if fromParams != nil && fromParams.Symbol != "" {
		return *fromParams
	}
	if s := env["CURRENCY_SYMBOL"]; s != "" {
		c := Currency{Symbol: s, Grouper: env["CURRENCY_GROUPER"], DecimalSeparator: env["CURRENCY_DECIMAL_SEPARATOR"]}
		c.GroupSize, _ = strconv.Atoi(env["CURRENCY_GROUP_SIZE"])
		c.DecimalSize, _ = strconv.Atoi(env["CURRENCY_DECIMAL_SIZE"])
		return c
	}
	return defaultCurrency
}

// formatCurrency follows the spec'd reverse-walk algorithm.
func formatCurrency(c Currency, cents int64) string {
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

func formatDue(due string) string {
	t, err := time.Parse("2006-01-02", due)
	if err != nil {
		return due
	}
	return t.Format("02 January 2006")
}

// outputFile writes pdf to stdout, or to invoice-<number>.pdf with -w.
func outputFile(cfg *Config, p *Params, pdf []byte) error {
	if !cfg.Write {
		_, err := os.Stdout.Write(pdf)
		return err
	}
	return os.WriteFile(fmt.Sprintf("invoice-%d.pdf", p.Invoice.Number), pdf, 0o644)
}

func initializePage(p *Params) (*template.Document, *template.PageBuilder) {
	doc := template.New(
		template.WithPageSize(document.A4),
		template.WithMargins(document.UniformEdges(document.Mm(20))),
		template.WithMetadata(document.DocumentMetadata{
			Title:  fmt.Sprintf("Invoice #%d", p.Invoice.Number),
			Author: p.Company.Name,
		}),
	)
	return doc, doc.AddPage()
}

// lines emits one Text per line so multiline details render as separate lines.
func lines(c *template.ColBuilder, text string, opts ...template.TextOption) {
	for _, l := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		c.Text(l, opts...)
	}
}

func spacer(page *template.PageBuilder, h float64) {
	page.AutoRow(func(r *template.RowBuilder) {
		r.Col(12, func(c *template.ColBuilder) { c.Spacer(document.Mm(h)) })
	})
}

// loadLogo returns SVG bytes from an inline SVG string or a file path; nil if
// absent or not an SVG.
func loadLogo(logo string) []byte {
	if logo == "" {
		return nil
	}
	data := []byte(logo)
	if !bytes.Contains(data, []byte("<svg")) {
		var err error
		if data, err = os.ReadFile(logo); err != nil {
			fmt.Fprintln(os.Stderr, "invoice: logo ignored:", err)
			return nil
		}
	}
	if !bytes.Contains(data, []byte("<svg")) {
		return nil
	}
	return data
}

func buildHeader(page *template.PageBuilder, p *Params) {
	logo := loadLogo(p.Company.Logo)
	page.AutoRow(func(r *template.RowBuilder) {
		r.Col(6, func(c *template.ColBuilder) {
			c.Text("INVOICE", template.FontSize(24), template.Bold())
			c.Text(fmt.Sprintf("Invoice #%d", p.Invoice.Number))
			c.Text("Payment due " + formatDue(p.Invoice.Due))
		})
		r.Col(6, func(c *template.ColBuilder) {
			if logo != nil {
				c.Image(logo, template.FitWidth(document.Pct(100)))
			}
		})
	})
	spacer(page, 8)
}

func buildCustomer(page *template.PageBuilder, p *Params) {
	page.AutoRow(func(r *template.RowBuilder) {
		r.Col(12, func(c *template.ColBuilder) { c.Text("BILL TO", template.Bold()) })
	})
	page.AutoRow(func(r *template.RowBuilder) {
		r.Col(6, func(c *template.ColBuilder) { c.Text(p.Customer.Name) })
		r.Col(6, func(c *template.ColBuilder) { lines(c, p.Customer.Details) })
	})
	spacer(page, 8)
}

func buildItems(page *template.PageBuilder, p *Params, cur Currency) {
	var subtotal int64 // accumulator: avoids recalculating at the end
	rows := make([][]string, 0, len(p.Items)+1)
	for _, it := range p.Items {
		total := it.PriceCents * it.Qty
		subtotal += total
		rows = append(rows, []string{it.Name, formatCurrency(cur, it.PriceCents), strconv.FormatInt(it.Qty, 10), formatCurrency(cur, total)})
	}
	rows = append(rows, []string{"Subtotal", "", "", formatCurrency(cur, subtotal)})
	page.AutoRow(func(r *template.RowBuilder) {
		r.Col(12, func(c *template.ColBuilder) {
			c.Table([]string{"Item", "Price", "Qty", "Total"}, rows,
				template.ColumnWidths(46, 20, 10, 24),
				template.ColumnAlign(document.AlignLeft, document.AlignRight, document.AlignRight, document.AlignRight),
				template.TableHeaderStyle(template.Bold(), template.TextColor(pdf.White), template.BgColor(pdf.Gray(0.2))),
			)
		})
	})
	spacer(page, 10)
}

func buildCompany(page *template.PageBuilder, p *Params) {
	page.AutoRow(func(r *template.RowBuilder) {
		r.Col(6, func(c *template.ColBuilder) {
			c.Text(p.Company.Name, template.Bold())
			lines(c, p.Company.Details)
		})
		r.Col(6, func(c *template.ColBuilder) { lines(c, p.Company.Banking) })
	})
}
