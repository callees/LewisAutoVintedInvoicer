package main

import (
	"context"
	"log"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type Item struct {
	ItemName  string  `json:item_name`
	ItemPrice float64 `json:item_price`
}

type Order struct {
	DatabaseID       int64   `json:database_id`
	OrderID          int64   `json:order_id`
	CustomerUsername string  `json:customer_username`
	CustomerEmail    string  `json:customer_email`
	CustomerAddress  string  `json:customer_address`
	ItemsBought      []Item  `json:items_bought`
	DeliveryPrice    float64 `json:delivery_price`
	ProtectionPrice  float64 `json:protection_price`
	TotalPrice       float64 `json:total_price`
	Date             string  `json:order_date`
}

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

	if err := emailClient.c.Login("sales@dealorean.co.uk", "tvtz xadt panf qpza"); err != nil {
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

var testingPanic int = 0

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

func handleOrder(message *imap.Message) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Error initialising mongoDB client")
		}
	}()

	log.Println("Processing Vinted order...")
	body := getBodyString(message)
	var order Order
	// Use the SetServerAPIOptions() method to set the version of the Stable API on the client
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI("mongodb+srv://caltriestocode:Rimming!1@cals-riot-api-data-test.mtghf.mongodb.net/?retryWrites=true&w=majority&appName=cals-riot-api-data-test").SetServerAPIOptions(serverAPI)
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

	defer func() {
		if r := recover(); r != nil {
			log.Println("Error handling order: ", body)
		}
	}()

	order.DatabaseID, _ = client.Database("dealorean").Collection("customer-orders").CountDocuments(ctx, bson.M{}, nil)
	order.OrderID = createOrderId(order.DatabaseID)
	order.CustomerUsername = getBuyerUsername(body)
	order.CustomerEmail = getCustomerEmail(body)
	order.CustomerAddress = getAddress(body)
	pdfFile := getPdfFileString(body, order.OrderID)
	order.ItemsBought = getBoughtItems(body, pdfFile)
	order.DeliveryPrice = getDeliveryPrice(pdfFile)
	order.ProtectionPrice = getProtectionPrice(pdfFile)
	order.TotalPrice = getTotalPrice(pdfFile)
	order.Date = getDate(body)

	saveOrderToDatabase(order)

	sendEmail(order)
}

func main() {
	emailClient := EmailClient{}
	emailClient.start()
	defer emailClient.logout()

	for {
		log.Println("Cecking for new orders...")
		emailClient.checkForNewOrders()
		time.Sleep(1 * time.Minute)
	}

}
