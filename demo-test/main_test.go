package main

import "testing"

func TestAdd(t *testing.T) {
	l, r := 1, 2
	expect := 3

	got := add(l, r)

	if expect != got {
		t.Errorf("Expected %v when adding %v and %v. Got %v\n", expect, 1, r, got)
	}
}
