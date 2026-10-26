// Package webhook sends notifications to an HTTP endpoint the user runs, with a GET carrying the
// text in the query or a POST carrying it in a body.
package webhook

import "databasus-backend/internal/modules/notifiers/internal/notification"

type Method string

const (
	MethodPOST Method = "POST"
	MethodGET  Method = "GET"
)

type Header struct {
	Name string
	// Value is a secret, because a header usually carries the endpoint's credentials.
	Value string
}

type Settings struct {
	// URL is not a secret and is returned in clear.
	URL    string
	Method Method
	// BodyTemplate is empty for the default JSON body. {{heading}} and {{message}} are replaced.
	BodyTemplate string
	Headers      []Header
	// AcceptedCategories narrows the notifications the webhook gets; CategoryAll accepts every one.
	AcceptedCategories []notification.Category
	// storedFieldsDecryptionErr is set by the repository when it cannot decrypt the URL or the body
	// template that an older release stored encrypted. The field is empty then, and Send refuses.
	storedFieldsDecryptionErr error
}

func (Settings) GetType() notification.NotifierType {
	return notification.NotifierTypeWebhook
}
