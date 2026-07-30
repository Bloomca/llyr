package main

import "runtime/debug"

// injectedVersion is set through linker flags for release builds that do not
// carry a Go module version, such as binaries built by GoReleaser (in the future)
var injectedVersion string

func currentVersion() string {
	moduleVersion := ""
	if info, ok := debug.ReadBuildInfo(); ok {
		moduleVersion = info.Main.Version
	}

	return resolveVersion(injectedVersion, moduleVersion)
}

func resolveVersion(injected, module string) string {
	if injected != "" {
		return injected
	}
	if module != "" && module != "(devel)" {
		return module
	}
	return "devel"
}
