package main

import (
	"demo4-oop/payment"
	"log"
)

func main() {
	cc := payment.NewCreditCard(
		"Bob Doe",
		"1111-2222-3333-4444",
		5,
		2026,
		123,
		5000,
	)

	err := cc.ProcessPayment(10000)
	if err != nil {
		log.Printf("Error processing payment: %v\n", err)
	} else {
		log.Printf("Process payment. Remaining credit: %v\n", cc.AvailableCredit())
	}

	err = cc.ProcessPayment(500)
	if err != nil {
		log.Printf("Error processing payment: %v\n", err)
	} else {
		log.Printf("Process payment. Remaining credit: %v\n", cc.AvailableCredit())
	}
}
