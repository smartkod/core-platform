package service

import (
	"testing"

	"core-platform/contract/core"
)

func TestOfferingTransitions(t *testing.T) {
	cases := []struct {
		from, to string
		want     bool
	}{
		{core.OfferingStatusDraft, core.OfferingStatusPublished, true},
		{core.OfferingStatusDraft, core.OfferingStatusArchived, true},
		{core.OfferingStatusPublished, core.OfferingStatusArchived, true},
		{core.OfferingStatusArchived, core.OfferingStatusDraft, true},
		{core.OfferingStatusDraft, core.OfferingStatusDraft, false},
		{core.OfferingStatusPublished, core.OfferingStatusDraft, false},
		{core.OfferingStatusArchived, core.OfferingStatusPublished, false},
	}

	for _, tc := range cases {
		if got := allowedOfferingTransition(tc.from, tc.to); got != tc.want {
			t.Errorf("%s -> %s = %v, want %v", tc.from, tc.to, got, tc.want)
		}
	}
}

func TestMediaRoundTrip(t *testing.T) {
	raw, err := encodeMedia([]string{"file-1", "file-2"})
	if err != nil {
		t.Fatal(err)
	}
	got := decodeMedia(raw)
	if len(got) != 2 || got[0] != "file-1" || got[1] != "file-2" {
		t.Fatalf("decoded = %v", got)
	}

	if raw, err := encodeMedia(nil); err != nil || raw != "[]" {
		t.Fatalf("nil media = %q, %v", raw, err)
	}
}

func TestExtensionRoundTrip(t *testing.T) {
	in := &core.ExtensionRef{ExtensionID: "booking", ResourceType: "service", ResourceID: "s1"}
	raw, err := encodeExtension(in)
	if err != nil || raw == nil {
		t.Fatalf("encode: %v", err)
	}
	out := decodeExtension(raw)
	if out == nil || out.ExtensionID != "booking" || out.ResourceID != "s1" {
		t.Fatalf("decoded = %+v", out)
	}
}

func TestPriceZeroIsAbsent(t *testing.T) {
	if optionalPrice(0) != nil {
		t.Fatal("zero price should mean 'not stated'")
	}
	if p := optionalPrice(12.5); p == nil || *p != 12.5 {
		t.Fatal("positive price should be kept")
	}
}
