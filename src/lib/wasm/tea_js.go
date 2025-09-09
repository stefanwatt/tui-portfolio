//go:build js
// +build js

package tea

const suspendSupported = false

func suspendProcess() error {
    return nil
}
