package main

import (
	"bytes"
	"encoding/base64"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/dslipak/pdf"
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

	if err := emailClient.c.Login("sales@dealorean.co.uk", "READ FROM ENV"); err != nil {
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
	_, err := emailClient.c.Select("INBOX", false)
	if err != nil {
		log.Fatal(err)
	}
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
			go handleOrder(msg)
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

func handleOrder(message *imap.Message) {
	log.Println("Processing Vinted order...")

	//create orderId

	section := &imap.BodySectionName{}
	r := message.GetBody(section)
	bodyBytes := make([]byte, r.Len())
	_, err := r.Read(bodyBytes)
	if err != nil {
		log.Println("Error reading message body:", err)
		return
	}

	body := string(bodyBytes)
	savePdf(body, 500)

	log.Println("body: " + body)
	log.Println("Buyer username: " + getBuyerUsername(body))
	ah, _ := readPdf("what.pdf")
	log.Println("PDF Content:", ah)
	ahh := getPrice(ah, getBoughtItems(body)[0])
	log.Println(ahh)
	log.Println("Item(s) bought: " + strings.Join(getBoughtItems(body), ", "))
	log.Println("Address: " + getAddress(body))
	log.Println("Customer email: " + getCustomerEmail(body))
}

func readPdf(path string) (string, error) {
	r, err := pdf.Open(path)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	b, err := r.GetPlainText()
	if err != nil {
		return "", err
	}
	buf.ReadFrom(b)
	return buf.String(), nil
}

func savePdf(messageBody string, orderId int) {
	xAttachRegexp, err := regexp.Compile("X-Attachment-Id: (.+?)\r\n")
	xAttachMatches := xAttachRegexp.FindAllStringSubmatch(messageBody, -1)
	xAttachId := xAttachMatches[0][0]

	left := xAttachId
	right := `--`
	rx := regexp.MustCompile(`(?s)` + regexp.QuoteMeta(left) + `(.*?)` + regexp.QuoteMeta(right))
	matches := rx.FindAllStringSubmatch(messageBody, -1)

	base64PDF := strings.ReplaceAll(matches[0][1], xAttachId+"\r\n", "")
	base64PDF = strings.ReplaceAll(base64PDF, "\r\n", "")

	pdfData, err := base64.StdEncoding.DecodeString(base64PDF)
	if err != nil {
		log.Fatalf("Failed to decode base64: %v", err)
	}

	f, _ := os.Create("what.pdf")
	defer f.Close()

	if _, err := f.Write(pdfData); err != nil {
		log.Fatalf("Failed to write PDF data to file: %v", err)
	}
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

func getMultipleItems(messageBody string) []string {
	left := `has bought the following items:`
	right := `We will transfer the buyer`
	rx := regexp.MustCompile(`(?s)` + regexp.QuoteMeta(left) + `(.*?)` + regexp.QuoteMeta(right))
	matches := rx.FindAllStringSubmatch(messageBody, -1)

	itemsString := strings.ReplaceAll(matches[0][1], "\r\n", "")
	items := strings.Split(itemsString, "**")

	for itemIndex := range items {
		items[itemIndex] = strings.ReplaceAll(items[itemIndex], "*", "")
	}
	return items
}

func getBoughtItems(messageBody string) []string {
	multipleItemsRegexp, _ := regexp.Compile("has bought the following items:")

	if multipleItemsRegexp.MatchString(messageBody) {
		return getMultipleItems(messageBody)
	} else {
		boughtItemRegexp, err := regexp.Compile("has bought(.+)")
		if err != nil {
			log.Println("Error compiling regex:", err)
			return []string{""}
		}
		matchedString := boughtItemRegexp.FindString(messageBody)

		boughtItem := strings.Replace(matchedString[:len(matchedString)-3], "has bought *", "", 1)

		return []string{boughtItem}
	}
	return []string{"meh"}
}

func getPrice(pdfFile string, itemName string) float64 {
	priceRegexp := regexp.MustCompile(itemName + "£[0-9]*.[0-9]*")
	matched := priceRegexp.FindStringSubmatch(pdfFile)
	if len(matched) > 0 {
		priceWithItemName := strings.TrimPrefix(matched[0], itemName)
		priceWithItemName = strings.TrimSpace(priceWithItemName)
		priceAsString := strings.ReplaceAll(priceWithItemName, "£", "")
		price, _ := strconv.ParseFloat(priceAsString, 64)
		return price
	}

	return 0.0
}

func getAddress(messageBody string) string {
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

func getCustomerEmail(messageBody string) string {
	addressRegexp := regexp.MustCompile(`>Email:</td>\s*([\s\S]+?)(?:\r?\n\r?\n|$)`)
	matched := addressRegexp.FindStringSubmatch(messageBody)

	left := `a href=3D"mailto:`
	right := `" target`
	rx := regexp.MustCompile(`(?s)` + regexp.QuoteMeta(left) + `(.*?)` + regexp.QuoteMeta(right))
	matches := rx.FindAllStringSubmatch(matched[0], -1)

	email := strings.ReplaceAll(matches[0][1], "=\r\n", "")

	return email
}
