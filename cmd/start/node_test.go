package start

import (
	"testing"
)

func TestParsePath(t *testing.T) {
	input := []string{
		"/node/@backstage/core-components/-/core-components-0.1.0.tgz",
		"/node/@backstage/core-components/",
		"/node/@aws-sdk/client-s3/-/client-s3-3.0.0.tgz",
		"/node/@aws-sdk/client-s3/",
		"/node/es-object-assign",
		"/node/es-object-assign/-/es-object-assign-1.1.0.tgz",
		"/node/yarn",
		"/node/yarn/-/yarn-1.22.10.tgz",
	}
	// initialize arrays, slices like this x:= []type { data, data, data}
	tests := []struct {
		expectedPackage NodePackage
	}{
		{NodePackage{scope: "@backstage", name: "core-components"}},
		{NodePackage{scope: "@backstage", name: "core-components"}},
		{NodePackage{scope: "@aws-sdk", name: "client-s3"}},
		{NodePackage{scope: "@aws-sdk", name: "client-s3"}},
		{NodePackage{scope: "", name: "es-object-assign"}},
		{NodePackage{scope: "", name: "es-object-assign"}},
		{NodePackage{scope: "", name: "yarn"}},
		{NodePackage{scope: "", name: "yarn"}},
	}

	for i, tt := range tests {
		nodePackage, err := parsePath(input[i])
		if err != nil {
			t.Fatalf("test[%d] - failed to parse path: %s", i, err)
		}

		if nodePackage.scope != tt.expectedPackage.scope {
			t.Fatalf("test[%d] - scope wrong. Expected %q got=%q",
				i, tt.expectedPackage.scope, nodePackage.scope)
		}
		if nodePackage.name != tt.expectedPackage.name {
			t.Fatalf("test[%d] - name wrong. Expected %q got=%q",
				i, tt.expectedPackage.name, nodePackage.name)
		}
	}
}

func TestMatchWildCard(t *testing.T) {
	input := []struct {
		s        string
		pattern  string
		expected bool
	}{
		{s: "es-object-atoms", pattern: "es-*", expected: true},
		{s: "@es/errors", pattern: "@es/*", expected: true},
		{s: "es-object", pattern: "es-errors-*", expected: false},
		{s: "@es/core", pattern: "*/core", expected: true},
	}

	for i, tt := range input {
		matched, err := matchWildCard(tt.s, tt.pattern)
		if err != nil {
			t.Fatalf("test[%d] - failed to match wildcard: %s", i, err)
		}
		if matched != tt.expected {
			t.Fatalf("test[%d] - match wrong. Expected %t got=%t", i, tt.expected, matched)
		}

		t.Log("Matched: ", matched, "Expected: ", tt.expected, "Pattern: ", tt.pattern, "String: ", tt.s)
	}
}
