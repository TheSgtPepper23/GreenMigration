package extras

import (
	"strconv"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

func StringToDateDefault(dateString string) time.Time {
	converted, err := time.Parse("2006-01-02", dateString)
	if err != nil {
		return time.Now()
	}

	return converted
}

func StringToIntDefault(numberString string) int {
	converted, err := strconv.Atoi(numberString)
	if err != nil {
		return 0
	}

	return converted
}

func StringToFloatDefault(numerString string) float32 {
	converted, err := strconv.ParseFloat(numerString, 32)
	if err != nil {
		converted = 0.0
	}

	return float32(converted)
}

func CompareStringsBothWays(stringA, stringB string) bool {
	return strings.Contains(NormalizeString(stringA), NormalizeString(stringB)) ||
		strings.Contains(NormalizeString(stringB), NormalizeString(stringA))
}

func NormalizeString(regularString string) string {
	regularString = strings.ToLower(regularString)
	normInput := norm.NFD.String(regularString)

	var sb strings.Builder
	for _, r := range normInput {
		if !unicode.Is(unicode.Mn, r) {
			sb.WriteRune(r)
		}
	}
	return norm.NFC.String(sb.String())
}

func MatchBooks(bookA, bookB *Book) bool {
	return CompareStringsBothWays(bookA.Author, bookB.Author) && CompareStringsBothWays(bookA.Title, bookB.Title)
}
