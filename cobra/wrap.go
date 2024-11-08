package cobra

import (
	"github.com/iocgo/sdk/stream"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/tidwall/gjson"
	"reflect"
	"strings"
)

type Command = cobra.Command

type ICobra interface {
	Command() *cobra.Command
}

type singleCobra struct {
	cmd *cobra.Command
}

func (c *singleCobra) Command() *cobra.Command {
	return c.cmd
}

var (
	kindSetters = map[reflect.Kind]func(*pflag.FlagSet, reflect.Value, string, string, string){
		reflect.Int: func(flags *pflag.FlagSet, value reflect.Value, field, short, usage string) {
			warp[int](value, elseOf(short, flags.IntVar, flags.IntVarP), field, short, usage, int(value.Int()))
		},
		reflect.Int8: func(flags *pflag.FlagSet, value reflect.Value, field, short, usage string) {
			warp[int8](value, elseOf(short, flags.Int8Var, flags.Int8VarP), field, short, usage, int8(value.Int()))
		},
		reflect.Int16: func(flags *pflag.FlagSet, value reflect.Value, field, short, usage string) {
			warp[int16](value, elseOf(short, flags.Int16Var, flags.Int16VarP), field, short, usage, int16(value.Int()))
		},
		reflect.Int32: func(flags *pflag.FlagSet, value reflect.Value, field, short, usage string) {
			warp[int32](value, elseOf(short, flags.Int32Var, flags.Int32VarP), field, short, usage, int32(value.Int()))
		},
		reflect.Int64: func(flags *pflag.FlagSet, value reflect.Value, field, short, usage string) {
			warp[int64](value, elseOf(short, flags.Int64Var, flags.Int64VarP), field, short, usage, value.Int())
		},
		reflect.Uint: func(flags *pflag.FlagSet, value reflect.Value, field, short, usage string) {
			warp[uint](value, elseOf(short, flags.UintVar, flags.UintVarP), field, short, usage, uint(value.Uint()))
		},
		reflect.Uint8: func(flags *pflag.FlagSet, value reflect.Value, field, short, usage string) {
			warp[uint8](value, elseOf(short, flags.Uint8Var, flags.Uint8VarP), field, short, usage, uint8(value.Uint()))
		},
		reflect.Uint16: func(flags *pflag.FlagSet, value reflect.Value, field, short, usage string) {
			warp[uint16](value, elseOf(short, flags.Uint16Var, flags.Uint16VarP), field, short, usage, uint16(value.Uint()))
		},
		reflect.Uint32: func(flags *pflag.FlagSet, value reflect.Value, field, short, usage string) {
			warp[uint32](value, elseOf(short, flags.Uint32Var, flags.Uint32VarP), field, short, usage, uint32(value.Uint()))
		},
		reflect.Uint64: func(flags *pflag.FlagSet, value reflect.Value, field, short, usage string) {
			warp[uint64](value, elseOf(short, flags.Uint64Var, flags.Uint64VarP), field, short, usage, value.Uint())
		},
		reflect.Float32: func(flags *pflag.FlagSet, value reflect.Value, field, short, usage string) {
			warp[float32](value, elseOf(short, flags.Float32Var, flags.Float32VarP), field, short, usage, float32(value.Float()))
		},
		reflect.Float64: func(flags *pflag.FlagSet, value reflect.Value, field, short, usage string) {
			warp[float64](value, elseOf(short, flags.Float64Var, flags.Float64VarP), field, short, usage, value.Float())
		},
		reflect.String: func(flags *pflag.FlagSet, value reflect.Value, field, short, usage string) {
			warp[string](value, elseOf(short, flags.StringVar, flags.StringVarP), field, short, usage, value.String())
		},
		reflect.Bool: func(flags *pflag.FlagSet, value reflect.Value, field, short, usage string) {
			warp[bool](value, elseOf(short, flags.BoolVar, flags.BoolVarP), field, short, usage, value.Bool())
		},
	}
)

func elseOf(str string, a1, a2 interface{}) interface{} {
	if strings.TrimSpace(str) == "" {
		return a1
	} else {
		return a2
	}
}

func warp[T any](value reflect.Value, f interface{}, field, short, usage string, def T) {
	if !value.CanSet() {
		return
	}
	exec := reflect.ValueOf(f)
	values := []reflect.Value{value.Addr(), reflect.ValueOf(field)}

	if short != "" {
		values = append(values, reflect.ValueOf(short))
	}

	values = append(values, reflect.ValueOf(def), reflect.ValueOf(usage))
	exec.Call(values)
}

func ICobraWrapper(instance interface{}, config string, children ...ICobra) (c ICobra) {
	cmd := &cobra.Command{}
	c = &singleCobra{cmd}
	for _, it := range children {
		cmd.AddCommand(it.Command())
	}

	parser := gjson.Parse(config)
	bindField(parser, "Use", func(value string) { cmd.Use = value })
	bindField(parser, "Short", func(value string) { cmd.Short = value })
	bindField(parser, "Long", func(value string) { cmd.Long = value })
	bindField(parser, "Version", func(value string) { cmd.Version = value })
	bindField(parser, "Example", func(value string) { cmd.Example = value })

	value := reflect.ValueOf(instance)
	bindMethod(parser, value, "Run", func(value func(*cobra.Command, []string)) { cmd.Run = value })

	bindTag(cmd, value)
	return
}

func bindField(parser gjson.Result, field string, f func(string)) {
	if result := parser.Get(field); result.Exists() {
		if field = result.String(); field != "" {
			f(field)
		}
	}
}

func bindMethod(parser gjson.Result, value reflect.Value, field string, f func(func(*cobra.Command, []string))) {
	if result := parser.Get(field); result.Exists() {
		if field = result.String(); field == "" {
			return
		}

		method := value.MethodByName(result.String())
		if !method.IsValid() {
			panic("`" + result.String() + "` method is not exist")
		}
		f(func(cmd *cobra.Command, args []string) {
			method.Call([]reflect.Value{reflect.ValueOf(cmd), reflect.ValueOf(args)})
		})
	}
}

func bindTag(cmd *cobra.Command, value reflect.Value) {
	flags := cmd.Flags()
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}
	for i := range value.NumField() {
		lookup, ok := value.Type().Field(i).Tag.Lookup("cobra")
		if !ok || lookup == "" {
			continue
		}

		pair, ok := sliceToPair(strings.Split(lookup, ","))
		if !ok || pair.Val1 == "" {
			continue
		}

		short, _ := value.Type().Field(i).Tag.Lookup("short")
		usage, _ := value.Type().Field(i).Tag.Lookup("usage")

		if pair.Val2 == "per" {
			flags = cmd.PersistentFlags()
		}

		setter(flags, value.Field(i), pair.Val1, short, usage)
	}
}

func sliceToPair(slice []string) (pair stream.Pair[string, string], ok bool) {
	if len(slice) == 0 {
		return
	}
	if len(slice) == 1 {
		slice = append(slice, "")
	}
	return stream.Pair[string, string]{Val1: strings.TrimSpace(slice[0]), Val2: strings.TrimSpace(slice[1])}, true
}

func setter(flags *pflag.FlagSet, value reflect.Value, field, short, usage string) {
	if exec, ok := kindSetters[value.Kind()]; ok {
		exec(flags, value, field, short, usage)
	}
}
