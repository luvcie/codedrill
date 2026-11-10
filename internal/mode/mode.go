// Package mode holds a drill's reference and pass-gate options. The four
// booleans are authoritative; the named presets are shortcuts that produce
// them. UI and evaluation can share this package without import cycles.
package mode

import "errors"

// Options are the reference and pass-gate settings for one drill.
type Options struct {
	CheckReference bool `json:"check_reference"`
	ShowReference  bool `json:"show_reference"`
	RequireExact   bool `json:"require_exact"`
	RequireTests   bool `json:"require_tests"`
}

// Study shows the reference, checks it and requires tests.
func Study() Options {
	return Options{CheckReference: true, ShowReference: true, RequireTests: true}
}

// Copy shows the reference and requires an exact match, without tests.
func Copy() Options {
	return Options{CheckReference: true, ShowReference: true, RequireExact: true}
}

// Recall checks the reference without showing it and requires an exact match.
func Recall() Options {
	return Options{CheckReference: true, RequireExact: true}
}

// Exam disables reference checking and requires tests.
func Exam() Options {
	return Options{RequireTests: true}
}

// Default is the mode for drills without saved preferences.
func Default() Options {
	return Study()
}

// Normalize clears reference options that cannot be used when checking is
// off. It does not enable a pass gate, so an invalid gate selection stays
// visible to Validate instead of turning tests on silently.
func (o Options) Normalize() Options {
	if !o.CheckReference {
		o.ShowReference = false
		o.RequireExact = false
	}
	return o
}

// Validate reports whether the normalized options leave at least one pass
// gate. Reference consistency is normalized rather than reported, because
// disabling reference checking intentionally clears the other reference
// options.
func (o Options) Validate() error {
	n := o.Normalize()
	if !n.RequireTests && !n.RequireExact {
		return errors.New("at least one pass gate must be enabled")
	}
	return nil
}
