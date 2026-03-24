package main

import (
	"demo4-oop/payment"
	"fmt"
	"log"
)

type PaymentProcessor[T payment.Float] interface {
	ProcessPayment(amount T) error
}

type Account interface {
	Available() T
}

type PaymentMethod[T payment.Float] interface {
	PaymentProcessor[T]
	Account[T]
}

func main() {
	var pm PaymentMethod[float64] = payment.NewCreditCard[T](
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

	switch m := pm.(type) {
	case *payment.CreditCard[float64]:
		fmt.Printf("CreditCard %T\n", m)
	case *payment.BankAccount[float64]:
		fmt.Printf("BankAccount %T\n", m)
	}
}
