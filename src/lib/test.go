// main.go
package main

import (
	"strings"
	"syscall/js"
)

func main() {
	c := make(chan struct{}, 0)

	// Expose a function to JS
	js.Global().Set("sendLine", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		line := args[0].String()
		result := strings.ToUpper(line)
		js.Global().Call("receiveLine", result) // call JS callback
		return nil
	}))

	<-c // keep WASM alive
}
