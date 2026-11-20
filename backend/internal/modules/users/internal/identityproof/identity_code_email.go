package identityproof

import "fmt"

const identityCodeSubject = "Databasus confirmation code"

func identityCodeBody(code string, purpose Purpose) string {
	action := "export the master key of this Databasus instance"
	if purpose == PurposeInstanceRecovery {
		action = "replace this Databasus installation with a Databasus backup"
	}

	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="margin: 0; padding: 0; font-family: Arial, sans-serif; background-color: #f4f4f4;">
    <div style="max-width: 600px; margin: 0 auto; background-color: #ffffff; padding: 20px;">
        <h2 style="color: #333333; margin-bottom: 20px;">Confirmation code</h2>
        <p style="color: #666666; line-height: 1.6; margin-bottom: 20px;">
            Root admin asked to %s. Enter the following code to confirm it:
        </p>
        <div style="background-color: #f8f9fa; border: 2px solid #e9ecef; border-radius: 8px; padding: 20px; text-align: center; margin: 30px 0;">
            <h1 style="color: #2c3e50; font-size: 36px; margin: 0; letter-spacing: 8px; font-family: monospace;">%s</h1>
        </div>
        <p style="color: #666666; line-height: 1.6; margin-bottom: 20px;">
            This code stops working <strong>5 minutes</strong> after it was sent and works only for this request.
        </p>
        <p style="color: #666666; line-height: 1.6; margin-bottom: 20px;">
            If you did not ask for it, change your password: somebody else is signed in as you.
        </p>
        <hr style="border: none; border-top: 1px solid #e9ecef; margin: 30px 0;">
        <p style="color: #999999; font-size: 12px; line-height: 1.6;">
            This is an automated message. Please do not reply to this email.
        </p>
    </div>
</body>
</html>
`, action, code)
}
