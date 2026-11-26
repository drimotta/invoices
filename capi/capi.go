// Command capi builds the C library declared in include/invoices.h:
//
//	go build -buildmode=c-shared -o libinvoices.so ./capi
package main

/*
#include <stdlib.h>
*/
import "C"

import "unsafe"

//export invoices_render
func invoices_render(request *C.char, response **C.char) C.int {
	if response == nil {
		return codeInput
	}
	req := ""
	if request != nil {
		req = C.GoString(request)
	}
	code, resp := renderJSON(req)
	*response = C.CString(resp)
	return C.int(code)
}

//export invoices_free
func invoices_free(response *C.char) {
	C.free(unsafe.Pointer(response))
}

func main() {}
