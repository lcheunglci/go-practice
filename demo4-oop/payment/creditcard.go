package payment

import "errors"

type CreditCard struct {
	OwnerName       string
	CardNumber      string
	ExpirationMonth int
	ExpirationYear  int
	SecurityCode    int
	AvailableCredit float32
}

func (cc *CreditCard) ProcessPayment(amount float32) error {
	if cc.AvailableCredit < amount {
		return errors.New("Insufficient funds to complete payment")
	}

	cc.AvailableCredit -= amount
	return nil
}
