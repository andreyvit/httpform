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

// ParseBool is similar to strconv.ParseBool, but recognizes on/off values that browsers send for checkboxes.
func ParseBool(str string) (bool, error) {
	switch str {
	case "1", "t", "T", "true", "TRUE", "True", "on", "ON", "On":
		return true, nil
	case "0", "f", "F", "false", "FALSE", "False", "off", "OFF", "Off":
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
