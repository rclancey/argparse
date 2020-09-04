package argparse

import (
	"reflect"
	"testing"
	"time"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }
type ArgParseSuite struct {}
var _ = Suite(&ArgParseSuite{})

type SubArgs struct {
	Bool bool `arg:"bool"`
	String string `arg:"string"`
	Int int
	Uint16s []uint16 `arg:"uints"`
	Float32s []float32 `arg:"f32s"`
}

type Args struct {
	Int8 int8 `arg:"i8"`
	Int16 int16 `arg:"i16"`
	Int32 int32 `arg:"i32"`
	Int64 int64 `arg:"i64"`
	Uint uint
	Uint8 uint8 `arg:"u8"`
	Uint16 uint16 `arg:"u16"`
	Uint32 uint32 `arg:"u32"`
	Uint64 uint64 `arg:"u64"`
	Float32 float32 `arg:"f32"`
	Float64 float64 `arg:"f"`
	String string `arg:"string"`
	Strings []string
	Ints []int
	Sub SubArgs `arg:"sub"`
	Ignored string `arg:"-"`
	private string `arg:"private"`
	Map map[string]string `arg:"map"`
	Subs []SubArgs `arg:"subs"`
	Times []time.Time `arg:"times"`
	Time time.Time `arg:"time"`
}

type mapHasChecker struct {
	*CheckerInfo
}

func (ch *mapHasChecker) Check(params []interface{}, names []string) (bool, string) {
	if len(params) != 2 {
		return false, "incorrect args"
	}
	rv := reflect.ValueOf(params[0])
	if rv.Kind() != reflect.Map {
		return false, "first arg is not a map"
	}
	kv := reflect.ValueOf(params[1])
	if kv.Type() != rv.Type().Key() {
		return false, "key is wrong type"
	}
	iter := rv.MapRange()
	for iter.Next() {
		if iter.Key().Interface() == kv.Interface() {
			return true, ""
		}
	}
	return false, ""
}

var MapHas = &mapHasChecker{
	&CheckerInfo{Name: "MapHas", Params: []string{"map", "key"}},
}

func (a *ArgParseSuite) TestMakeArgMap(c *C) {
	args := &Args{}
	rv := reflect.ValueOf(args).Elem()
	m := makeArgMap(rv)
	keys := []string{}
	for k := range m {
		keys = append(keys, k)
	}
	c.Log("keys = ", keys)
	c.Check(m, MapHas, "sub-bool")
}

func (a *ArgParseSuite) TestParseArgs(c *C) {
	loc, err := time.LoadLocation("America/Los_Angeles")
	c.Assert(err, IsNil)
	orig := time.Local
	time.Local = loc
	defer func() { time.Local = orig }()
	cmd := []string{
		"--i8=-1",
		"--i16",
		"0x2",
		"-i32=-3",
		"-i64",
		"-4",
		"-uint=5",
		"--u8",
		"6",
		"-u16",
		"7",
		"-u32",
		"0xf",
		"--u64=9",
		"--f32",
		"-1.23",
		"-f=10",
		"-string",
		"blah",
		"--strings",
		"a",
		"b",
		"cdef",
		"-ints",
		"-10",
		"11",
		"100",
		"0xab",
		"-sub-bool",
		"-sub-string=foo",
		"-sub-int",
		"-25",
		"-sub-uints",
		"21",
		"0x22",
		"-sub-f32s=1.23,4.5,6",
		"-time=2019-10-30 19:25:36.765-07:00",
		"-times",
		"10/30/19",
		"Wed, 30 Oct 2019 19:25:36 PDT",
	}
	args := &Args{}
	err = parseArgs(args, cmd)
	c.Check(err, IsNil)
	c.Check(args.Int8, Equals, int8(-1))
	c.Check(args.Int16, Equals, int16(2))
	c.Check(args.Int32, Equals, int32(-3))
	c.Check(args.Int64, Equals, int64(-4))
	c.Check(args.Uint, Equals, uint(5))
	c.Check(args.Uint8, Equals, uint8(6))
	c.Check(args.Uint16, Equals, uint16(7))
	c.Check(args.Uint32, Equals, uint32(15))
	c.Check(args.Uint64, Equals, uint64(9))
	c.Check(args.Float32, Equals, float32(-1.23))
	c.Check(args.Float64, Equals, float64(10.0))
	c.Check(args.String, Equals, "blah")
	c.Check(len(args.Strings), Equals, 3)
	c.Check(args.Strings[0], Equals, "a")
	c.Check(args.Strings[1], Equals, "b")
	c.Check(args.Strings[2], Equals, "cdef")
	c.Check(len(args.Ints), Equals, 4)
	c.Check(args.Ints[0], Equals, int(-10))
	c.Check(args.Ints[1], Equals, int(11))
	c.Check(args.Ints[2], Equals, int(100))
	c.Check(args.Ints[3], Equals, int(171))
	c.Check(args.Sub.Bool, Equals, true)
	c.Check(args.Sub.String, Equals, "foo")
	c.Check(args.Sub.Int, Equals, int(-25))
	c.Check(len(args.Sub.Uint16s), Equals, 2)
	c.Check(args.Sub.Uint16s[0], Equals, uint16(21))
	c.Check(args.Sub.Uint16s[1], Equals, uint16(34))
	c.Check(len(args.Sub.Float32s), Equals, 3)
	c.Check(args.Sub.Float32s[0], Equals, float32(1.23))
	c.Check(args.Sub.Float32s[1], Equals, float32(4.5))
	c.Check(args.Sub.Float32s[2], Equals, float32(6.0))
	c.Check(args.Time.Unix(), Equals, int64(1572488736))
	c.Check(args.Time.Nanosecond(), Equals, int(765000000))
	c.Check(len(args.Times), Equals, 2)
	c.Check(args.Times[0].Unix(), Equals, int64(1572418800))
	c.Check(args.Times[1].Unix(), Equals, int64(1572488736))
}

