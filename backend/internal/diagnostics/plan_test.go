package diagnostics

import (
	"strings"
	"testing"
)

func TestGenerate_Plans(t *testing.T) {
	b1 := Generate(true, "1-2", "b1")
	if b1.Score != 75 || b1.RecommendedTier != "basic" || len(b1.Steps) == 0 {
		t.Fatalf("unexpected b1 plan: %+v", b1)
	}
	if !strings.Contains(strings.Join(stepTitles(b1), " "), "Английский") {
		t.Fatal("b1 plan must include language practice")
	}
	c1 := Generate(true, "3", "c1")
	if c1.Score != 96 || c1.RecommendedTier != "accelerator" {
		t.Fatalf("unexpected c1 plan: %+v", c1)
	}
	senior := Generate(true, "senior", "b2")
	if senior.Warning == "" {
		t.Fatal("senior plan must warn about J-1")
	}
	invalid := Generate(false, "3", "b2")
	if invalid.Warning == "" || invalid.RecommendedTier != "" {
		t.Fatalf("ineligible plan should not sell a tier: %+v", invalid)
	}
	text := TelegramText(c1)
	if !strings.Contains(text, "не гарантия гранта") {
		t.Fatal("plan must not promise a grant")
	}
}

func stepTitles(p Plan) []string {
	out := make([]string, 0, len(p.Steps))
	for _, step := range p.Steps {
		out = append(out, step.Title)
	}
	return out
}
