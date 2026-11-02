package config

import "testing"

func TestFixupMigratesOldBufferAndBackend(t *testing.T) {
	p := Defaults()
	p.Version = 1
	p.BufferFrames = 256
	p.AudioBackend = "bogus"
	p.fixup()
	if p.Version != prefsVersion {
		t.Fatalf("version = %d", p.Version)
	}
	if p.BufferFrames != 512 {
		t.Fatalf("buffer = %d, want 512", p.BufferFrames)
	}
	if p.AudioBackend != "auto" {
		t.Fatalf("backend = %q, want auto", p.AudioBackend)
	}
}

func TestFixupKeepsDeliberateBuffer(t *testing.T) {
	p := Defaults()
	p.Version = prefsVersion
	p.BufferFrames = 128
	p.fixup()
	if p.BufferFrames != 128 {
		t.Fatalf("buffer = %d, want 128", p.BufferFrames)
	}
}
