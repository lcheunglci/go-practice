package main

import (
	"demo4-oop/payment"
	"log"
)

type PaymentProcessor interface {
	ProcessPayment(amount float32) error
}

type Account interface {
	Available() float32
}

type PaymentMethod interface {
	PaymentProcessor
	Account
}

func main() {
	var pm PaymentMethod = payment.NewCreditCard(
		"Bob Doe",
		"1111-2222-3333-4444",
		5,
		2026,
		123,
		5000,
	)

	err := pm.ProcessPayment(10000)
	if err != nil {
		log.Printf("Error processing payment: %v\n", err)
	} else {
		log.Printf("Process payment. Remaining credit: %v\n", pm.Available())
	}

	pm = payment.NewBankAccount("Bob Doe", "1234", 3500)

	err = pm.ProcessPayment(500)
	if err != nil {
		log.Printf("Error processing payment: %v\n", err)
	} else {
		log.Printf("Process payment. Remaining credit: %v\n", pm.Available())
	}
}
