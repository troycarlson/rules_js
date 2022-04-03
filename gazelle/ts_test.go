package gazelle

import (
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
)

func TestTypescriptApi(t *testing.T) {
	

	t.Run("parse a tsconfig", func(t *testing.T) {
		runfile, err := bazel.Runfile("gazelle/tests/ts_baseurl/project/tsconfig.json")
		if err != nil {
			t.Fatalf("cannot lookup runfile: %v", err)
		}
		
		options, err := ParseOptions(runfile)
		if err != nil {
			t.Fatalf("failed to parse options: %v", err)
		}
		if (options.BaseUrl != "src") {
			t.Errorf("ParseOptions:\nactual:   %s\nexpected:  %s\n", options.BaseUrl, "src")
		}
	})

}
