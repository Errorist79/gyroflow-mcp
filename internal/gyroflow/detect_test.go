package gyroflow

import "testing"

func TestParseVersion(t *testing.T) {
	cases := map[string]string{
		"Gyroflow v1.6.2\n":                    "1.6.2",
		"gyroflow 1.6.3":                       "1.6.3",
		"Video stabilization\nGyroflow v1.7.0": "1.7.0",
	}
	for in, want := range cases {
		got, err := parseVersion(in)
		if err != nil {
			t.Fatalf("parseVersion(%q) error: %v", in, err)
		}
		if got != want {
			t.Errorf("parseVersion(%q) = %q, want %q", in, got, want)
		}
	}
	if _, err := parseVersion("no version here"); err == nil {
		t.Error("expected error for missing version")
	}
}
