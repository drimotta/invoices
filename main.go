package main

func main() {
	cfg, err := loadConfig()
	check(err)
	out, err := buildInvoice(cfg)
	check(err)
	check(outputFile(cfg, out))
}
