package argparse

import (
	"fmt"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

func parseTime(v string) (time.Time, error) {
	layouts := []string{
		"2006-01-02T15:04:05Z0700",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05Z0700",
		"2006-01-02 15:04:05Z07:00",
		"2006-01-02 15:04:05 -0700",
		"2006-01-02 15:04:05 -07:00",
		"2006-01-02 15:04:05 MST",
		"2006-01-02 15:04:05",
		"2006-01-02",
		time.UnixDate,
		time.RFC1123,
		time.RFC1123Z,
		"1/2/2006 15:04:05 -0700",
		"1/2/2006 15:04:05 -07:00",
		"1/2/2006 15:04:05 MST",
		"1/2/2006 3:04:05 PM",
		"1/2/2006 3:04:05 PM",
		"1/2/2006 3:04:05 PM",
		"1/2/2006",
		"1/2/06",
	}
	for _, layout := range layouts {
		t, err := time.ParseInLocation(layout, v, time.Local)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, errors.Errorf("can't parse %s as a time stamp", v)
}

func parseInt(v string) (int64, error) {
	base := 10
	if strings.HasPrefix(v, "0x") {
		base = 16
		v = v[2:]
	}
	return strconv.ParseInt(v, base, 64)
}

func parseUint(v string) (uint64, error) {
	base := 10
	if strings.HasPrefix(v, "0x") {
		base = 16
		v = v[2:]
	}
	return strconv.ParseUint(v, base, 64)
}

var timeType = reflect.TypeOf(time.Time{})

func parseInto(rv reflect.Value, vals ...string) error {
	log.Printf("parse %s into %T", vals, rv.Interface())
	if rv.Kind() != reflect.Slice && len(vals) != 1 {
		return errors.Errorf("can't set a scalar to %d values", len(vals))
	}
	switch rv.Kind() {
	case reflect.String:
		rv.SetString(vals[0])
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uv, err := parseUint(vals[0])
		if err != nil {
			return errors.Wrap(err, "error parsing uint")
		}
		rv.SetUint(uv)
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		iv, err := parseInt(vals[0])
		if err != nil {
			return errors.Wrap(err, "error parsing int")
		}
		rv.SetInt(iv)
		return nil
	case reflect.Float32, reflect.Float64:
		fv, err := strconv.ParseFloat(vals[0], 64)
		if err != nil {
			return errors.Wrap(err, "error parsing float")
		}
		rv.SetFloat(fv)
		return nil
	case reflect.Slice:
		s := reflect.MakeSlice(reflect.SliceOf(rv.Type().Elem()), len(vals), len(vals))
		for i, sv := range vals {
			err := parseInto(s.Index(i), sv)
			if err != nil {
				return errors.Wrapf(err, "error parsing list element %d", i)
			}
		}
		rv.Set(s)
		log.Println("set slice", s.Interface())
		return nil
	default:
		if rv.Type() == timeType {
			t, err := parseTime(vals[0])
			if err != nil {
				return errors.Wrap(err, "error parsing time")
			}
			rv.Set(reflect.ValueOf(t))
			return nil
		} else {
			return errors.Errorf("don't know how to parse into %T", rv.Interface())
		}
	}
	return nil
}

func makeArgMap(rv reflect.Value) map[string]reflect.Value {
	m := map[string]reflect.Value{}
	rt := rv.Type()
	n := rt.NumField()
	for i := 0; i < n; i++ {
		rf := rt.Field(i)
		if rf.PkgPath != "" {
			continue
		}
		tag := rf.Tag.Get("arg")
		if tag == "-" {
			continue
		}
		if tag == "" {
			tag = strings.ToLower(rf.Name)
		} else {
			tag = strings.Trim(tag, "-")
		}
		if rf.Type == timeType {
			m[tag] = rv.Field(i)
		} else if rf.Type.Kind() == reflect.Struct {
			xm := makeArgMap(rv.Field(i))
			if rf.Anonymous {
				for k, v := range xm {
					m[k] = v
				}
			} else {
				for k, v := range xm {
					m[tag+"-"+k] = v
				}
			}
		} else if rf.Type.Kind() == reflect.Ptr && rf.Type.Elem().Kind() == reflect.Struct {
			var xv reflect.Value
			if rv.Field(i).IsNil() {
				xv = reflect.New(rf.Type.Elem()).Elem()
			} else {
				xv = rv.Field(i).Elem()
			}
			xm := makeArgMap(xv)
			if rf.Anonymous {
				for k, v := range xm {
					m[k] = v
				}
			} else {
				for k, v := range xm {
					m[tag+"-"+k] = v
				}
			}
		} else {
			m[tag] = rv.Field(i)
		}
	}
	return m
}

func ParseArgs(recv interface{}) error {
	return parseArgs(recv, os.Args[1:])
}

func parseArgs(recv interface{}, args []string) error {
	rv := reflect.ValueOf(recv).Elem()
	m := makeArgMap(rv)
	for k := range m {
		fmt.Println(k)
	}
	n := len(args)
	i := 0
	for i < n {
		parts := strings.SplitN(args[i], "=", 2)
		flag := strings.Trim(parts[0], "-")
		rf, ok := m[flag]
		if !ok {
			return errors.Errorf("Unknown arg '%s'", parts[0])
		}
		i += 1
		switch rf.Kind() {
		case reflect.Bool:
			rf.SetBool(true)
		case reflect.Slice:
			vals := []string{}
			if len(parts) == 2 {
				vals = strings.Split(parts[1], ",")
			}
			for i < n {
				if strings.HasPrefix(args[i], "-") {
					xparts := strings.SplitN(args[i], "=", 2)
					xflag := strings.Trim(xparts[0], "-")
					if _, ok := m[xflag]; ok {
						break
					}
				}
				vals = append(vals, args[i])
				i += 1
			}
			err := parseInto(rf, vals...)
			if err != nil {
				return errors.Wrapf(err, "error parsing arg %s (%s) into %T", parts[0], strings.Join(vals, " "), rf.Interface())
			}
		default:
			var val string
			if len(parts) == 2 {
				val = parts[1]
			} else if i < n {
				val = args[i]
				i += 1
			}
			err := parseInto(rf, val)
			if err != nil {
				return errors.Wrapf(err, "error parsing arg %s (%s) into %T", parts[0], val, rf.Interface())
			}
		}
	}
	return nil
}
