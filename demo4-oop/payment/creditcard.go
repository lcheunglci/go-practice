package payment

import "errors"

type CreditCard struct {
	ownerName       string
	cardNumber      string
	expirationMonth int
	expirationYear  int
	securityCode    int
	availableCredit float32
}

func NewCreditCard(ownerName, cardNumber string, expirationMonth, expirationYear, securityCode int, availableCredit float32) CreditCard {

	return CreditCard{
		ownerName:       ownerName,
		cardNumber:      cardNumber,
		expirationMonth: expirationMonth,
		expirationYear:  expirationYear,
		securityCode:    securityCode,
		availableCredit: availableCredit,
	}
}

func (cc CreditCard) AvailableCredit() float32 {
	return cc.availableCredit
}

func (cc *CreditCard) ProcessPayment(amount float32) error {
	if cc.availableCredit < amount {
		return errors.New("Insufficient funds to complete payment")
	}

	cc.availableCredit -= amount
	return nil
}
