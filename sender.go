package main

import (
	"bytes"
	"fmt"
	"log"
	"net/smtp"
	"os"

	dkim "github.com/toorop/go-dkim"
)

func send(body string) {
	email := "sales@dealorean.co.uk"
	options := dkim.NewSigOptions()
	privateKey, err := os.ReadFile("dkim-private.key")
	if err != nil {
		log.Fatalf("Unable to read private key file: %v", err)
	}
	options.PrivateKey = privateKey
	options.Domain = "dealorean.co.uk"
	options.Selector = "invoiceapp"
	options.SignatureExpireIn = 3600
	options.BodyLength = 50
	options.Headers = []string{"from", "to", "subject", "mime-version", "content-type"}
	options.AddSignatureTimestamp = true
	options.Canonicalization = "relaxed/relaxed"

	pass := "READ FROM ENV"

	recipient := ""

	msg := createEmailBody(recipient)
	err = dkim.Sign(&msg, options)
	if err != nil {
		log.Fatal("DKIM signing failed:", err)
	}

	auth := smtp.PlainAuth("", email, pass, "smtp.gmail.com")
	err = smtp.SendMail("smtp.gmail.com:587", auth,
		email,
		[]string{recipient},
		msg)

	if err != nil {
		log.Fatal("Send failed:", err)
	}

	log.Println("Email sent with DKIM signature.")
}

func createEmailBody(recipient string) []byte {
	var body bytes.Buffer

	body.WriteString("From: Dealorean <sales@dealorean.co.uk>\r\n")
	body.WriteString(fmt.Sprintf("To: %s\r\n", recipient))
	body.WriteString("Subject: Your invoice sent from the future\r\n")
	body.WriteString("MIME-Version: 1.0\r\n")
	body.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	body.WriteString("\r\n")
	body.WriteString("Hello Lewis\r\nInvoice goes here, but not generated it yet.\r\n")

	return body.Bytes()
}
