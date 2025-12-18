package mcgen

import (
	"testing"
)

func TestParseMinecraftVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    minecraftVersion
		wantErr bool
	}{
		{
			name:    "simple version",
			version: "1.21.1",
			want:    minecraftVersion{major: 1, minor: 21, patch: 1},
			wantErr: false,
		},
		{
			name:    "snapshot version",
			version: "26.1-snapshot-1",
			want:    minecraftVersion{major: 26, minor: 1, patch: 0},
			wantErr: false,
		},
		{
			name:    "two part version",
			version: "26.1",
			want:    minecraftVersion{major: 26, minor: 1, patch: 0},
			wantErr: false,
		},
		{
			name:    "invalid version",
			version: "invalid",
			want:    minecraftVersion{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseMinecraftVersion(tt.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseMinecraftVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseMinecraftVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNeedsYarnMappings(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    bool
	}{
		{
			name:    "1.21.1 needs Yarn",
			version: "1.21.1",
			want:    true,
		},
		{
			name:    "1.21.10 needs Yarn",
			version: "1.21.10",
			want:    true,
		},
		{
			name:    "26.0 needs Yarn",
			version: "26.0",
			want:    true,
		},
		{
			name:    "26.1 does not need Yarn",
			version: "26.1",
			want:    false,
		},
		{
			name:    "26.1-snapshot-1 does not need Yarn",
			version: "26.1-snapshot-1",
			want:    false,
		},
		{
			name:    "26.2 does not need Yarn",
			version: "26.2",
			want:    false,
		},
		{
			name:    "27.0 does not need Yarn",
			version: "27.0",
			want:    false,
		},
		{
			name:    "invalid version defaults to needs Yarn",
			version: "invalid",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := needsYarnMappings(tt.version); got != tt.want {
				t.Errorf("needsYarnMappings() = %v, want %v", got, tt.want)
			}
		})
	}
}
