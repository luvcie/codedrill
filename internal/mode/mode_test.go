package mode

import "testing"

func TestPresets(t *testing.T) {
	cases := []struct {
		name string
		got  Options
		want Options
	}{
		{"study", Study(), Options{CheckReference: true, ShowReference: true, RequireTests: true}},
		{"copy", Copy(), Options{CheckReference: true, ShowReference: true, RequireExact: true}},
		{"recall", Recall(), Options{CheckReference: true, RequireExact: true}},
		{"exam", Exam(), Options{RequireTests: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Fatalf("preset is %+v, want %+v", tc.got, tc.want)
			}
			if err := tc.got.Validate(); err != nil {
				t.Fatalf("preset is invalid: %v", err)
			}
		})
	}
	if Default() != Study() {
		t.Fatalf("default is %+v, want Study", Default())
	}
}

func TestNormalizeReferenceConsistency(t *testing.T) {
	cases := []struct {
		name string
		in   Options
		want Options
	}{
		{
			name: "checking off clears show and exact",
			in:   Options{ShowReference: true, RequireExact: true, RequireTests: true},
			want: Options{RequireTests: true},
		},
		{
			name: "exact keeps checking on",
			in:   Options{CheckReference: true, RequireExact: true},
			want: Options{CheckReference: true, RequireExact: true},
		},
		{
			name: "show keeps checking on",
			in:   Options{CheckReference: true, ShowReference: true},
			want: Options{CheckReference: true, ShowReference: true},
		},
		{
			name: "gates are untouched",
			in:   Options{CheckReference: true, ShowReference: true},
			want: Options{CheckReference: true, ShowReference: true},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.in.Normalize()
			if got != tc.want {
				t.Fatalf("normalized to %+v, want %+v", got, tc.want)
			}
			if again := got.Normalize(); again != got {
				t.Fatalf("second normalization changed %+v to %+v", got, again)
			}
		})
	}
}

func TestValidateRequiresAPassGate(t *testing.T) {
	invalid := []Options{
		{},
		{CheckReference: true, ShowReference: true},
		{RequireExact: true},
		{ShowReference: true, RequireExact: true},
	}
	for _, opts := range invalid {
		if err := opts.Validate(); err == nil {
			t.Errorf("%+v validated without a pass gate", opts)
		}
	}
	valid := []Options{
		Study(),
		Copy(),
		Recall(),
		Exam(),
		{CheckReference: true, RequireExact: true, RequireTests: true},
		{CheckReference: false, ShowReference: true, RequireExact: true, RequireTests: true},
	}
	for _, opts := range valid {
		if err := opts.Validate(); err != nil {
			t.Errorf("%+v is invalid: %v", opts, err)
		}
	}
}
