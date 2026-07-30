package main

import "testing"

func TestResolveVersion(t *testing.T) {
	tests := []struct {
		name     string
		injected string
		module   string
		want     string
	}{
		{name: "injected release version", injected: "v1.2.3", module: "(devel)", want: "v1.2.3"},
		{name: "module version", module: "v1.2.3", want: "v1.2.3"},
		{name: "module pseudo-version", module: "v0.0.0-20260729213933-915b3a77665b", want: "v0.0.0-20260729213933-915b3a77665b"},
		{name: "development build", module: "(devel)", want: "devel"},
		{name: "missing build information", want: "devel"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveVersion(tt.injected, tt.module); got != tt.want {
				t.Fatalf("resolveVersion(%q, %q) = %q, want %q", tt.injected, tt.module, got, tt.want)
			}
		})
	}
}
