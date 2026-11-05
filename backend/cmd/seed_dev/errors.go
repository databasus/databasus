package main

type apiError struct {
	StatusCode int
	Reason     string
}

func (e *apiError) Error() string {
	return e.Reason
}
