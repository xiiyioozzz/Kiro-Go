package auth

import "testing"

func TestNormalizeIamSsoStartURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "old awsapps start URL",
			in:   "https://d-99674a7810.awsapps.com/start",
			want: "https://d-99674a7810.awsapps.com/start",
		},
		{
			name: "missing scheme on old awsapps URL",
			in:   "d-99674a7810.awsapps.com/start",
			want: "https://d-99674a7810.awsapps.com/start",
		},
		{
			name: "new app aws portal URL with extra slashes",
			in:   "//https://ssoins-65080ad437a5402d.portal.eu-north-1.app.aws///",
			want: "https://ssoins-65080ad437a5402d.portal.eu-north-1.app.aws/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeIamSsoStartURL(tt.in)
			if err != nil {
				t.Fatalf("normalizeIamSsoStartURL returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("normalizeIamSsoStartURL = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDiscoverIamSsoRegionFromAppAWSPortalURL(t *testing.T) {
	got, err := discoverIamSsoRegion("//https://ssoins-65080ad437a5402d.portal.eu-north-1.app.aws///")
	if err != nil {
		t.Fatalf("discoverIamSsoRegion returned error: %v", err)
	}
	if got != "eu-north-1" {
		t.Fatalf("discoverIamSsoRegion = %q, want %q", got, "eu-north-1")
	}
}
