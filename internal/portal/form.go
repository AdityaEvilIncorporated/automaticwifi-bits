package portal

// Input is one <input> element of a form. Type is lower-cased; empty when absent.
type Input struct {
	Name  string
	Type  string
	Value string
}

// Form is one <form> element: its action and method attributes as written (method
// lower-cased, empty when absent) and its inputs in document order.
type Form struct {
	Action string
	Method string
	Inputs []Input
}

// Has reports whether the form has an input with exactly this name (case-sensitive: field
// names are case-sensitive in HTTP form posts).
func (f Form) Has(name string) bool {
	panic("not implemented")
}

// Value returns the value of the first input with this name, and whether it exists.
func (f Form) Value(name string) (string, bool) {
	panic("not implemented")
}

// ParseForms parses HTML tolerantly with golang.org/x/net/html (tag and attribute names are
// case-insensitive there) and returns every <form> in document order. Inputs that appear
// outside any form are ignored. An input nested inside a form at any depth belongs to that
// form. Malformed HTML never returns an error: whatever was parsed is returned; nil body
// returns nil.
func ParseForms(body []byte) []Form {
	panic("not implemented")
}
