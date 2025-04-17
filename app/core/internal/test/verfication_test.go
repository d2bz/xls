package test

import (
	"testing"
	"xls/app/core/internal/helper"
	"xls/pkg/send_email"
)

// go test -v -run TestGenRandomCode
func TestGenRandomCode(t *testing.T) {
	code, err := helper.GenRandomCode(6)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(code)
}

func TestSendEmail(t *testing.T) {
	to := "3180324210@qq.com"
	code := "123456"
	err := send_email.PostVerificationCode(to, code)
	if err != nil {
		t.Fatal(err)
	}
}
