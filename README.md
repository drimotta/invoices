# invoices

Renders an invoice PDF from a params JSON (see `example.json`).

## Command line

```bash
go run ./cmd/invoices example.json > invoice.pdf
go run ./cmd/invoices -w example.json   # writes invoice-42.pdf
```

The currency comes from the params, then from `.env`
(`CURRENCY_SYMBOL`, `CURRENCY_GROUPER`, `CURRENCY_GROUP_SIZE`,
`CURRENCY_DECIMAL_SEPARATOR`, `CURRENCY_DECIMAL_SIZE`), then USD. The logo is
an inline SVG or the path of an SVG file.

## Library

`software.drimotta.dev/invoices` exposes `Render(*Params) ([]byte, error)`.
It validates the params (every problem is reported with its field) and reads
no files or environment, so the logo must be inline SVG.

## C API

`include/invoices.h`, built with `make lib` into `build/libinvoices.so`
(`.dylib` on macOS):

```c
char *response = NULL;
int rc = invoices_render(params_json, &response);
/* rc == INVOICES_OK: {"pdf_b64": "..."}; otherwise {"error": {...}} */
invoices_free(response);
```

## Tests

```bash
go test ./...
make c-test   # builds the shared library and calls it from C
```
