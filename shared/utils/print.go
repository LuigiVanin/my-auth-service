package utils

import (
	"encoding/json"
	"fmt"
	"reflect"

	"auth_service/shared/constants"
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

// PrintHeader, PrintSection, PrintSuccess, PrintWarning, PrintError, PrintKeyValue
// and PrintHint format the terminal output of the CLI entrypoints (seeds, migrations
// and helpers) so every command speaks the same visual language.

func PrintHeader(text string) {
	fmt.Printf("%s%s=== %s ===%s\n", constants.ColorCyan, constants.Bold, text, constants.ColorReset)
}

func PrintSection(text string) {
	fmt.Printf("\n%s%s%s%s\n", constants.ColorYellow, constants.Bold, text, constants.ColorReset)
}

func PrintSuccess(text string) {
	fmt.Printf("%s✔ %s%s\n", constants.ColorGreen, text, constants.ColorReset)
}

func PrintWarning(text string) {
	fmt.Printf("%s! %s%s\n", constants.ColorYellow, text, constants.ColorReset)
}

func PrintError(msg string, err error) {
	fmt.Printf("%s✘ %s: %v%s\n", constants.ColorRed, msg, err, constants.ColorReset)
}

func PrintKeyValue(key, value string) {
	fmt.Printf("%s%-20s:%s %s\n", constants.ColorBlue, key, constants.ColorReset, value)
}

func PrintHint(text string) {
	fmt.Printf("%s%s%s%s\n", constants.ColorGray, constants.Bold, text, constants.ColorReset)
}
