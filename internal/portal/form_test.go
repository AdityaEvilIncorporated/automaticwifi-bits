package portal

import (
	"reflect"
	"testing"
)

func TestParseForms(t *testing.T) {
	body := []byte(`<html><body>
<input name="outside" value="x">
<FORM Action="/" METHOD="POST">
  <div><input type="hidden" name="4Tredir" value="http://neverssl.com/"></div>
  <input TYPE="Hidden" name="magic" value="0a1b">
  <table><tr><td><input name="username" id="u"></td></tr></table>
  <input type="password" name="password">
  <input type="submit" value="Continue">
</FORM>
<form action="/two"><input name="q" value="&amp;&lt;"></form>
</body></html>`)
	got := ParseForms(body)
	want := []Form{
		{Action: "/", Method: "post", Inputs: []Input{
			{Name: "4Tredir", Type: "hidden", Value: "http://neverssl.com/"},
			{Name: "magic", Type: "hidden", Value: "0a1b"},
			{Name: "username", Type: "", Value: ""},
			{Name: "password", Type: "password", Value: ""},
			{Name: "", Type: "submit", Value: "Continue"},
		}},
		{Action: "/two", Method: "", Inputs: []Input{{Name: "q", Value: "&<"}}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got  %#v\nwant %#v", got, want)
	}
	if !got[0].Has("magic") || got[0].Has("Magic") || got[0].Has("outside") {
		t.Fatal("Has wrong")
	}
	if v, ok := got[0].Value("magic"); !ok || v != "0a1b" {
		t.Fatalf("Value = %q %v", v, ok)
	}
	if _, ok := got[0].Value("nope"); ok {
		t.Fatal("Value found missing input")
	}
}

func TestParseFormsMalformedAndEmpty(t *testing.T) {
	if ParseForms(nil) != nil {
		t.Fatal("nil body should give nil")
	}
	if got := ParseForms([]byte("no html at all")); len(got) != 0 {
		t.Fatalf("got %v", got)
	}
	got := ParseForms([]byte(`<form action=/ ><input name=a value=1><input name=b`))
	if len(got) != 1 || !got[0].Has("a") {
		t.Fatalf("unclosed form: %#v", got)
	}
}
