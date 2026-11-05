package httpapi

// getValueOrZero reads an optional field of the contract, where absent means the zero value.
func getValueOrZero[T any](value *T) T {
	if value == nil {
		var zeroValue T

		return zeroValue
	}

	return *value
}

// toOptionalValue leaves a zero value out of the response.
func toOptionalValue[T comparable](value T) *T {
	var zeroValue T
	if value == zeroValue {
		return nil
	}

	return &value
}

// toNonNilStrings keeps a list the contract requires an array, never null.
func toNonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}

	return values
}
