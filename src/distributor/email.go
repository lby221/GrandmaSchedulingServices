package distributor

import (
	"net/smtp"
)

func sendEmailMsg(endpoint string, message string) error {
	auth := smtp.PlainAuth("", "email", "sTech123", "smtp.gmail.com")

	to := []string{endpoint}
	msg := []byte("To: " + endpoint + "\r\n" + message)
	err := smtp.SendMail("email", auth, "toemail", to, msg)
	if err != nil {
		return err
	}
	return nil
}
