package portal

import (
	"bytes"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

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
	_, ok := f.Value(name)
	return ok
}

// Value returns the value of the first input with this name, and whether it exists.
func (f Form) Value(name string) (string, bool) {
	for _, in := range f.Inputs {
		if in.Name == name {
			return in.Value, true
		}
	}
	return "", false
}

// ParseForms parses HTML tolerantly with golang.org/x/net/html (tag and attribute names are
// case-insensitive there) and returns every <form> in document order. Inputs that appear
// outside any form are ignored. An input nested inside a form at any depth belongs to that
// form. Malformed HTML never returns an error: whatever was parsed is returned; nil body
// returns nil.
func ParseForms(body []byte) []Form {
	if body == nil {
		return nil
	}
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil || doc == nil {
		return nil
	}
	var forms []Form
	var walk func(n *html.Node, cur *Form)
	walk = func(n *html.Node, cur *Form) {
		if n.Type == html.ElementNode {
			switch n.DataAtom {
			case atom.Form:
				f := Form{
					Action: attr(n, "action"),
					Method: strings.ToLower(attr(n, "method")),
				}
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					walk(c, &f)
				}
				forms = append(forms, f)
				return
			case atom.Input:
				if cur != nil {
					cur.Inputs = append(cur.Inputs, Input{
						Name:  attr(n, "name"),
						Type:  strings.ToLower(attr(n, "type")),
						Value: attr(n, "value"),
					})
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, cur)
		}
	}
	walk(doc, nil)
	return forms
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}
