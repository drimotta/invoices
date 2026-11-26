package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
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
	Params   *Params
	Write    bool
	File     string // empty = stdin
	Currency Currency
	Env      map[string]string
}

var defaultCurrency = Currency{Symbol: "USD", Grouper: ",", GroupSize: 3, DecimalSeparator: ".", DecimalSize: 2}

// loadConfig reads .env, the output mode and the params file, and resolves
// the currency (params > .env > USD default).
func loadConfig() (*Config, error) {
	cfg, err := loadConf()
	if err != nil {
		return nil, err
	}
	if cfg.Params, err = loadInvoiceParams(cfg); err != nil {
		return nil, err
	}
	cfg.Currency = resolveCurrency(cfg.Params.Currency, cfg.Env)
	return cfg, nil
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

// outputFile writes pdf to stdout, or to invoice-<number>.pdf with -w.
func outputFile(cfg *Config, pdf []byte) error {
	if !cfg.Write {
		_, err := os.Stdout.Write(pdf)
		return err
	}
	return os.WriteFile(fmt.Sprintf("invoice-%d.pdf", cfg.Params.Invoice.Number), pdf, 0o644)
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "invoice:", err)
		os.Exit(1)
	}
}
