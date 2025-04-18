package send_email

import (
	"fmt"

	"gopkg.in/gomail.v2"
)

func SendEmail(to, code string) error {
	m := gomail.NewMessage()
	m.SetAddressHeader("From", EMAIL_FROM, "小蓝书")
	m.SetHeader("To", to)
	m.SetHeader("Subject", "注册验证码")
	m.SetBody("text/html", fmt.Sprintf("您的验证码是：<h1>%s</h1>请在3分钟内完成注册", code))

	d := gomail.NewDialer("smtp.qq.com", 587, EMAIL_FROM, EMAIL_PASS)
	return d.DialAndSend(m)
}
