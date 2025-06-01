package main

import (
	"log"
	"time"
)

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
