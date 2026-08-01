package observability

var enabled = true

func Enable() {
	enabled = true
}

func Disable() {
	enabled = false
}

func Enabled() bool {
	return enabled
}
