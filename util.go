package httpform

import (
	"fmt"
	"mime"
	"net/http"
)

const (
	formContentType          = "application/x-www-form-urlencoded"
	multipartFormContentType = "multipart/form-data"
	jsonContentType          = "application/json"
)

func determineMIMEType(r *http.Request) string {
	s := r.Header.Get("Content-Type")
	if s == "" {
		return ""
	}
	ctype, _, err := mime.ParseMediaType(s)
	if ctype == "" || err != nil {
		return ""
	}
	return ctype
}

func LimitBody(w http.ResponseWriter, r *http.Request, maxSize int64) {
	if r.Body != nil {
		r.Body = http.MaxBytesReader(w, r.Body, maxSize)
	}
}

// ParseBool is similar to strconv.ParseBool, but recognizes on/off values that browsers send for checkboxes, and yes/no.
func ParseBool(str string) (bool, error) {
	switch str {
	case "1", "t", "T", "y", "Y", "true", "TRUE", "True", "on", "ON", "On", "yes", "YES", "Yes":
		return true, nil
	case "0", "f", "F", "n", "N", "false", "FALSE", "False", "off", "OFF", "Off", "no", "NO", "No":
		return false, nil
	}
	return false, fmt.Errorf("invalid bool value %q", str)
}

func parseBoolDefault(str string, dflt bool) bool {
	v, err := ParseBool(str)
	if err == nil {
		return v
	} else {
		return dflt
	}
}

func filterInPlace[S ~[]T, T any](slice S, filter func(item T) (T, bool)) S {
	o := 0
	for i, item := range slice {
		item, add := filter(item)
		if add {
			if o != i {
				slice[o] = item
			}
			o++
		}
	}
	return slice[:o]
}
