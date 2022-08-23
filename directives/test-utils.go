package directives

import (
	"github.com/syntax-framework/shtml/sht"
	"io/fs"
	"strings"
	"testing"
)

// testGDs test global directives
var testGDs = &sht.Directives{}

// testFileLoader in memory file loader
func testFileLoader(files map[string]string) func(filepath string) (string, error) {
	return func(filepath string) (string, error) {
		content, exist := files[filepath]
		if !exist {
			return "", fs.ErrNotExist
		}
		return content, nil
	}
}

func testForErrorCode(t *testing.T, template string, errorCode string) {
	ts := &sht.TemplateSystem{
		Loader: testFileLoader(map[string]string{
			"template.html": sht.TestUnindentedTemplate(template),
		}),
		Directives: testGDs.NewChild(),
	}
	_, err := ts.Compile("template.html")

	if err == nil {
		t.Errorf("compiler.Compile(template) | expect to receive compilation error")
	} else {
		errStr := err.Error()
		if !strings.HasPrefix(errStr, "["+errorCode+"]") {
			t.Errorf("compiler.Compile(template) | invalid error\n expected: [%s] .......\n   actual: %s", errorCode, errStr)
		}
	}
}

func init() {
	testGDs.Add(IFElement)
	testGDs.Add(IFAttribute)
	testGDs.Add(Component)
}
