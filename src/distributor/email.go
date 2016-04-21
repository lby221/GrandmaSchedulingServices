package distributor

import (
	"net/smtp"
)

func sendEmailMsg(endpoint string, message string) error {
	auth := smtp.PlainAuth("", "tech@sellyx.com", "sTech123", "smtp.gmail.com")

	to := []string{endpoint}
	msg := []byte("To: " + endpoint + "\r\n" + message)
	err := smtp.SendMail("smtp.gmail.com:587", auth, "donotreply@sellyx.com", to, msg)
	if err != nil {
		return err
	}
	return nil
}
