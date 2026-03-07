package httpapi

import "testing"

func TestResolveModelRequestTimeoutMS_Default(t *testing.T) {
	if got := resolveModelRequestTimeoutMS(nil); got != 30000 {
		t.Fatalf("expected default request timeout 30000, got %d", got)
	}
	if got := resolveModelRequestTimeoutMS(&ModelRuntimeSpec{}); got != 30000 {
		t.Fatalf("expected default request timeout for empty runtime, got %d", got)
	}
}

func TestResolveModelRequestTimeoutMS_ClampsOutOfRangeValues(t *testing.T) {
	if got := resolveModelRequestTimeoutMS(&ModelRuntimeSpec{RequestTimeoutMS: intPtr(999)}); got != 1000 {
		t.Fatalf("expected lower clamp to 1000, got %d", got)
	}
	if got := resolveModelRequestTimeoutMS(&ModelRuntimeSpec{RequestTimeoutMS: intPtr(120001)}); got != 120000 {
		t.Fatalf("expected upper clamp to 120000, got %d", got)
	}
}

func TestValidateModelRuntimeSpec_Range(t *testing.T) {
	cases := []struct {
		name    string
		runtime *ModelRuntimeSpec
		wantErr bool
	}{
		{name: "nil runtime", runtime: nil, wantErr: false},
		{name: "empty runtime", runtime: &ModelRuntimeSpec{}, wantErr: false},
		{name: "minimum", runtime: &ModelRuntimeSpec{RequestTimeoutMS: intPtr(1000)}, wantErr: false},
		{name: "maximum", runtime: &ModelRuntimeSpec{RequestTimeoutMS: intPtr(120000)}, wantErr: false},
		{name: "below minimum", runtime: &ModelRuntimeSpec{RequestTimeoutMS: intPtr(999)}, wantErr: true},
		{name: "above maximum", runtime: &ModelRuntimeSpec{RequestTimeoutMS: intPtr(120001)}, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateModelRuntimeSpec(tc.runtime)
			if tc.wantErr && err == nil {
				t.Fatalf("expected validation error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}