func (a *ArgParseSuite) TestParseArgsError(c *C) {
	cmd := []string{
		"--junk=10",
	}
	args := &Args{}
	err := parseArgs(args, cmd)
	c.Check(err, ErrorMatches, "^.*Unknown arg '--junk'.*$")
	cmd = []string{
		"--ignored=what",
	}
	args = &Args{}
	err = parseArgs(args, cmd)
	c.Check(err, ErrorMatches, "^.*Unknown arg '--ignored'.*$")
	cmd = []string{
		"--private=eyes",
	}
	args = &Args{}
	err = parseArgs(args, cmd)
	c.Check(err, ErrorMatches, "^.*Unknown arg '--private'.*$")
	cmd = []string{
		"--i8=junk",
	}
	args = &Args{}
	err = parseArgs(args, cmd)
	c.Check(err, ErrorMatches, `^.*error parsing arg --i8 \(junk\) into int8.*$`)
	cmd = []string{
		"-u32=-10",
	}
	args = &Args{}
	err = parseArgs(args, cmd)
	c.Check(err, ErrorMatches, `^.*error parsing arg -u32 \(-10\) into uint32.*$`)
	cmd = []string{
		"--f",
		"0x32",
	}
	args = &Args{}
	err = parseArgs(args, cmd)
	c.Check(err, ErrorMatches, `^.*error parsing arg --f \(0x32\) into float64.*$`)
	cmd = []string{
		"--map=k:v,foo:bar",
	}
	args = &Args{}
	err = parseArgs(args, cmd)
	c.Check(err, ErrorMatches, `.*don't know how to parse into map\[string\]string.*$`)
	cmd = []string{
		"--subs",
		"abc",
	}
	args = &Args{}
	err = parseArgs(args, cmd)
	c.Check(err, ErrorMatches, `.*don't know how to parse into argparse.SubArgs.*$`)
	cmd = []string{
		"--time",
		"2019-10-30T19:43:21",
	}
	args = &Args{}
	err = parseArgs(args, cmd)
	c.Check(err, ErrorMatches, `^.*can't parse .* as a time stamp.*$`)
	cmd = []string{
		"-ints=0xnan",
	}
	args = &Args{}
	err = parseArgs(args, cmd)
	c.Check(err, ErrorMatches, `^.*error parsing arg -ints \(0xnan\) into \[\]int.*$`)
	cmd = []string{
		"--sub-uints=10,15,-20",
	}
	args = &Args{}
	err = parseArgs(args, cmd)
	c.Check(err, ErrorMatches, `^.*error parsing arg --sub-uints \(10 15 -20\) into \[\]uint16.*$`)
	cmd = []string{
		"--sub-f32s=1.23,infin,4.56",
	}
	args = &Args{}
	err = parseArgs(args, cmd)
	c.Check(err, ErrorMatches, `^.*error parsing arg --sub-f32s \(1.23 infin 4.56\) into \[\]float32.*$`)
}
