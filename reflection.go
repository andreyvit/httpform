package httpform

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
)

var (
	requestType   = reflect.TypeOf((*http.Request)(nil))
	urlType       = reflect.TypeOf((*url.URL)(nil))
	urlValuesType = reflect.TypeOf((url.Values)(nil))
	headersType   = reflect.TypeOf((http.Header)(nil))
)

type structMeta struct {
	NamedFields   map[string]*fieldMeta
	UnnamedFields []*fieldMeta
	HasRawBody    bool
	HasFullBody   bool
	HasBodyForm   bool
}

type specialMeta struct {
	Index  int
	Source source
}

type fieldMeta struct {
	fieldIdx  int
	name      string
	Parse     ParserFunc
	Stringify StringerFunc
	Source    source
	Optional  bool
	NotInBody bool
}

func getVal(structVal reflect.Value, fm *fieldMeta) reflect.Value {
	return structVal.Field(fm.fieldIdx)
}

func getString(structVal reflect.Value, fm *fieldMeta) string {
	v := getVal(structVal, fm)

	s, err := fm.Stringify(v)
	if err != nil {
		panic(fmt.Errorf("failed to encode value of %s: %v", fm.name, v.Interface()))
	}
	return s
}

func setVal(structVal reflect.Value, sm *structMeta, src source, name string, rawValue string) error {
	fm := sm.NamedFields[name]
	if fm == nil || fm.Source != src {
		if src != formSrc {
			panic(fmt.Errorf("no input field for %v param %q", src, name))
		}
		return nil
	}
	return setField(structVal, fm, rawValue)
}

func setField(structVal reflect.Value, fm *fieldMeta, rawValue string) error {
	value, err := fm.Parse(rawValue)
	if err != nil {
		return fmt.Errorf("invalid %s: %w", fm.name, err)
	}
	setFieldVal(structVal, fm, value)
	return nil
}

func setFieldVal(structVal reflect.Value, fm *fieldMeta, val reflect.Value) {
	fieldVal := structVal.Field(fm.fieldIdx)
	fieldTyp := fieldVal.Type()
	if !val.IsValid() {
		val = reflect.Zero(fieldTyp)
	}
	if !val.CanConvert(fieldTyp) {
		panic(fmt.Errorf("%s: cannot convert from %s to %s", fm.name, val.Type(), fieldTyp))
	}
	fieldVal.Set(val.Convert(fieldTyp))
}

func (conf *Configuration) lookupStruct(structTyp reflect.Type) *structMeta {
	v, _ := conf.structCache.Load(structTyp)
	if v != nil {
		return v.(*structMeta)
	}

	sm := conf.examineStruct(structTyp)
	conf.structCache.LoadOrStore(structTyp, sm)
	return sm
}

func (conf *Configuration) examineStruct(structTyp reflect.Type) *structMeta {
	n := structTyp.NumField()
	sm := &structMeta{
		NamedFields: make(map[string]*fieldMeta),
	}
	for i := 0; i < n; i++ {
		field := structTyp.Field(i)
		fm := conf.examineField(i, &field, structTyp)
		if fm != nil {
			if fm.Source.IsNamed() {
				sm.NamedFields[fm.name] = fm
			} else {
				sm.UnnamedFields = append(sm.UnnamedFields, fm)
			}
			if fm.Source == rawBodySrc {
				sm.HasRawBody = true
			} else if fm.Source == fullBodySrc {
				sm.HasFullBody = true
			} else if fm.Source == formSrc && !fm.NotInBody {
				sm.HasBodyForm = true
			}
		}
	}
	return sm
}

