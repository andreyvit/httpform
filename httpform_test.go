package httpform

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestDecode_header_string(t *testing.T) {
	var in struct {
		Foo string `form:"X-Foo,header" json:"-"`
	}
	r := httptest.NewRequest("GET", "https://example.com/foo", nil)
	r.Header.Set("X-Foo", "bar")
	ok(t, Default.Decode(r, nil, &in))
	eq(t, in.Foo, "bar")
}

func TestDecode_header_int(t *testing.T) {
	var in struct {
		Foo int `form:"X-Foo,header" json:"-"`
	}
	r := httptest.NewRequest("GET", "https://example.com/foo", nil)
	r.Header.Set("X-Foo", "42")
	ok(t, Default.Decode(r, nil, &in))
	eq(t, in.Foo, 42)
}

func TestDecode_header_int_invalid(t *testing.T) {
	var in struct {
		Foo int `form:"X-Foo,header" json:"-"`
	}
	r := httptest.NewRequest("GET", "https://example.com/foo", nil)
	r.Header.Set("X-Foo", "bar")
	fails(t, Default.Decode(r, nil, &in), `[400] invalid X-Foo: strconv.ParseInt: parsing "bar": invalid syntax`)
}

func TestDecode_header_missing(t *testing.T) {
	var in struct {
		Foo string `form:"X-Foo,header" json:"-"`
	}
	r := httptest.NewRequest("GET", "https://example.com/foo", nil)
	fails(t, Default.Decode(r, nil, &in), "[400] missing header X-Foo")
}

func TestDecode_header_missing_optional(t *testing.T) {
	var in struct {
		Foo string `form:"X-Foo,header,optional" json:"-"`
	}
	r := httptest.NewRequest("GET", "https://example.com/foo", nil)
	ok(t, Default.Decode(r, nil, &in))
}

func TestDecode_headers(t *testing.T) {
	var in struct {
		Foo http.Header `json:"-"`
	}
	r := httptest.NewRequest("GET", "https://example.com/foo", nil)
	r.Header.Set("X-Foo", "bar")
	ok(t, Default.Decode(r, nil, &in))
	eq(t, in.Foo.Get("X-Foo"), "bar")
}

func ok(t testing.TB, err error) {
	if err != nil {
		t.Helper()
		t.Fatalf("** %v", err)
	}
}

func eq[T comparable](t testing.TB, a, e T) {
	if a != e {
		t.Helper()
		t.Fatalf("** got %v, wanted %v", a, e)
	}
}

func deepEqual(t testing.TB, a, e any) {
	if !reflect.DeepEqual(a, e) {
		t.Helper()
		t.Errorf("** got %v, wanted %v", a, e)
	}
}

func fails(t testing.TB, err error, e string) {
	t.Helper()
	if e == "" {
		if err != nil {
			t.Errorf("** failed: %v", err)
		}
	} else {
		if err == nil {
			t.Errorf("** succeeded, expected to fail with: %v", e)
		} else if a := err.Error(); a != e {
			t.Errorf("** failed with:\n\t%v\nexpected to fail with:\n\t%v", a, e)
		}
	}
}
