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

	"software.drimotta.dev/invoices"
)

type Config struct {
	Params *invoices.Params
	Write  bool
	File   string // empty = stdin
	Env    map[string]string
}

// loadConfig reads .env, the output mode and the params file, and resolves
// the currency (params > .env > USD default) and the logo file.
func loadConfig() (*Config, error) {
	cfg, err := loadConf()
	if err != nil {
		return nil, err
	}
	if cfg.Params, err = loadInvoiceParams(cfg); err != nil {
		return nil, err
	}
	c := resolveCurrency(cfg.Params.Currency, cfg.Env)
	cfg.Params.Currency = &c
	cfg.Params.Company.Logo = resolveLogo(cfg.Params.Company.Logo)
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

func loadInvoiceParams(cfg *Config) (*invoices.Params, error) {
	var r io.Reader = os.Stdin
	if cfg.File != "" {
		f, err := os.Open(cfg.File)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		r = f
	}
	var p invoices.Params
	if err := json.NewDecoder(r).Decode(&p); err != nil {
		return nil, fmt.Errorf("parsing params: %w", err)
	}
	return &p, nil
}

// resolveCurrency: params first, then .env, then USD default.
func resolveCurrency(fromParams *invoices.Currency, env map[string]string) invoices.Currency {
	if fromParams != nil && fromParams.Symbol != "" {
		return *fromParams
	}
	if s := env["CURRENCY_SYMBOL"]; s != "" {
		c := invoices.Currency{Symbol: s, Grouper: env["CURRENCY_GROUPER"], DecimalSeparator: env["CURRENCY_DECIMAL_SEPARATOR"]}
		c.GroupSize, _ = strconv.Atoi(env["CURRENCY_GROUP_SIZE"])
		c.DecimalSize, _ = strconv.Atoi(env["CURRENCY_DECIMAL_SIZE"])
		return c
	}
	return invoices.DefaultCurrency
}

// resolveLogo returns an inline SVG as is, or the contents of an SVG file;
// anything else becomes no logo.
func resolveLogo(logo string) string {
	if logo == "" || strings.Contains(logo, "<svg") {
		return logo
	}
	data, err := os.ReadFile(logo)
	if err != nil {
		fmt.Fprintln(os.Stderr, "invoice: logo ignored:", err)
		return ""
	}
	if !strings.Contains(string(data), "<svg") {
		return ""
	}
	return string(data)
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
