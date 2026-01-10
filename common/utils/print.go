package utils

import (
	"encoding/json"
	"fmt"
	"reflect"
)

func PrintObj(v interface{}) {
	t := reflect.TypeOf(v)
	name := ""
	if t != nil {
		// If it's a pointer, get the underlying element
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}

		// Check if it is a struct and print its name
		if t.Kind() == reflect.Struct {
			name = t.Name()
		}
	}

	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Println(err)
		return
	}

	if name != "" {
		fmt.Println(name, ": ", string(b))
		return
	}

	fmt.Println(string(b))
}
