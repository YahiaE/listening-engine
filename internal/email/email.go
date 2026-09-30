package email

import (
	"github.com/wneessen/go-mail"
	"github.com/YahiaE/listening-engine/internal/config"
	"github.com/YahiaE/listening-engine/internal/auth"
	"log"
)

func SendCode(string email) (string, error) {
	message := mail.NewMsg()
	if err := message.From("spotify.listening.engine@gmail.com"); err != nil {
		log.Printf("Failed to set From address: %s", err)
		return nil, err
	}
	if err := message.To(email); err != nil {
		log.Printf("Failed to set To address: %s", err)
		return nil, err
	}

	message.Subject("Verification Code")

	otp, err := auth.GenerateOTP(6)
	if err != nil {
		log.Println("Error generating OTP:", err)
		return nil, err
	}
	
	message.SetBodyString(mail.TypeTextPlain, otp)

	user := config.Get("USER")
	pass := config.Get("PASS")
	// dont use fatal!!!
	client, err := mail.NewClient("smtp.gmail.com", mail.WithSMTPAuth(mail.SMTPAuthAutoDiscover),
		mail.WithUsername(user), mail.WithPassword(pass))
	if err != nil {
		log.Printf("failed to create mail client: %s", err)
		return nil, err
	}

	if err := client.DialAndSend(message); err != nil {
		log.Printf("failed to send mail: %s", err)
		return nil, err
	}
	
}