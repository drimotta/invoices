/* Renders example.json through include/invoices.h. */
#include "invoices.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

static char *slurp(const char *path) {
    FILE *f = fopen(path, "rb");
    if (!f) return NULL;
    fseek(f, 0, SEEK_END);
    long n = ftell(f);
    rewind(f);
    char *buf = malloc((size_t)n + 1);
    if (buf && fread(buf, 1, (size_t)n, f) == (size_t)n) {
        buf[n] = '\0';
    } else {
        free(buf);
        buf = NULL;
    }
    fclose(f);
    return buf;
}

int main(void) {
    int failures = 0;
    char *params = slurp("example.json");
    if (!params) {
        fprintf(stderr, "cannot read example.json\n");
        return 1;
    }

    char *response = NULL;
    int rc = invoices_render(params, &response);
    if (rc != INVOICES_OK || !response || !strstr(response, "\"pdf_b64\":\"JVBERi")) {
        fprintf(stderr, "FAIL render: %d %.120s\n", rc, response ? response : "(null)");
        failures++;
    }
    invoices_free(response);
    free(params);

    response = NULL;
    rc = invoices_render("{\"invoice\":{\"number\":0}}", &response);
    if (rc != INVOICES_ERR_INPUT || !response || !strstr(response, "invoice.number")) {
        fprintf(stderr, "FAIL invalid: %d %s\n", rc, response ? response : "(null)");
        failures++;
    }
    invoices_free(response);

    response = NULL;
    rc = invoices_render(NULL, &response);
    if (rc != INVOICES_ERR_INPUT) {
        fprintf(stderr, "FAIL null: %d\n", rc);
        failures++;
    }
    invoices_free(response);
    invoices_free(NULL);

    if (failures) return 1;
    puts("C API smoke test passed");
    return 0;
}
