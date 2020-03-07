package vtu

import "testing"

// basic payload functionality
func TestSetHeader(t *testing.T) {
	p := NewPayload()
	if p.head.Len() > 0 {
		t.Errorf("New header is not empty, len: %v", p.head.Len())
	}

	if p.body.Len() > 0 {
		t.Errorf("New body is not empty, len: %v", p.body.Len())
	}

	p.setHeader()
	if p.head.Len() != 4 {
		t.Errorf("Int32 header not right length: %v", p.head.Len())
	}

	if p.compressed() {
		t.Errorf("New header should not be compressed.")
	}
}
