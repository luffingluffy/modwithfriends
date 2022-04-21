package smtp

import (
	"fmt"
	"net/smtp"
	"strconv"
)

type EmailClient struct {
	senderEmail string
	smtpAddr    string
	smtpAuth    smtp.Auth
}

func NewEmailClient(email string, password string, hostname string, port int) *EmailClient {
	return &EmailClient{
		senderEmail: email,
		smtpAddr:    hostname + ":" + strconv.Itoa(port),
		smtpAuth:    smtp.PlainAuth("", email, password, hostname),
	}
}

func (ec *EmailClient) Send(subject string, recipients []string, message string) error {
	content := "To: "
	for index, recipient := range recipients {
		content += fmt.Sprintf("%s", recipient)
		if index != len(recipients)-1 {
			content += ","
		}
	}
	content += fmt.Sprintf("\r\nSubject: %s\r\n\r\n%s\r\n", subject, message)

	err := smtp.SendMail(ec.smtpAddr, ec.smtpAuth, ec.senderEmail, recipients, []byte(content))
	if err != nil {
		return fmt.Errorf("Failed to send out email: %w", err)
	}
	return nil
}
