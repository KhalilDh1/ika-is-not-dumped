package utils

import (
	// "os"

	"github.com/mailjet/mailjet-apiv3-go"
)

func SendMail(userEmail string, subject string, html string) (bool, error) {
	// publicKey := os.Getenv("EMAIL_API_KEY")
	// privateKey := os.Getenv("EMAIL_SECRET_KEY")

	mailjetClient := mailjet.NewMailjetClient("585b2110d2b4c5e95caa530b623744e4", "fab0d1fda1777ebff1da260d24207bf1")
	messagesInfo := []mailjet.InfoMessagesV31{
		{
			From: &mailjet.RecipientV31{
				Email: "dhminekhalil@gmail.com",
			},
			To: &mailjet.RecipientsV31{
				mailjet.RecipientV31{
					Email: userEmail,
				},
			},
			Subject:  subject,
			HTMLPart: html,
		},
	}

	messages := mailjet.MessagesV31{Info: messagesInfo}
	_, err := mailjetClient.SendMailV31(&messages)
	if err != nil {
		return false, err
	}

	return true, nil
}
