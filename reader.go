package main

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

var vintedSalesEmail = "no-reply@vinted.co.uk"

var testEmail = "payments-noreply@google.com"

type EmailClient struct {
	c *client.Client
}

func (emailClient *EmailClient) start() {
	log.Println("Starting email reader...")

	log.Println("Connecting to server...")

	var err error
	emailClient.c, err = client.DialTLS("imap.gmail.com:993", nil)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Connected")

	if err := emailClient.c.Login("sales@dealorean.co.uk", "READ FROM ENV); err != nil {
		log.Fatal(err)
	}
	log.Println("Logged in")
}

func (emailClient EmailClient) logout() {
	if err := emailClient.c.Logout(); err != nil {
		log.Fatal(err)
	}
	log.Println("Logged out")
}

func (emailClient *EmailClient) checkForNewOrders() {
	mailbox, err := emailClient.c.Select("INBOX", false)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Mailbox:", mailbox)
	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{"\\Seen"}
	uids, err := emailClient.c.Search(criteria)
	if err != nil {
		log.Println(err)
	}

	if len(uids) <= 0 {
		log.Println("No new orders")
		return
	}

	seqset := new(imap.SeqSet)
	seqset.AddNum(uids...)
	section := &imap.BodySectionName{}
	items := []imap.FetchItem{imap.FetchEnvelope, imap.FetchFlags, imap.FetchInternalDate, section.FetchItem()}
	messages := make(chan *imap.Message)
	done := make(chan error, 1)
	go func() {
		done <- emailClient.c.Fetch(seqset, items, messages)
	}()

	for msg := range messages {
		if /*isEmailFromVinted(msg) && */ isVintedEmailSubject(msg) {
			parseOrderEmail(msg)
		}
	}
}

func isVintedEmailSubject(message *imap.Message) bool {
	if strings.Contains(message.Envelope.Subject, "You’ve sold an item on Vinted") {
		return true
	}
	return false
}

func isEmailFromVinted(message *imap.Message) bool {
	if message.Envelope.From[0].MailboxName == "no-reply" && message.Envelope.From[0].HostName == "vinted.co.uk" {
		return true
	}
	return false
}

func parseOrderEmail(message *imap.Message) {
	log.Println("Processing Vinted order...")

	section := &imap.BodySectionName{}
	r := message.GetBody(section)
	bodyBytes := make([]byte, r.Len())
	_, err := r.Read(bodyBytes)
	if err != nil {
		log.Println("Error reading message body:", err)
		return
	}
	body := string(bodyBytes)
	fmt.Println(body)
	log.Println("User who bought: " + getBuyerUsername(body))
	log.Println("Item bought: " + getBoughtItem(body))
	log.Println("Address: " + getAddress(body))
}

func getBuyerUsername(messageBody string) string {
	usernameRegexp, err := regexp.Compile(".+? has bought")
	if err != nil {
		log.Println("Error compiling regex:", err)
		return ""
	}
	matchedString := usernameRegexp.FindString(messageBody)

	username := strings.Replace(matchedString[1:], "* has bought", "", 1)
	return username
}

func getBoughtItem(messageBody string) string {
	// commaHtmlCode := "=E2=80=99"
	boughtItemRegexp, err := regexp.Compile("has bought(.+)")
	if err != nil {
		log.Println("Error compiling regex:", err)
		return ""
	}
	matchedString := boughtItemRegexp.FindString(messageBody)

	boughtItem := strings.Replace(matchedString[:len(matchedString)-3], "has bought *", "", 1)

	return boughtItem
}

func getAddress(messageBody string) string {
	// This regex captures everything after the marker up to the next double newline or end of string
	addressRegexp := regexp.MustCompile(`Buyer=E2=80=99s contact information\s*([\s\S]+?)(?:\r?\n\r?\n|$)`)
	matched := addressRegexp.FindStringSubmatch(messageBody)
	if len(matched) > 1 {
		address := strings.TrimSpace(matched[1])
		address = strings.ReplaceAll(address, "\r\n", "")
		address = strings.ReplaceAll(address, "*Address:", "")
		if idx := strings.Index(address, "Email"); idx != -1 {
			address = address[:idx]
			address = strings.TrimSpace(address)
		}
		return address
	}
	return ""
}
