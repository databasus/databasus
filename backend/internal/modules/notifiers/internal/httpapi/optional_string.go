package httpapi

// getStringOrEmpty reads an optional field of the contract, where absent means empty.
func getStringOrEmpty(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

// toOptionalString leaves an empty value out of the response.
func toOptionalString(value string) *string {
	if value == "" {
		return nil
	}

	return &value
}
