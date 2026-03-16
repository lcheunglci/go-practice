package user

import (
	"testing"
	"time"
)

func TestGetOne(t *testing.T) {
	expect := User{
		ID:       42,
		Username: "mrobot",
	}
	users = []User{expect}

	got, err := getOne(expect.ID)

	if err != nil {
		t.Fatal(err)
	}
	if got != expect {
		t.Errorf("did not get expected user. Got %+v, expected %+v", got, expect)
	}
}

func TestSlowOne(t *testing.T) {
	t.Parallel()
	time.Sleep(1 * time.Second)
}
func TestSlowTwo(t *testing.T) {
	t.Parallel()
	time.Sleep(1 * time.Second)
}
