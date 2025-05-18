package httpform

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestDecode_query_string(t *testing.T) {
	var in struct {
		Foo string `json:"foo"`
	}
	r := httptest.NewRequest("GET", "https://example.com/subdir/?foo=bar", nil)
	ok(t, Default.Decode(r, nil, &in))
	eq(t, in.Foo, "bar")
}

func TestDecode_embedded(t *testing.T) {
	type Inner struct {
		Foo string `json:"foo"`
	}
	var in struct {
		Inner
	}
	r := httptest.NewRequest("GET", "https://example.com/subdir/?foo=bar", nil)
	ok(t, Default.Decode(r, nil, &in))
	eq(t, in.Foo, "bar")
}

func TestDecode_query_int(t *testing.T) {
	var in struct {
		Foo int `json:"foo"`
	}
	r := httptest.NewRequest("GET", "https://example.com/subdir/?foo=42", nil)
	ok(t, Default.Decode(r, nil, &in))
	eq(t, in.Foo, 42)
}

func TestDecode_urlencoded_string(t *testing.T) {
	var in struct {
		Foo string `json:"foo"`
	}
	r := httptest.NewRequest("POST", "https://example.com/subdir/", strings.NewReader(`foo=bar`))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ok(t, Default.Decode(r, nil, &in))
	eq(t, in.Foo, "bar")
}

func TestDecode_urlencoded_array(t *testing.T) {
	t.Skip("arrays not supported yet")
	var in struct {
		Foo []string `json:"foo"`
	}
	r := httptest.NewRequest("POST", "https://example.com/subdir/", strings.NewReader(`foo=bar&foo=boz`))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ok(t, Default.Decode(r, nil, &in))
	deepEqual(t, in.Foo, []string{"bar", "boz"})
}

func TestDecode_json_string(t *testing.T) {
	var in struct {
		Foo string `json:"foo"`
	}
	r := httptest.NewRequest("POST", "https://example.com/subdir/", strings.NewReader(`{ "foo": "bar" }`))
	r.Header.Set("Content-Type", "application/json")
	ok(t, Default.Decode(r, nil, &in))
	eq(t, in.Foo, "bar")
}

func TestDecode_json_invalid_ignored_when_no_form_fields(t *testing.T) {
	var in struct {
	}
	r := httptest.NewRequest("POST", "https://example.com/subdir/", strings.NewReader(`< "foo": "bar" }`))
	r.Header.Set("Content-Type", "application/json")
	ok(t, Default.Decode(r, nil, &in))
}

func TestDecode_header_string(t *testing.T) {
	var in struct {
		Foo string `form:"X-Foo,header" json:"-"`
	}
	r := httptest.NewRequest("POST", "https://example.com/subdir/", nil)
	r.Header.Set("X-Foo", "bar")
	ok(t, Default.Decode(r, nil, &in))
	eq(t, in.Foo, "bar")
}

func TestDecode_header_int(t *testing.T) {
	var in struct {
		Foo int `form:"X-Foo,header" json:"-"`
	}
	r := httptest.NewRequest("POST", "https://example.com/subdir/", nil)
	r.Header.Set("X-Foo", "42")
	ok(t, Default.Decode(r, nil, &in))
	eq(t, in.Foo, 42)
}

func TestDecode_header_int_invalid(t *testing.T) {
	var in struct {
		Foo int `form:"X-Foo,header" json:"-"`
	}
	r := httptest.NewRequest("POST", "https://example.com/subdir/", nil)
	r.Header.Set("X-Foo", "bar")
	fails(t, Default.Decode(r, nil, &in), `[400] invalid X-Foo: strconv.ParseInt: parsing "bar": invalid syntax`)
}

func TestDecode_header_missing(t *testing.T) {
	var in struct {
		Foo string `form:"X-Foo,header" json:"-"`
	}
	r := httptest.NewRequest("POST", "https://example.com/subdir/", nil)
	fails(t, Default.Decode(r, nil, &in), "[400] missing header X-Foo")
}

func TestDecode_header_missing_optional(t *testing.T) {
	var in struct {
		Foo string `form:"X-Foo,header,optional" json:"-"`
	}
	r := httptest.NewRequest("POST", "https://example.com/subdir/", nil)
	ok(t, Default.Decode(r, nil, &in))
}

func TestDecode_headers(t *testing.T) {
	var in struct {
		Foo http.Header `json:"-"`
	}
	r := httptest.NewRequest("POST", "https://example.com/subdir/", nil)
	r.Header.Set("X-Foo", "bar")
	ok(t, Default.Decode(r, nil, &in))
	eq(t, in.Foo.Get("X-Foo"), "bar")
}

func TestDecode_raw_simple(t *testing.T) {
	var in struct {
		Body string `form:",rawbody" json:"-"`
	}
	r := httptest.NewRequest("POST", "https://example.com/subdir/", strings.NewReader(`{ "foo": "bar" }`))
	r.Header.Set("Content-Type", "application/json")
	ok(t, Default.Decode(r, nil, &in))
	eq(t, in.Body, `{ "foo": "bar" }`)
}

func TestDecode_raw_mixed(t *testing.T) {
	var in struct {
		Body string `form:",rawbody" json:"-"`
		Foo  string `json:"foo"`
	}
	r := httptest.NewRequest("POST", "https://example.com/subdir/", strings.NewReader(`{ "foo": "bar" }`))
	r.Header.Set("Content-Type", "application/json")
	ok(t, Default.Decode(r, nil, &in))
	eq(t, in.Body, `{ "foo": "bar" }`)
	eq(t, in.Foo, "bar")
}

func TestDecode_raw_typed(t *testing.T) {
	var in struct {
		Body json.RawMessage `form:",rawbody" json:"-"`
	}
	r := httptest.NewRequest("POST", "https://example.com/subdir/", strings.NewReader(`{ "foo": "bar" }`))
	r.Header.Set("Content-Type", "application/json")
	ok(t, Default.Decode(r, nil, &in))
	eq(t, string(in.Body), `{ "foo": "bar" }`)
}

func TestDecode_raw_invalid(t *testing.T) {
	var in struct {
		Body []byte `form:",rawbody" json:"-"`
	}
	r := httptest.NewRequest("POST", "https://example.com/subdir/", strings.NewReader(`<"foo": "bar" }`))
	r.Header.Set("Content-Type", "application/json")
	ok(t, Default.Decode(r, nil, &in))
	eq(t, string(in.Body), `<"foo": "bar" }`)
}

func TestDecode_fullbody_solo(t *testing.T) {
	var in struct {
		Body any `form:",fullbody" json:"-"`
	}
	r := httptest.NewRequest("POST", "https://example.com/subdir/", strings.NewReader(`{ "foo": "bar" }`))
	r.Header.Set("Content-Type", "application/json")
	ok(t, Default.Decode(r, nil, &in))
	deepEqual(t, in.Body, map[string]any{"foo": "bar"})
}

func TestDecode_fullbody_mixed(t *testing.T) {
	var in struct {
		Body any    `form:",fullbody" json:"-"`
		Foo  string `json:"foo"`
	}
	r := httptest.NewRequest("POST", "https://example.com/subdir/", strings.NewReader(`{ "foo": "bar" }`))
	r.Header.Set("Content-Type", "application/json")
	ok(t, Default.Decode(r, nil, &in))
	deepEqual(t, in.Body, map[string]any{"foo": "bar"})
	eq(t, in.Foo, "bar")
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
