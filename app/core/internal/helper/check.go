package helper

import "regexp"

func CheckEmailFormat(email string) bool {
	patern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(patern, email)
	return matched
}

func CheckPasswordFormat(password string) bool {
	patern := `^[a-zA-Z0-9]{6,16}$`
	matched, _ := regexp.MatchString(patern, password)
	return matched
}
