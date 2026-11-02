package hands

import (
	"testing"

	"github.com/akhenakh/gmid/internal/smf"
)

func TestAssignDemo(t *testing.T) {
	s, err := smf.ReadFile("../../assets/demo/Scott Joplin - Entertainer.mid")
	if err != nil {
		t.Skipf("demo missing: %v", err)
	}
	AssignSong(s, Defaults())
	l, r, none := 0, 0, 0
	for i := range s.Tracks {
		for _, n := range s.Tracks[i].Notes {
			switch n.Hand {
			case "L":
				l++
			case "R":
				r++
			default:
				none++
			}
		}
	}
	if l+r == 0 {
		t.Fatal("no hands assigned")
	}
	if none > 0 {
		t.Fatalf("%d notes without a hand", none)
	}
	if l == 0 || r == 0 {
		t.Fatalf("one-handed assignment: L=%d R=%d", l, r)
	}
	t.Logf("L=%d R=%d", l, r)
}
