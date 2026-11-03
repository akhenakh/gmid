package ui

import (
	"testing"

	. "go.hasen.dev/shirei"
)

func TestQuitCombo(t *testing.T) {
	primary := PrimaryMod()
	if !isQuitCombo(KeyCombo{Key: KeyQ, Mod: primary}) {
		t.Fatal("quit combo not recognized")
	}
	if isQuitCombo(KeyCombo{Key: KeyQ}) {
		t.Fatal("bare Q must not quit")
	}
	if isQuitCombo(KeyCombo{Key: KeyW, Mod: primary}) {
		t.Fatal("other keys must not quit")
	}
}
