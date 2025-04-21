package test

import (
	"regexp"
	"testing"
	"xls/app/core/internal/helper"
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

func TestCheckPassword(t *testing.T) {
	password := "123456"
	hashedPwd, err := helper.EncryptPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	matched := helper.CheckPassword(password, hashedPwd)
	if !matched {
		t.Fatal("password unmatched")
	}
}
