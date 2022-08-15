package sht

import (
	"bytes"
	"strings"
	"testing"
)

func unindented(template string) string {
	if strings.HasPrefix(template, "\n    ") {
		template = strings.ReplaceAll(template, "\n    ", "\n")
	}
	return strings.TrimSpace(template)
}

// TestCompile compiles a template and already tests the expected output
func TestCompile(t *testing.T, template string, static []string, globalDirectives *Directives) *Compiled {
	template = unindented(template)
	compiler := NewCompiler(&TemplateSystem{Directives: globalDirectives.NewChild()})
	compiled, err := compiler.Compile(template)
	if err != nil {
		t.Fatal(err)
	}
	for i, expected := range static {
		if actual := compiled.static[i]; actual != expected {
			t.Name()
			t.Errorf("compiler.Compile(template) | invalid compiled.Static[%d]\n   actual: %q\n expected: %q", i, actual, expected)
		}
	}
	return compiled
}

// TestRender renders a compiled and already tests the expected result
func TestRender(t *testing.T, compiled *Compiled, values map[string]interface{}, expected string) {
	expected = unindented(expected)

	scope := NewRootScope()
	if values != nil {
		for key, value := range values {
			scope.Set(key, value)
		}
	}
	rendered := compiled.Exec(scope)

	out := &bytes.Buffer{}
	rendered.Write(out)
	if actual := out.String(); actual != expected {
		t.Errorf("compiled.Write(*bytes.Buffer) | invalid output\n   actual: %q\n expected: %q", actual, expected)
	}
}

func TestTemplate(
	t *testing.T,
	template string,
	values map[string]interface{},
	expected string,
	globalDirectives *Directives,
) {
	template = unindented(template)
	compiler := NewCompiler(&TemplateSystem{Directives: globalDirectives.NewChild()})
	compiled, err := compiler.Compile(template)
	if err != nil {
		t.Fatal(err)
	}
	TestRender(t, compiled, values, expected)
}
