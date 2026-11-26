package main

import "testing"

func TestFormatCurrency(t *testing.T) {
	cases := []struct {
		c    Currency
		in   int64
		want string
	}{
		{Currency{"JPY", "", 0, "", 0}, 12345, "JPY12345"},
		{Currency{"BRL", ".", 3, ",", 2}, 1234567, "BRL12.345,67"},
		{Currency{"USD", ",", 3, ".", 2}, 1234567, "USD12,345.67"},
		{Currency{"USD", ",", 3, ".", 2}, 5, "USD0.05"},
		{Currency{"USD", ",", 3, ".", 2}, 123456, "USD1,234.56"},
	}
	for _, tc := range cases {
		if got := formatCurrency(tc.c, tc.in); got != tc.want {
			t.Errorf("%v %d: got %q want %q", tc.c, tc.in, got, tc.want)
		}
	}
}
