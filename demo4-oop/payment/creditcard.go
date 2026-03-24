package payment

import "errors"

type CreditCard[T Float] struct {
	ownerName       string
	cardNumber      string
	expirationMonth int
	expirationYear  int
	securityCode    int
	availableCredit T
}

func NewCreditCard[T Float](ownerName, cardNumber string, expirationMonth, expirationYear, securityCode int, availableCredit T) *CreditCard {

	return &CreditCard[T]{
		ownerName:       ownerName,
		cardNumber:      cardNumber,
		expirationMonth: expirationMonth,
		expirationYear:  expirationYear,
		securityCode:    securityCode,
		availableCredit: availableCredit,
	}
}

func (cc CreditCard[T]) Available() T {
	return cc.availableCredit
}

func (cc *CreditCard[T]) ProcessPayment(amount T) error {
	if cc.availableCredit < amount {
		return errors.New("Insufficient funds to complete payment")
	}

	cc.availableCredit -= amount
	return nil
}
