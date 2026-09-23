package parser

import (
	"testing"
)

type VersionObject struct {
	Version string
}

type TestCase struct {
	Version     string
	Latest      string
	LatestMinor string
	LatestPatch string
}

func Test_GetVersionInfo(t *testing.T) {
	versions := []struct{ Version string }{
		{Version: "0.1.0"},
		{Version: "0.2.0"},
		{Version: "1.0.0"},
		{Version: "1.0.1"},
		{Version: "1.0.2"},
		{Version: "1.1.0"},
		{Version: "1.2.0"},
		{Version: "2.0.0"},
	}

	testCases := []TestCase{
		{Version: "0.1.0", Latest: "2.0.0", LatestMinor: "0.2.0", LatestPatch: "0.1.0"},
		{Version: "0.2.0", Latest: "2.0.0", LatestMinor: "0.2.0", LatestPatch: "0.2.0"},
		{Version: "1.0.0", Latest: "2.0.0", LatestMinor: "1.2.0", LatestPatch: "1.0.2"},
		{Version: "1.0.1", Latest: "2.0.0", LatestMinor: "1.2.0", LatestPatch: "1.0.2"},
		{Version: "1.1.0", Latest: "2.0.0", LatestMinor: "1.2.0", LatestPatch: "1.1.0"},
	}

	// Test should be
	for _, testCase := range testCases {
		runTest_GetVersionInfo(t, testCase, versions)
	}
}

func runTest_GetVersionInfo(
	t *testing.T,
	testCase TestCase,
	versions []struct{ Version string },
) {
	latest, latestMinor, latestPatch := GetVersionInfo(versions, "v"+testCase.Version)

	if latest != "v"+testCase.Latest {
		t.Errorf("GetVersionInfo(%v, List).Latest %v, want %v", testCase.Version, latest, testCase.Latest)
	}

	if latestMinor != "v"+testCase.LatestMinor {
		t.Errorf("GetVersionInfo(%v, List).LatestMinor %v, want %v", testCase.Version, latestMinor, "v"+testCase.LatestMinor)
	}

	if latestPatch != "v"+testCase.LatestPatch {
		t.Errorf("GetVersionInfo(%s, List).LatestPatch %v, want %v", testCase.Version, latestPatch, "v"+testCase.LatestPatch)
	}
}
