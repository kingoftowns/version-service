package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseAppID(t *testing.T) {
	tests := []struct {
		name          string
		appID         string
		wantProjectID string
		wantAppName   string
		wantErr       bool
	}{
		{
			name:          "valid app ID",
			appID:         "1234-user-service",
			wantProjectID: "1234",
			wantAppName:   "user-service",
			wantErr:       false,
		},
		{
			name:          "app name with multiple hyphens",
			appID:         "5678-payment-gateway-service",
			wantProjectID: "5678",
			wantAppName:   "payment-gateway-service",
			wantErr:       false,
		},
		{
			name:          "invalid app ID - no hyphen",
			appID:         "invalidappid",
			wantProjectID: "",
			wantAppName:   "",
			wantErr:       true,
		},
		{
			name:          "invalid app ID - only project ID",
			appID:         "1234",
			wantProjectID: "",
			wantAppName:   "",
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projectID, appName, err := ParseAppID(tt.appID)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantProjectID, projectID)
				assert.Equal(t, tt.wantAppName, appName)
			}
		})
	}
}

func TestFormatAppID(t *testing.T) {
	tests := []struct {
		name      string
		projectID string
		appName   string
		want      string
	}{
		{
			name:      "simple app name",
			projectID: "1234",
			appName:   "user-service",
			want:      "1234-user-service",
		},
		{
			name:      "app name with hyphens",
			projectID: "5678",
			appName:   "payment-gateway-service",
			want:      "5678-payment-gateway-service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatAppID(tt.projectID, tt.appName)
			assert.Equal(t, tt.want, got)
		})
	}
}