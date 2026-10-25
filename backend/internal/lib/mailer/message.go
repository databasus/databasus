package mailer

import (
	"bytes"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Message struct {
	Sender    *mail.Address
	Recipient string
	Subject   string
	HTMLBody  string
}

var headerBreakingCharacters = strings.NewReplacer("\r", "", "\n", "", "\x00", "")

func buildMessage(sender, recipient *mail.Address, subject, htmlBody string, sentAt time.Time) ([]byte, error) {
	var content bytes.Buffer

	headers := [][2]string{
		{"From", sender.String()},
		{"To", recipient.String()},
		{"Subject", mime.QEncoding.Encode("UTF-8", headerBreakingCharacters.Replace(subject))},
		{"Date", sentAt.Format(time.RFC1123Z)},
		{"Message-ID", fmt.Sprintf("<%s@%s>", uuid.New(), getAddressDomain(sender.Address))},
		{"MIME-Version", "1.0"},
		{"Content-Type", `text/html; charset="UTF-8"`},
		{"Content-Transfer-Encoding", "quoted-printable"},
	}

	for _, header := range headers {
		content.WriteString(header[0] + ": " + header[1] + "\r\n")
	}

	content.WriteString("\r\n")

	bodyWriter := quotedprintable.NewWriter(&content)
	if _, err := bodyWriter.Write([]byte(htmlBody)); err != nil {
		return nil, fmt.Errorf("failed to encode message body: %w", err)
	}

	if err := bodyWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to encode message body: %w", err)
	}

	return content.Bytes(), nil
}

func getAddressDomain(address string) string {
	return address[strings.LastIndex(address, "@")+1:]
}
