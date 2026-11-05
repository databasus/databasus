package apierror

// Problem is an error a service returns when the client must tell its cause apart. Write turns it
// into the RFC 9457 body that every contract operation answers with.
type Problem struct { //nolint:errname // named after the RFC 9457 Problem body it is written as
	Code   Code
	Detail string
	Errors []FieldError
}

type FieldError struct {
	Field  string `json:"field"`
	Detail string `json:"detail"`
}

func (p *Problem) Error() string {
	return string(p.Code) + ": " + p.Detail
}
