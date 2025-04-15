package test

import (
	"regexp"
	"testing"
)

func TestCheckEmail(t *testing.T) {
	patern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	email := "123@qq.com"
	matched, err := regexp.MatchString(patern, email)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(matched)
}
