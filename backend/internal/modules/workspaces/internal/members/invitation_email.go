package members

import (
	"fmt"
	"html"
)

type invitationEmail struct {
	subject string
	body    string
}

// newInvitationEmail links to the instance when the operator configured its public address, and
// otherwise asks the reader to visit it.
func newInvitationEmail(workspaceName, inviterName, role, publicURL string) invitationEmail {
	signUpLink := `<p style="margin: 20px 0; color: #666;">
			Please visit your Databasus instance to sign up and access the workspace.
		</p>`
	if publicURL != "" {
		signUpLink = fmt.Sprintf(`<p style="margin: 20px 0;">
			<a href="%s" style="display: inline-block; padding: 12px 24px; background-color: #0d6efd; color: white; text-decoration: none; border-radius: 4px;">
				Sign up
			</a>
		</p>`, html.EscapeString(publicURL))
	}

	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
	<div style="background-color: #f8f9fa; border-radius: 8px; padding: 30px; margin: 20px 0;">
		<h1 style="color: #0d6efd; margin-top: 0;">Workspace Invitation</h1>

		<p style="font-size: 16px; margin: 20px 0;">
			<strong>%s</strong> has invited you to join the <strong>%s</strong> workspace as a <strong>%s</strong>.
		</p>

		%s

		<hr style="border: none; border-top: 1px solid #dee2e6; margin: 30px 0;">

		<p style="font-size: 14px; color: #6c757d; margin: 0;">
			This is an automated message from Databasus. If you didn't expect this invitation, you can safely ignore this email.
		</p>
	</div>
</body>
</html>
	`, html.EscapeString(inviterName), html.EscapeString(workspaceName), html.EscapeString(role), signUpLink)

	return invitationEmail{
		subject: fmt.Sprintf("You've been invited to %s workspace", workspaceName),
		body:    body,
	}
}
