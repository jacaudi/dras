package version

import (
	"runtime"
	"strings"
	"testing"
)

func TestGet(t *testing.T) {
	info := Get()
	
	// Verify that all fields are populated
	if info.Version == "" {
		t.Error("Version should not be empty")
	}
	if info.BuildTime == "" {
		t.Error("BuildTime should not be empty")
	}
	if info.GitCommit == "" {
		t.Error("GitCommit should not be empty")
	}
	if info.GitBranch == "" {
		t.Error("GitBranch should not be empty")
	}
	if info.GoVersion == "" {
		t.Error("GoVersion should not be empty")
	}
	
	// Verify GoVersion matches runtime
	if info.GoVersion != runtime.Version() {
		t.Errorf("Expected GoVersion to be %s, got %s", runtime.Version(), info.GoVersion)
	}
}

func TestInfo_String(t *testing.T) {
	tests := []struct {
		name     string
		info     Info
		contains []string
	}{
		{
			name: "development version",
			info: Info{
				Version:   "development",
				BuildTime: "2023-01-01T12:00:00Z",
				GitCommit: "abc123",
				GitBranch: "main",
				GoVersion: "go1.21.0",
			},
			contains: []string{"DRAS development", "commit: abc123", "branch: main", "go: go1.21.0"},
		},
		{
			name: "release version",
			info: Info{
				Version:   "1.2.3",
				BuildTime: "2023-01-01T12:00:00Z",
				GitCommit: "abc123",
				GitBranch: "main",
				GoVersion: "go1.21.0",
			},
			contains: []string{"DRAS v1.2.3", "built: 2023-01-01T12:00:00Z", "commit: abc123", "go: go1.21.0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.info.String()
			
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("String() result should contain %q, got: %s", expected, result)
				}
			}
		})
	}
}

func TestInfo_Short(t *testing.T) {
	tests := []struct {
		name     string
		info     Info
		expected string
	}{
		{
			name: "development version",
			info: Info{
				Version: "development",
			},
			expected: "DRAS development",
		},
		{
			name: "release version",
			info: Info{
				Version: "1.2.3",
			},
			expected: "DRAS v1.2.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.info.Short()
			if result != tt.expected {
				t.Errorf("Short() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestDefaultValues(t *testing.T) {
	// Test that default values are set correctly
	if Version != "development" {
		t.Errorf("Expected default Version to be 'development', got %s", Version)
	}
	if BuildTime != "unknown" {
		t.Errorf("Expected default BuildTime to be 'unknown', got %s", BuildTime)
	}
	if GitCommit != "unknown" {
		t.Errorf("Expected default GitCommit to be 'unknown', got %s", GitCommit)
	}
	if GitBranch != "unknown" {
		t.Errorf("Expected default GitBranch to be 'unknown', got %s", GitBranch)
	}
	if GoVersion != runtime.Version() {
		t.Errorf("Expected GoVersion to be %s, got %s", runtime.Version(), GoVersion)
	}
}