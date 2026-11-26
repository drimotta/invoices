/*
 * invoices: render invoice PDFs.
 *
 * invoices_render takes the params JSON (as example.json; the logo must be
 * an inline SVG) and writes a JSON response to *response, released with
 * invoices_free(): {"pdf_b64": "..."} on success, or
 * {"error": {"code": "input", "message": "..."}}.
 */
#ifndef INVOICES_H
#define INVOICES_H

#ifdef __cplusplus
extern "C" {
#endif

#define INVOICES_OK 0
/* Malformed JSON or invalid params; the message names each field. */
#define INVOICES_ERR_INPUT 1
#define INVOICES_ERR_INTERNAL 99

int invoices_render(const char *request, char **response);

/* Releases a response. NULL is ignored. */
void invoices_free(char *response);

#ifdef __cplusplus
}
#endif

#endif /* INVOICES_H */
