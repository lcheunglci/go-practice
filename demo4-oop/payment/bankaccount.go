package payment

import "errors"

type BankAccount struct {
	ownerName     string
	accountNumber string
	balance       float32
}

func NewBankAccount(ownerName, accountNumber string, balance float32) *BankAccount {
	return &BankAccount{
		ownerName:     ownerName,
		accountNumber: accountNumber,
		balance:       balance,
	}
}

func (ba BankAccount) Available() float32 {
	return ba.balance
}

func (ba *BankAccount) ProcessPayment(amount float32) error {
	if ba.balance >= amount {
		ba.balance -= amount
		return nil
	}

	return errors.New("insufficient funds to complete payment")
}
