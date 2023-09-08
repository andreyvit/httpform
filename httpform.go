package httpform

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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

	JSONBodyFallbackParam string

	MaxMultipartMemory int64

	DisallowUnknownFields    bool
	AllowUnknownFieldsHeader string

	structCache sync.Map
}

var Default = &Configuration{
	AllowJSON:      true,
	AllowForm:      true,
	AllowMultipart: true,

	JSONBodyFallbackParam: "_body",

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
	defer r.Body.Close()

	isBodiless := (r.Method == http.MethodGet || r.Method == http.MethodHead)

	if destValPtr.Kind() != reflect.Ptr {
		panic(fmt.Errorf("httpform: destination must be a pointer, got %v", destValPtr.Type()))
	}
	destVal := destValPtr.Elem()
	if destVal.Kind() != reflect.Struct {
		panic(fmt.Errorf("httpform: destination must be a pointer to a struct, got %v", destValPtr.Type()))
	}

	var cookies map[string]*http.Cookie

	sm := conf.lookupStruct(destVal.Type())

	body := func() io.Reader { return r.Body }
	var rawBody []byte
	if sm.HasRawBody || (sm.HasBodyForm && sm.HasFullBody) {
		var err error
		rawBody, err = io.ReadAll(r.Body)
		if err != nil {
			return &Error{400, "", err}
		}
		r.Body = io.NopCloser(bytes.NewReader(rawBody))
		body = func() io.Reader { return bytes.NewReader(rawBody) }
	}

	var fullBody any

	mtype := determineMIMEType(r)
	if isBodiless {
		mtype = ""
	}

	var isBodyParsed bool
	parseJSONBody := func(body func() io.Reader) error {
		if sm.HasBodyForm {
			decoder := json.NewDecoder(body())

			if conf.DisallowUnknownFields && (conf.AllowUnknownFieldsHeader == "" || !parseBoolDefault(r.Header.Get(conf.AllowUnknownFieldsHeader), false)) {
				decoder.DisallowUnknownFields()
			}

			err := decoder.Decode(destValPtr.Interface())
			if err != nil {
				return &Error{http.StatusBadRequest, "JSON input", err}
			}
		}
		if sm.HasFullBody {
			decoder := json.NewDecoder(body())
			err := decoder.Decode(&fullBody)
			if err != nil {
				return &Error{http.StatusBadRequest, "JSON input", err}
			}
		}
		isBodyParsed = true
		return nil
	}

	switch mtype {
	case jsonContentType:
		if !conf.AllowJSON {
			return &Error{http.StatusUnsupportedMediaType, "JSON input not allowed", nil}
		}
		if err := parseJSONBody(body); err != nil {
			return err
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
	if !isBodyParsed && conf.JSONBodyFallbackParam != "" {
		bodyStr := r.Form.Get(conf.JSONBodyFallbackParam)
		if bodyStr != "" {
			// log.Printf("parsing fallback body:\n===\n%s\n===\n", bodyStr)
			err := parseJSONBody(func() io.Reader { return strings.NewReader(bodyStr) })
			if err != nil {
				return err
			}
		}
	}

	pp := interpretPathParams(pathParams)

	for _, fm := range sm.NamedFields {
		switch fm.Source {
		case pathSrc:
			v := pp.Get(fm.name)
			if v == "" {
				if fm.Optional {
					continue
				}
				paramKeys := pp.Keys()
				panic(fmt.Errorf("httpform: missing path parameter %q (got %d path params: %s)", fm.name, len(paramKeys), strings.Join(paramKeys, ", ")))
			}
			err := setField(destVal, fm, v)
			if err != nil {
				return &Error{http.StatusBadRequest, "", err}
			}
		case headerSrc:
			v := r.Header.Get(fm.name)
			if v == "" {
				if fm.Optional {
					continue
				}
				return &Error{http.StatusBadRequest, fmt.Sprintf("missing header %s", fm.name), nil}
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

	for _, fm := range sm.UnnamedFields {
		var v any
		switch fm.Source {
		case requestSrc:
			v = r
		case urlSrc:
			v = r.URL
		case queryValuesSrc:
			v = r.URL.Query()
		case headersSrc:
			v = r.Header
		case methodSrc:
			v = r.Method
		case isSaveSrc:
			v = (r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch)
		case rawBodySrc:
			fv := destVal.Field(fm.fieldIdx)
			if isBytes(fv) {
				fv.Set(reflect.ValueOf(rawBody).Convert(fv.Type()))
			} else if isString(fv) {
				fv.Set(reflect.ValueOf(string(rawBody)).Convert(fv.Type()))
			} else {
				panic(fmt.Errorf("httpform: invalid type of rawbody param: %v", fv.Type()))
			}
			continue
		case fullBodySrc:
			v = fullBody
		default:
			continue
		}
		setFieldVal(destVal, fm, reflect.ValueOf(v))
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
	headerSrc
	requestSrc // sources here and below are unnamed
	urlSrc
	queryValuesSrc
	headersSrc
	methodSrc
	isSaveSrc
	rawBodySrc
	fullBodySrc
)

var _sources = []string{"none", "path", "form", "cookie", "header", "request", "url", "query values", "headers", "method", "issave", "rawbody", "fullbody"}

func (v source) String() string {
	return _sources[v]
}

func (v source) IsNamed() bool {
	return v < requestSrc
}
