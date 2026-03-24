package payment

import "errors"

type Float interface {
	float32 | float64
}

type BankAccount[T Float] struct {
	ownerName     string
	accountNumber string
	balance       T
}

func NewBankAccount[T Float](ownerName, accountNumber string, balance T) *BankAccount {
	return &BankAccount[T]{
		ownerName:     ownerName,
		accountNumber: accountNumber,
		balance:       balance,
	}
}

func (ba BankAccount[T]) Available() T {
	return ba.balance
}

func (ba *BankAccount[T]) ProcessPayment(amount T) error {
	if ba.balance >= amount {
		ba.balance -= amount
		return nil
	}

	return errors.New("insufficient funds to complete payment")
}
