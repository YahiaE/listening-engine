package email

import (
	"github.com/wneessen/go-mail"
	"github.com/YahiaE/listening-engine/internal/config"
	"github.com/YahiaE/listening-engine/internal/auth"
	"log"
)

func SendCode(userEmail string) (string, error) {
	message := mail.NewMsg()
	if err := message.From("spotify.listening.engine@gmail.com"); err != nil {
		log.Printf("Failed to set From address: %s", err)
		return "", err
	}
	if err := message.To(userEmail); err != nil {
		log.Printf("Failed to set To address: %s", err)
		return "", err
	}

	message.Subject("Verification Code")
	message.SetImportance(mail.ImportanceHigh) 

	otp, err := auth.GenerateOTP(6)
	if err != nil {
		log.Println("Error generating OTP:", err)
		return "", err
	}
	
	message.SetBodyString(mail.TypeTextPlain, otp)

	user := config.Get("EMAIL")
	pass := config.Get("PASS")
	// dont use fatal!!!
	client, err := mail.NewClient("smtp.gmail.com", mail.WithSMTPAuth(mail.SMTPAuthAutoDiscover),
		mail.WithUsername(user), mail.WithPassword(pass))
	if err != nil {
		log.Printf("failed to create mail client: %s", err)
		return "", err
	}

	if err := client.DialAndSend(message); err != nil {
		log.Printf("failed to send mail: %s", err)
		return "", err
	}

	log.Printf("OTP is sent!")

	return otp, nil
	
}