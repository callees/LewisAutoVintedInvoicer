package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/smtp"
	"os"
	"os/exec"
	"strconv"

	dkim "github.com/toorop/go-dkim"
)

func sendEmail(order Order) {
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

	pass := "SECURE THIS SOMEHOW"

	msg := createEmailBody(order)
	err = dkim.Sign(&msg, options)
	if err != nil {
		log.Fatal("DKIM signing failed:", err)
	}

	auth := smtp.PlainAuth("", email, pass, "smtp.gmail.com")
	err = smtp.SendMail("smtp.gmail.com:587", auth,
		email,
		[]string{"lewfog@gmail.com"},
		msg)

	if err != nil {
		log.Fatal("Send failed:", err)
	}

	log.Println("Email sent with DKIM signature.")
}

func createEmailBody(order Order) []byte {
	var body bytes.Buffer

	body.WriteString("From: Dealorean <sales@dealorean.co.uk>\r\n")
	body.WriteString(fmt.Sprintf("To: %s\r\n", "lewfog@gmail.com"))
	body.WriteString("Subject: Your invoice sent from the future\r\n")
	body.WriteString("MIME-Version: 1.0\r\n")
	body.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	body.WriteString("\r\n")

	orderId := strconv.FormatInt(order.OrderID, 10)

	itemJsonData, _ := json.Marshal(order.ItemsBought)
	itemJsonString := string(itemJsonData)
	deliveryPrice := strconv.FormatFloat(order.DeliveryPrice, 'f', 2, 32)
	protectionPrice := strconv.FormatFloat(order.ProtectionPrice, 'f', 2, 32)
	totalPrice := strconv.FormatFloat(order.TotalPrice, 'f', 2, 32)
	cmd := exec.Command("node", "/home/audipitre/repos/LewisAutoInvoicer/mjml-email-gen/gen-html.js", orderId, order.Date, order.CustomerUsername, itemJsonString, deliveryPrice, protectionPrice, totalPrice)

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()

	if err != nil {
		log.Println(stderr.String())
		log.Fatalf("Error executing command: %v", err)
	}
	body.WriteString(out.String())

	return body.Bytes()
}
