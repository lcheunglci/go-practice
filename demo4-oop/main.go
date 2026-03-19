package main

import (
	"demo4-oop/payment"
	"log"
)

func main() {
	cc := payment.CreditCard{
		OwnerName:       "Bob Doe",
		CardNumber:      "1111-2222-3333-4444",
		ExpirationMonth: 5,
		ExpirationYear:  2026,
		SecurityCode:    123,
		AvailableCredit: 5000,
	}

	err := payment.ProcessPayment(&cc, 10000)
	if err != nil {
		log.Printf("Error processing payment: %v\n", err)
	} else {
		log.Printf("Process payment. Remaining credit: %v\n", cc.AvailableCredit)
	}

	err = payment.ProcessPayment(&cc, 500)
	if err != nil {
		log.Printf("Error processing payment: %v\n", err)
	} else {
		log.Printf("Process payment. Remaining credit: %v\n", cc.AvailableCredit)
	}
}
