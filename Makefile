# The C API check: builds libinvoices and renders example.json from C.
CC ?= cc
OUT ?= build
UNAME := $(shell uname)
ifeq ($(UNAME),Darwin)
EXT := dylib
else
EXT := so
endif

.PHONY: lib c-test
lib:
	mkdir -p $(OUT)
	go build -buildmode=c-shared -o $(OUT)/libinvoices.$(EXT) ./capi

c-test: lib
	$(CC) -Wall -Wextra -Werror -std=c11 -Iinclude tests/c/smoke.c -L$(OUT) -linvoices -o $(OUT)/c-smoke
	LD_LIBRARY_PATH=$(OUT) DYLD_LIBRARY_PATH=$(OUT) $(OUT)/c-smoke
