package httpform

import (
	"encoding"
	"reflect"
	"strconv"
)

type ParserFunc func(s string) (reflect.Value, error)
type StringerFunc func(v reflect.Value) (string, error)

var textMarshaller = reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem()
var textUnmarshaller = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()

func pickParser(typ reflect.Type) ParserFunc {
	if typ.AssignableTo(textUnmarshaller) {
		return func(s string) (reflect.Value, error) {
			v := reflect.New(typ).Elem()
			err := v.Interface().(encoding.TextUnmarshaler).UnmarshalText([]byte(s))
			if err != nil {
				return reflect.Value{}, err
			}
			return v, nil
		}
	} else if reflect.PointerTo(typ).AssignableTo(textUnmarshaller) {
		return func(s string) (reflect.Value, error) {
			v := reflect.New(typ)
			err := v.Interface().(encoding.TextUnmarshaler).UnmarshalText([]byte(s))
			if err != nil {
				return reflect.Value{}, err
			}
			return v.Elem(), nil
		}
	}
	switch typ.Kind() {
	case reflect.String:
		return func(s string) (reflect.Value, error) {
			return reflect.ValueOf(s).Convert(typ), nil
		}
	case reflect.Bool:
		return func(s string) (reflect.Value, error) {
			v, err := ParseBool(s)
			if err != nil {
				return reflect.Value{}, err
			}
			return reflect.ValueOf(v).Convert(typ), nil
		}
	case reflect.Int:
		return func(s string) (reflect.Value, error) {
			v, err := strconv.ParseInt(s, 10, 0)
			if err != nil {
				return reflect.Value{}, err
			}
			return reflect.ValueOf(int(v)).Convert(typ), nil
		}
	case reflect.Int8:
		return func(s string) (reflect.Value, error) {
			v, err := strconv.ParseInt(s, 10, 8)
			if err != nil {
				return reflect.Value{}, err
			}
			return reflect.ValueOf(int8(v)).Convert(typ), nil
		}
	case reflect.Int16:
		return func(s string) (reflect.Value, error) {
			v, err := strconv.ParseInt(s, 10, 16)
			if err != nil {
				return reflect.Value{}, err
			}
			return reflect.ValueOf(int16(v)).Convert(typ), nil
		}
	case reflect.Int32:
		return func(s string) (reflect.Value, error) {
			v, err := strconv.ParseInt(s, 10, 32)
			if err != nil {
				return reflect.Value{}, err
			}
			return reflect.ValueOf(int32(v)).Convert(typ), nil
		}
	case reflect.Int64:
		return func(s string) (reflect.Value, error) {
			v, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				return reflect.Value{}, err
			}
			return reflect.ValueOf(int64(v)).Convert(typ), nil
		}
	case reflect.Uint:
		return func(s string) (reflect.Value, error) {
			v, err := strconv.ParseUint(s, 10, 0)
			if err != nil {
				return reflect.Value{}, err
			}
			return reflect.ValueOf(uint(v)).Convert(typ), nil
		}
	case reflect.Uintptr:
		return func(s string) (reflect.Value, error) {
			v, err := strconv.ParseUint(s, 10, 64)
			if err != nil {
				return reflect.Value{}, err
			}
			return reflect.ValueOf(uintptr(v)).Convert(typ), nil
		}
	case reflect.Uint8:
		return func(s string) (reflect.Value, error) {
			v, err := strconv.ParseUint(s, 10, 8)
			if err != nil {
				return reflect.Value{}, err
			}
			return reflect.ValueOf(uint8(v)).Convert(typ), nil
		}
	case reflect.Uint16:
		return func(s string) (reflect.Value, error) {
			v, err := strconv.ParseUint(s, 10, 16)
			if err != nil {
				return reflect.Value{}, err
			}
			return reflect.ValueOf(uint16(v)).Convert(typ), nil
		}
	case reflect.Uint32:
		return func(s string) (reflect.Value, error) {
			v, err := strconv.ParseUint(s, 10, 32)
			if err != nil {
				return reflect.Value{}, err
			}
			return reflect.ValueOf(uint32(v)).Convert(typ), nil
		}
	case reflect.Uint64:
		return func(s string) (reflect.Value, error) {
			v, err := strconv.ParseUint(s, 10, 64)
			if err != nil {
				return reflect.Value{}, err
			}
			return reflect.ValueOf(int64(v)).Convert(typ), nil
		}
	case reflect.Float32:
		return func(s string) (reflect.Value, error) {
			v, err := strconv.ParseFloat(s, 32)
			if err != nil {
				return reflect.Value{}, err
			}
			return reflect.ValueOf(float32(v)).Convert(typ), nil
		}
	case reflect.Float64:
		return func(s string) (reflect.Value, error) {
			v, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return reflect.Value{}, err
			}
			return reflect.ValueOf((v)).Convert(typ), nil
		}
	default:
		return nil
	}
}

func pickStringer(typ reflect.Type) StringerFunc {
	if typ.AssignableTo(textMarshaller) {
		return func(v reflect.Value) (string, error) {
			raw, err := v.Interface().(encoding.TextMarshaler).MarshalText()
			if err != nil {
				return "", err
			}
			return string(raw), nil
		}
	} else if reflect.PointerTo(typ).AssignableTo(textMarshaller) {
		return func(v reflect.Value) (string, error) {
			ptrVal := v.Addr()
			raw, err := ptrVal.Interface().(encoding.TextMarshaler).MarshalText()
			if err != nil {
				return "", err
			}
			return string(raw), nil
		}
	}
	switch typ.Kind() {
	case reflect.String:
		return func(v reflect.Value) (string, error) {
			return v.String(), nil
		}
	case reflect.Bool:
		return func(v reflect.Value) (string, error) {
			return strconv.FormatBool(v.Bool()), nil
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return func(v reflect.Value) (string, error) {
			return strconv.FormatInt(v.Int(), 10), nil
		}
	case reflect.Uint, reflect.Uintptr, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return func(v reflect.Value) (string, error) {
			return strconv.FormatUint(v.Uint(), 10), nil
		}
	case reflect.Float32:
		return func(v reflect.Value) (string, error) {
			return strconv.FormatFloat(v.Float(), 'g', -1, 32), nil
		}
	case reflect.Float64:
		return func(v reflect.Value) (string, error) {
			return strconv.FormatFloat(v.Float(), 'g', -1, 64), nil
		}
	default:
		return nil
	}
}
