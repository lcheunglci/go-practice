package user_test

import (
	"testing"
	"user"
)

func TestGetOne(t *testing.T) {
	// u, err := user.getOne(999); cannot access private
	u, err := user.GetOne(999)
	if u != (user.User{}) {
		t.Error()
	}
	if err == nil {
		t.Error()
	}

}
