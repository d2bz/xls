package send_email

import (
	"fmt"
	"net/smtp"

	"github.com/jordan-wright/email"
)

func PostVerificationCode(emailAddr string, code string) error {
	e := email.NewEmail()
	e.From = fmt.Sprintf("小蓝书 <%s>", EMAIL_FROM)
	e.To = []string{emailAddr}
	e.Subject = "注册验证码"
	e.HTML = []byte(fmt.Sprintf("您的验证码是：<h1>%s</h1>，请在3分钟内完成注册", code))

	smtpHost := "smtp.qq.com:465"
	anth := smtp.PlainAuth("", EMAIL_FROM, EMAIL_PASS, "smtp.qq.com")

	err := e.Send(smtpHost, anth)
	if err != nil {
		return err
	}
	return nil
}
