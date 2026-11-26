package main

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gpdf-dev/gpdf/document"
	"github.com/gpdf-dev/gpdf/pdf"
	"github.com/gpdf-dev/gpdf/template"
)

// buildInvoice renders the invoice described by cfg and returns the PDF bytes.
func buildInvoice(cfg *Config) ([]byte, error) {
	p := cfg.Params
	page, doc := initializePage(p)
	buildHeader(page, p)
	buildCustomer(page, p)
	buildItems(page, p, cfg.Currency)
	buildCompany(page, p)
	return doc.Generate()
}

func formatDue(due string) string {
	t, err := time.Parse("2006-01-02", due)
	if err != nil {
		return due
	}
	return t.Format("02 January 2006")
}

func initializePage(p *Params) (*template.PageBuilder, *template.Document) {
	doc := template.New(
		template.WithPageSize(document.A4),
		template.WithMargins(document.UniformEdges(document.Mm(20))),
		template.WithMetadata(document.DocumentMetadata{
			Title:  fmt.Sprintf("Invoice #%d", p.Invoice.Number),
			Author: p.Company.Name,
		}),
	)
	return doc.AddPage(), doc
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