func (conf *Configuration) examineField(fieldIdx int, field *reflect.StructField, structTyp reflect.Type) *fieldMeta {
	if !field.IsExported() {
		return nil
	}
	fieldTyp := field.Type

	src := noSrc
	if fieldTyp == requestType {
		src = requestSrc
	} else if fieldTyp == urlType {
		src = urlSrc
	} else if fieldTyp == urlValuesType {
		src = queryValuesSrc
	} else if fieldTyp == headersType {
		src = headersSrc
	}

	jsonTag, jsonPresent := field.Tag.Lookup("json")
	var (
		jsonName    string = field.Name
		jsonNamed   bool
		jsonSkipped bool
	)
	if jsonPresent {
		comps := strings.Split(jsonTag, ",")
		if n := comps[0]; n != "" {
			jsonName = comps[0]
			jsonNamed = true
			jsonSkipped = (n == "-")
		}
	}

	formTag, formPresent := field.Tag.Lookup("form")
	var (
		formName    string
		isOptional  bool
		isNotInBody bool
	)
	if formPresent {
		comps := strings.Split(formTag, ",")
		if n := comps[0]; n != "" {
			if n == "-" {
				return nil
			}
			formName = comps[0]
		}

		for _, mod := range comps[1:] {
			switch mod {
			case "path":
				if src != noSrc {
					panic(fmt.Errorf(`field %v.%s has conflicting modifier %q in form:%q tag`, structTyp, field.Name, mod, formTag))
				}
				src = pathSrc
			case "cookie":
				if src != noSrc {
					panic(fmt.Errorf(`field %v.%s has conflicting modifier %q in form:%q tag`, structTyp, field.Name, mod, formTag))
				}
				src = cookieSrc
			case "header":
				if src != noSrc {
					panic(fmt.Errorf(`field %v.%s has conflicting modifier %q in form:%q tag`, structTyp, field.Name, mod, formTag))
				}
				src = headerSrc
			case "method":
				if src != noSrc {
					panic(fmt.Errorf(`field %v.%s has conflicting modifier %q in form:%q tag`, structTyp, field.Name, mod, formTag))
				}
				src = methodSrc
			case "issave":
				if src != noSrc {
					panic(fmt.Errorf(`field %v.%s has conflicting modifier %q in form:%q tag`, structTyp, field.Name, mod, formTag))
				}
				src = isSaveSrc
			case "rawbody":
				if src != noSrc {
					panic(fmt.Errorf(`field %v.%s has conflicting modifier %q in form:%q tag`, structTyp, field.Name, mod, formTag))
				}
				src = rawBodySrc
			case "fullbody":
				if src != noSrc {
					panic(fmt.Errorf(`field %v.%s has conflicting modifier %q in form:%q tag`, structTyp, field.Name, mod, formTag))
				}
				src = fullBodySrc
			case "notinbody":
				isNotInBody = true
			case "optional":
				isOptional = true
			default:
				panic(fmt.Errorf(`field %v.%s has unknown modifier %q in form:%q tag`, structTyp, field.Name, mod, formTag))
			}
		}
	}
	if src == noSrc {
		src = formSrc
	}

	if src.IsNamed() && !formPresent && !jsonPresent {
		panic(fmt.Errorf(`field %v.%s must have form:"..." or json:"..." tag; use json:"-" to skip`, structTyp, field.Name))
	}

	if conf.AllowJSON && src != formSrc && !jsonSkipped {
		panic(fmt.Errorf(`field %v.%s is sourced from %v and must have json:"-" tag to disallow populating it from a JSON body`, structTyp, field.Name, src))
	}

	if !src.IsNamed() {
		if formName != "" {
			panic(fmt.Errorf(`field %v.%s is sourced from %v and cannot have a name in form:%q tag`, structTyp, field.Name, src, formTag))
		}
		return &fieldMeta{
			fieldIdx: fieldIdx,
			Source:   src,
		}
	}

	var name string
	if src == formSrc && conf.AllowJSON {
		if !jsonNamed {
			panic(fmt.Errorf(`field %v.%s must have json:"somename" tag`, structTyp, field.Name))
		}
		name = jsonName
		if formName != "" && formName != name {
			panic(fmt.Errorf(`field %v.%s has unnecessary name in form:%q tag that doesn't match the name in json:%q tag, recommended: drop the name in the form tag`, structTyp, field.Name, formTag, jsonTag))
		}
	} else {
		if formName == "" {
			panic(fmt.Errorf(`field %v.%s must have form:"somename" tag`, structTyp, field.Name))
		}
		name = formName
	}

	fm := &fieldMeta{
		fieldIdx:  fieldIdx,
		name:      name,
		Parse:     pickParser(fieldTyp),
		Stringify: pickStringer(fieldTyp),
		Source:    src,
		Optional:  isOptional,
		NotInBody: isNotInBody,
	}
	if fm.Parse == nil {
		panic(fmt.Errorf("field %v.%v: don't know how to parse %v from a string", structTyp, field.Name, fieldTyp))
	}
	if fm.Stringify == nil {
		panic(fmt.Errorf("field %v.%v: don't know how to convert %v to a string", structTyp, field.Name, fieldTyp))
	}
	return fm
}

func isBytes(v reflect.Value) bool {
	return v.Kind() == reflect.Slice && v.Type().Elem().Kind() == reflect.Uint8
}

func isString(v reflect.Value) bool {
	return v.Kind() == reflect.String
}
