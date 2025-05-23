package main

func main() {
	emailClient := EmailClient{}
	emailClient.start()
	emailClient.checkForNewOrders()
}
