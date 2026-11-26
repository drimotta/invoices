// Command invoices renders an invoice PDF from a params JSON file or stdin.
package main

import "software.drimotta.dev/invoices"

func main() {
	cfg, err := loadConfig()
	check(err)
	out, err := invoices.Render(cfg.Params)
	check(err)
	check(outputFile(cfg, out))
}
