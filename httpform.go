package httpform

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"
)

// MB is 1 megabyte in bytes, i.e. 1024 * 1024
const MB = 1 << 20

type Configuration struct {
	AllowJSON      bool
	AllowForm      bool
	AllowMultipart bool

	MaxMultipartMemory int64

	DisallowUnknownFields    bool
	AllowUnknownFieldsHeader string

	structCache sync.Map
}

var Default = &Configuration{
	AllowJSON:      true,
	AllowForm:      true,
	AllowMultipart: true,

	MaxMultipartMemory: 32 * MB, // matches http.defaultMaxMemory

	DisallowUnknownFields: false,
}

func (conf Configuration) Clone() *Configuration {
	conf.structCache = sync.Map{}
	return &conf
}

func (conf *Configuration) Strict() *Configuration {
	conf = conf.Clone()
	conf.DisallowUnknownFields = true
	conf.AllowUnknownFieldsHeader = ""
	return conf
}

// Decode ...
//
// Warning: use LimitBody on request before calling Decode to avoid out-of-memory DoS attacks.
func (conf *Configuration) Decode(r *http.Request, pathParams any, dest any) error {
	return conf.DecodeVal(r, pathParams, reflect.ValueOf(dest))
}

// DecodeVal ...
//
// Warning: use LimitBody on request before calling DecodeVal to avoid out-of-memory DoS attacks.
func (conf *Configuration) DecodeVal(r *http.Request, pathParams any, destValPtr reflect.Value) error {
	if destValPtr.Kind() != reflect.Ptr {
		panic(fmt.Errorf("httpform: destination must be a pointer, got %v", destValPtr.Type()))
	}
	destVal := destValPtr.Elem()
	if destVal.Kind() != reflect.Struct {
		panic(fmt.Errorf("httpform: destination must be a pointer to a struct, got %v", destValPtr.Type()))
	}

	var cookies map[string]*http.Cookie

	sm := conf.lookupStruct(destVal.Type())

	mtype := determineMIMEType(r)
	switch mtype {
	case jsonContentType:
		if !conf.AllowJSON {
			return &Error{http.StatusUnsupportedMediaType, "JSON input not allowed", nil}
		}
		defer r.Body.Close()
		decoder := json.NewDecoder(r.Body)

		if conf.DisallowUnknownFields && (conf.AllowUnknownFieldsHeader == "" || !parseBoolDefault(r.Header.Get(conf.AllowUnknownFieldsHeader), false)) {
			decoder.DisallowUnknownFields()
		}

		err := decoder.Decode(destValPtr.Interface())
		if err != nil {
			return &Error{http.StatusBadRequest, "JSON input", err}
		}

		r.PostForm = make(url.Values) // prevent ParseForm from parsing body
		if err := r.ParseForm(); err != nil {
			return &Error{http.StatusBadRequest, "query string", err}
		}
	case "":
		r.PostForm = make(url.Values) // prevent ParseForm from parsing body
		if err := r.ParseForm(); err != nil {
			return &Error{http.StatusBadRequest, "query string", err}
		}
	case formContentType:
		if err := r.ParseForm(); err != nil {
			return &Error{http.StatusBadRequest, "", err}
		}
	case multipartFormContentType:
		err := r.ParseMultipartForm(conf.MaxMultipartMemory)
		if err != nil {
			return &Error{http.StatusBadRequest, "", err}
		}
	}

	for k, vv := range r.Form {
		for _, v := range vv {
			err := setVal(destVal, sm, formSrc, k, v)
			if err != nil {
				return &Error{http.StatusBadRequest, "", err}
			}
		}
	}

	pp := interpretPathParams(pathParams)

	for _, fm := range sm.NamedFields {
		switch fm.Source {
		case pathSrc:
			v := pp.Get(fm.name)
			if v == "" {
				panic(fmt.Errorf("httpform: missing path parameter %q, got: %s", fm.name, strings.Join(pp.Keys(), ", ")))
			}
			err := setField(destVal, fm, v)
			if err != nil {
				return &Error{http.StatusBadRequest, "", err}
			}
		case cookieSrc:
			if cookies == nil {
				cookies = make(map[string]*http.Cookie)
				for _, cookie := range r.Cookies() {
					cookies[cookie.Name] = cookie
				}
			}
			c := cookies[fm.name]
			if c != nil {
				err := setField(destVal, fm, c.Value)
				if err != nil {
					return &Error{http.StatusBadRequest, "", err}
				}
			}
		default:
			break
		}
	}

	return nil
}

// EncodeToValues is a counterpart to Decode.
func (conf *Configuration) EncodeToValues(source any, values url.Values) {
	sourceVal := reflect.ValueOf(source)
	if sourceVal.Kind() == reflect.Ptr {
		if sourceVal.IsNil() {
			return
		}
		sourceVal = sourceVal.Elem()
	}
	if sourceVal.Kind() != reflect.Struct {
		panic(fmt.Errorf("httpform: source must be a struct (or a pointer to one), got %T", source))
	}

	sm := conf.lookupStruct(sourceVal.Type())

	for _, fm := range sm.NamedFields {
		if fm.Source != formSrc {
			continue
		}
		values.Set(fm.name, getString(sourceVal, fm))
	}
}

func (conf *Configuration) EncodeToPath(source any, path string) string {
	sourceVal := reflect.ValueOf(source)
	if sourceVal.Kind() == reflect.Ptr {
		if sourceVal.IsNil() {
			return path
		}
		sourceVal = sourceVal.Elem()
	}
	if sourceVal.Kind() != reflect.Struct {
		panic(fmt.Errorf("httpform: source must be a struct (or a pointer to one), got %T", source))
	}

	sm := conf.lookupStruct(sourceVal.Type())

	origPath := path
	for _, fm := range sm.NamedFields {
		if fm.Source != pathSrc {
			continue
		}
		s := getString(sourceVal, fm)
		key := ":" + fm.name
		newPath := strings.ReplaceAll(path, key, s)
		if newPath == path {
			panic(fmt.Errorf("%s is not found in %s", key, origPath))
		}
		path = newPath
	}

	return path
}

type source int

const (
	noSrc = source(iota)
	pathSrc
	formSrc
	cookieSrc
	requestSrc // sources here and below are unnamed
	urlSrc
	queryValuesSrc
)

var _sources = []string{"none", "path", "form", "cookie", "request", "url", "query values"}

func (v source) String() string {
	return _sources[v]
}

func (v source) IsNamed() bool {
	return v < requestSrc
}
