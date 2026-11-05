package filename

func SanitizeFilename(name string) string {
	replacer := map[rune]rune{
		' ':  '_',
		'/':  '-',
		'\\': '-',
		':':  '-',
		'*':  '-',
		'?':  '-',
		'"':  '-',
		'<':  '-',
		'>':  '-',
		'|':  '-',
	}

	result := make([]rune, 0, len(name))
	for _, char := range name {
		if replacement, exists := replacer[char]; exists {
			result = append(result, replacement)
		} else {
			result = append(result, char)
		}
	}

	return string(result)
}
