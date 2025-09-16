package semver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *Version
		wantErr bool
	}{
		{
			name:  "valid version",
			input: "1.2.3",
			want: &Version{
				Major: 1,
				Minor: 2,
				Patch: 3,
			},
			wantErr: false,
		},
		{
			name:  "valid version with prerelease",
			input: "1.2.3-dev-abc1234",
			want: &Version{
				Major:      1,
				Minor:      2,
				Patch:      3,
				Prerelease: "dev-abc1234",
			},
			wantErr: false,
		},
		{
			name:    "invalid version",
			input:   "invalid",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "incomplete version",
			input:   "1.2",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestVersion_String(t *testing.T) {
	tests := []struct {
		name    string
		version *Version
		want    string
	}{
		{
			name: "version without prerelease",
			version: &Version{
				Major: 1,
				Minor: 2,
				Patch: 3,
			},
			want: "1.2.3",
		},
		{
			name: "version with prerelease",
			version: &Version{
				Major:      1,
				Minor:      2,
				Patch:      3,
				Prerelease: "dev-abc1234",
			},
			want: "1.2.3-dev-abc1234",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.version.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestVersion_IncrementPatch(t *testing.T) {
	v := &Version{Major: 1, Minor: 2, Patch: 3}
	result := v.IncrementPatch()
	assert.Equal(t, "1.2.4", result.String())
	assert.Equal(t, "", result.Prerelease)
}

func TestVersion_IncrementMinor(t *testing.T) {
	v := &Version{Major: 1, Minor: 2, Patch: 3}
	result := v.IncrementMinor()
	assert.Equal(t, "1.3.0", result.String())
	assert.Equal(t, "", result.Prerelease)
}

func TestVersion_IncrementMajor(t *testing.T) {
	v := &Version{Major: 1, Minor: 2, Patch: 3}
	result := v.IncrementMajor()
	assert.Equal(t, "2.0.0", result.String())
	assert.Equal(t, "", result.Prerelease)
}

func TestVersion_WithDevSuffix(t *testing.T) {
	tests := []struct {
		name string
		sha  string
		want string
	}{
		{
			name: "short SHA",
			sha:  "abc1234",
			want: "1.2.3-dev-abc1234",
		},
		{
			name: "long SHA",
			sha:  "abc1234567890def",
			want: "1.2.3-dev-abc1234",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &Version{Major: 1, Minor: 2, Patch: 3}
			result := v.WithDevSuffix(tt.sha)
			assert.Equal(t, tt.want, result.String())
		})
	}
}

func TestIsValid(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    bool
	}{
		{"valid version", "1.2.3", true},
		{"valid with prerelease", "1.2.3-dev", true},
		{"invalid version", "invalid", false},
		{"incomplete version", "1.2", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValid(tt.version)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCompare(t *testing.T) {
	tests := []struct {
		name    string
		v1      string
		v2      string
		want    int
		wantErr bool
	}{
		{"equal versions", "1.2.3", "1.2.3", 0, false},
		{"v1 greater major", "2.0.0", "1.0.0", 1, false},
		{"v1 lesser major", "1.0.0", "2.0.0", -1, false},
		{"v1 greater minor", "1.2.0", "1.1.0", 1, false},
		{"v1 lesser minor", "1.1.0", "1.2.0", -1, false},
		{"v1 greater patch", "1.1.2", "1.1.1", 1, false},
		{"v1 lesser patch", "1.1.1", "1.1.2", -1, false},
		{"release vs prerelease", "1.2.3", "1.2.3-dev", 1, false},
		{"prerelease vs release", "1.2.3-dev", "1.2.3", -1, false},
		{"invalid v1", "invalid", "1.2.3", 0, true},
		{"invalid v2", "1.2.3", "invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Compare(tt.v1, tt.v2)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.want > 0 {
					assert.Greater(t, got, 0)
				} else if tt.want < 0 {
					assert.Less(t, got, 0)
				} else {
					assert.Equal(t, 0, got)
				}
			}
		})
	}
}