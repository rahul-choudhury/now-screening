package movies

import (
	"html"
	"strings"

	"github.com/sahilm/fuzzy"
)

func NormalizeQuery(query string) string {
	htmlDecoded := html.UnescapeString(query)

	// Convert non-breaking spaces to ASCII spaces so fuzzy matching lines up
	// with titles stored in the database.
	cleaned := strings.ReplaceAll(htmlDecoded, "\u00a0", " ")

	return strings.TrimSpace(cleaned)
}

func FuzzySearch(list []Movie, query string) []Movie {
	if len(list) == 0 {
		return list
	}

	titles := make([]string, len(list))
	for i, movie := range list {
		titles[i] = movie.Title
	}

	matches := fuzzy.Find(query, titles)

	result := make([]Movie, 0, len(matches))
	for _, match := range matches {
		result = append(result, list[match.Index])
	}

	return result
}
