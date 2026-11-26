package main

import (
	"os"
	"path/filepath"
	"testing"

	"software.drimotta.dev/invoices"
)

func TestCurrencyComesFromParamsThenEnvThenUSD(t *testing.T) {
	brl := &invoices.Currency{Symbol: "R$", Grouper: ".", GroupSize: 3, DecimalSeparator: ",", DecimalSize: 2}
	env := map[string]string{"CURRENCY_SYMBOL": "EUR", "CURRENCY_GROUPER": ".", "CURRENCY_GROUP_SIZE": "3",
		"CURRENCY_DECIMAL_SEPARATOR": ",", "CURRENCY_DECIMAL_SIZE": "2"}
	if got := resolveCurrency(brl, env); got != *brl {
		t.Errorf("params: %v", got)
	}
	if got := resolveCurrency(nil, env); got.Symbol != "EUR" || got.DecimalSize != 2 {
		t.Errorf("env: %v", got)
	}
	if got := resolveCurrency(nil, map[string]string{}); got != invoices.DefaultCurrency {
		t.Errorf("default: %v", got)
	}
}

func TestLogoFileIsInlined(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "logo.svg")
	if err := os.WriteFile(path, []byte("<svg id='x'></svg>"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := resolveLogo(path); got != "<svg id='x'></svg>" {
		t.Errorf("file: %q", got)
	}
	if got := resolveLogo("<svg></svg>"); got != "<svg></svg>" {
		t.Errorf("inline: %q", got)
	}
	if got := resolveLogo(filepath.Join(dir, "missing.svg")); got != "" {
		t.Errorf("missing: %q", got)
	}
}
