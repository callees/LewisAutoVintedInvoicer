package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/dslipak/pdf"
	"github.com/emersion/go-imap"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var vintedSalesEmail = "no-reply@vinted.co.uk"

var testEmail = "payments-noreply@google.com"

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

func saveOrderToDatabase(order Order) {
	// Use the SetServerAPIOptions() method to set the version of the Stable API on the client
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI("SECURE THIS").SetServerAPIOptions(serverAPI)
	// Create a new client and connect to the server
	var ctx context.Context
	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err = client.Disconnect(context.TODO()); err != nil {
			panic(err)
		}
	}()
	// Send a ping to confirm a successful connection
	if err := client.Ping(context.TODO(), readpref.Primary()); err != nil {
		panic(err)
	}
	fmt.Println("Pinged your deployment. You successfully connected to MongoDB!")
	collection := client.Database("dealorean").Collection("customer-orders")
	_, errz := collection.InsertOne(nil, order)
	if errz != nil {
		log.Println("Error saving order to database:", errz)
		return //PANIC
	}
	log.Println("Order saved to database with ID:", order.OrderID)
}

func getBodyString(message *imap.Message) string {
	section := &imap.BodySectionName{}
	r := message.GetBody(section)
	bodyBytes := make([]byte, r.Len())
	_, err := r.Read(bodyBytes)
	if err != nil {
		log.Println("Error reading message body:", err)
		return "" //PANIC
	}

	return string(bodyBytes)
}

func createOrderId(databaseId int64) int64 {
	var orderId int64 = (93451*databaseId + 21444) % 99887
	return orderId
}

func getMatchesBetweenTwoStrings(stringToSearch string, leftString string, rightString string) [][]string {
	rx := regexp.MustCompile(`(?s)` + regexp.QuoteMeta(leftString) + `(.*?)` + regexp.QuoteMeta(rightString))
	return rx.FindAllStringSubmatch(stringToSearch, -1)
}

func getDate(messageBody string) string {
	matches := getMatchesBetweenTwoStrings(messageBody, "Received", "X-Received")

	matches = getMatchesBetweenTwoStrings(matches[0][0], "\r\n", "\r\n")

	date := strings.TrimSpace(matches[0][0])
	return date
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

func getBase64PDFString(messageBody string) string {
	xAttachRegexp, _ := regexp.Compile("X-Attachment-Id: (.+?)\r\n")
	xAttachMatches := xAttachRegexp.FindAllStringSubmatch(messageBody, -1)
	xAttachId := xAttachMatches[0][0]

	matches := getMatchesBetweenTwoStrings(messageBody, xAttachId, `--`)

	base64PDF := strings.ReplaceAll(matches[0][1], xAttachId+"\r\n", "")
	base64PDF = strings.ReplaceAll(base64PDF, "\r\n", "")

	return base64PDF
}

func savePdf(messageBody string, orderIdString string) {
	base64PDF := getBase64PDFString(messageBody)

	pdfData, err := base64.StdEncoding.DecodeString(base64PDF)
	if err != nil {
		log.Fatalf("Failed to decode base64: %v", err)
	}

	f, _ := os.Create(orderIdString + ".pdf")
	defer f.Close()

	if _, err := f.Write(pdfData); err != nil {
		log.Fatalf("Failed to write PDF data to file: %v", err)
	}
}

func getPdfFileString(messageBody string, orderId int64) string {
	orderIdString := strconv.FormatInt(orderId, 10)

	savePdf(messageBody, orderIdString)

	pdfFile, _ := readPdf(orderIdString + ".pdf")
	os.Remove(orderIdString + ".pdf")
	return pdfFile
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

func getMultipleItemNames(messageBody string) []string {
	matches := getMatchesBetweenTwoStrings(messageBody, `has bought the following items:`, `We will transfer the buyer`)

	itemsString := strings.ReplaceAll(matches[0][1], "\r\n", "")
	return strings.Split(itemsString, "**")
}

func getMultipleItems(messageBody string, pdfFile string) []Item {
	itemNames := getMultipleItemNames(messageBody)

	var items []Item
	for itemIndex := range itemNames {
		itemNames[itemIndex] = strings.ReplaceAll(itemNames[itemIndex], "*", "")
		var item Item = Item{ItemName: itemNames[itemIndex], ItemPrice: getPrice(pdfFile, itemNames[itemIndex])}
		items = append(items, item)
	}
	return items
}

func getItemName(messageBody string) string {
	boughtItemRegexp, err := regexp.Compile("has bought(.+)")
	if err != nil {
		log.Println("Error compiling regex:", err)
		//panic
	}
	matchedString := boughtItemRegexp.FindString(messageBody)

	return strings.Replace(matchedString[:len(matchedString)-3], "has bought *", "", 1)
}

func getBoughtItems(messageBody string, pdfFile string) []Item {

	multipleItemsRegexp, _ := regexp.Compile("has bought the following items:")

	if multipleItemsRegexp.MatchString(messageBody) {
		return getMultipleItems(messageBody, pdfFile)
	} else {
		var item Item

		item.ItemName = getItemName(messageBody)
		item.ItemPrice = getPrice(pdfFile, item.ItemName)

		return []Item{item}
	}
	//panic
	return []Item{}
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

	//panic
	return 0.0
}

func getDeliveryPrice(pdfFile string) float64 {
	deliveryRegexp := regexp.MustCompile("Shipping£[0-9]*.[0-9]*")
	matched := deliveryRegexp.FindStringSubmatch(pdfFile)
	deliveryPriceString := strings.ReplaceAll(matched[0], "Shipping£", "")
	deliveryPrice, err := strconv.ParseFloat(deliveryPriceString, 64)

	if err != nil {
		panic("TODO PANIC")
	}

	return deliveryPrice
}

func getProtectionPrice(pdfFile string) float64 {
	protectionRegexp := regexp.MustCompile("Buyer Protection Pro£[0-9]*.[0-9]*")
	matched := protectionRegexp.FindStringSubmatch(pdfFile)
	protectionPriceString := strings.ReplaceAll(matched[0], "Buyer Protection Pro£", "")
	protectionPrice, err := strconv.ParseFloat(protectionPriceString, 64)

	if err != nil {
		panic("")
	}

	return protectionPrice
}

func getTotalPrice(pdfFile string) float64 {
	totalPriceRegexp := regexp.MustCompile("Total:£[0-9]*.[0-9]*")
	matched := totalPriceRegexp.FindStringSubmatch(pdfFile)
	totalPriceString := strings.ReplaceAll(matched[0], "Total:£", "")
	totalPrice, err := strconv.ParseFloat(totalPriceString, 64)

	if err != nil {
		panic("TODO PANIC")
	}

	return totalPrice
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
	//panic
	return ""
}

func getCustomerEmail(messageBody string) string {
	addressRegexp := regexp.MustCompile(`>Email:</td>\s*([\s\S]+?)(?:\r?\n\r?\n|$)`)
	matched := addressRegexp.FindStringSubmatch(messageBody)

	matches := getMatchesBetweenTwoStrings(matched[0], `a href=3D"mailto:`, `" target`)

	email := strings.ReplaceAll(matches[0][1], "=\r\n", "")

	return email
}
