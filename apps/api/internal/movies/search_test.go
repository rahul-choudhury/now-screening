package movies

import "testing"

func TestNormalizeQuery(t *testing.T) {
	t.Parallel()

	input := "  How&nbsp;to\u00a0Train&nbsp;Your&nbsp;Dragon  "
	got := NormalizeQuery(input)

	if got != "How to Train Your Dragon" {
		t.Fatalf("NormalizeQuery() = %q, want %q", got, "How to Train Your Dragon")
	}
}

func TestFuzzySearchPrefersBestMatches(t *testing.T) {
	t.Parallel()

	list := []Movie{
		{Title: "Ballerina", Href: "/ballerina"},
		{Title: "The Ballad of Wallis Island", Href: "/ballad"},
		{Title: "Ballerina (2025)", Href: "/ballerina-2025"},
	}

	got := FuzzySearch(list, "Ballerina")

	if len(got) < 2 {
		t.Fatalf("FuzzySearch() returned %d matches, want at least 2", len(got))
	}

	if got[0].Title != "Ballerina" {
		t.Fatalf("first match = %q, want %q", got[0].Title, "Ballerina")
	}

	if got[1].Title != "Ballerina (2025)" {
		t.Fatalf("second match = %q, want %q", got[1].Title, "Ballerina (2025)")
	}
}

func TestFuzzySearchEmptyList(t *testing.T) {
	t.Parallel()

	var list []Movie
	got := FuzzySearch(list, "anything")

	if len(got) != 0 {
		t.Fatalf("FuzzySearch() returned %d items, want 0", len(got))
	}
}
