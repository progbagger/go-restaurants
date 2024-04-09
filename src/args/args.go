package args

import (
	"flag"
	"fmt"
	"reflect"
)

type Arg struct {
	Name         string // argument name
	Description  string // argument description for --help option
	DefaultValue any    // default value for this argument
	Required     bool   // indicates if argument must be present in arguments list
}

func ParseArgs(args ...Arg) (map[string]any, []string, error) {
	// creating flags
	result := make(map[string]any, len(args))
	for _, arg := range args {
		parsedFlag, err := parseArg(&arg)

		if err != nil {
			return nil, nil, err
		}

		result[arg.Name] = parsedFlag
	}

	flag.Parse()
	if err := checkRequiredFlags(args...); err != nil {
		return nil, nil, err
	}

	// replacing pointers with values
	for key, value := range result {
		result[key] = reflect.ValueOf(value).Elem().Interface()
	}

	return result, flag.Args(), nil
}

func parseArg(arg *Arg) (any, error) {
	switch arg.DefaultValue.(type) {
	case int:
		return flag.Int(arg.Name, arg.DefaultValue.(int), arg.Description), nil
	case int64:
		return flag.Int64(arg.Name, arg.DefaultValue.(int64), arg.Description), nil
	case float64:
		return flag.Float64(arg.Name, arg.DefaultValue.(float64), arg.Description), nil
	case bool:
		return flag.Bool(arg.Name, arg.DefaultValue.(bool), arg.Description), nil
	case string:
		return flag.String(arg.Name, arg.DefaultValue.(string), arg.Description), nil
	default:
		return nil, fmt.Errorf("unknown argument type")
	}
}

func checkRequiredFlags(args ...Arg) error {
	requiredFlags := make(map[string]bool)
	for _, arg := range args {
		if arg.Required {
			requiredFlags[arg.Name] = false
		}
	}

	flag.Visit(func(f *flag.Flag) {
		if _, exists := requiredFlags[f.Name]; exists {
			requiredFlags[f.Name] = true
		}
	})

	for name, present := range requiredFlags {
		if !present {
			return fmt.Errorf("flag \"%s\" is not present but required", name)
		}
	}

	return nil
}
