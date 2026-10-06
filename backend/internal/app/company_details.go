package app

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

type CompanyDetails struct {
	Tagline       string   `json:"tagline" maxLength:"160" required:"false"`
	Industry      string   `json:"industry" maxLength:"100" required:"false"`
	LegalName     string   `json:"legal_name" maxLength:"200" required:"false"`
	INN           string   `json:"inn" maxLength:"12" required:"false"`
	City          string   `json:"city" maxLength:"100" required:"false"`
	Address       string   `json:"address" maxLength:"300" required:"false"`
	FoundedYear   int      `json:"founded_year" minimum:"0" maximum:"2100" required:"false"`
	TeamSize      string   `json:"team_size" enum:",1-10,11-50,51-200,201-500,501-1000,1000+" required:"false"`
	Products      string   `json:"products" maxLength:"3000" required:"false"`
	Culture       string   `json:"culture" maxLength:"3000" required:"false"`
	HiringProcess string   `json:"hiring_process" maxLength:"3000" required:"false"`
	Technologies  []string `json:"technologies" maxItems:"30" required:"false"`
	Benefits      []string `json:"benefits" maxItems:"15" required:"false"`
	WorkFormats   []string `json:"work_formats" maxItems:"3" required:"false"`
	CareersURL    string   `json:"careers_url" maxLength:"500" required:"false"`
	ContactName   string   `json:"contact_name" maxLength:"100" required:"false"`
	ContactRole   string   `json:"contact_role" maxLength:"100" required:"false"`
	ContactEmail  string   `json:"contact_email" maxLength:"254" required:"false"`
	Phone         string   `json:"phone" maxLength:"40" required:"false"`
	Telegram      string   `json:"telegram" maxLength:"32" required:"false"`
}

type Contacts struct {
	Phone    string `json:"phone" maxLength:"40" required:"false"`
	Telegram string `json:"telegram" maxLength:"32" required:"false"`
}

var phonePattern = regexp.MustCompile(`^\+?[0-9 ()-]{7,40}$`)
var telegramPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{4,31}$`)
var innPattern = regexp.MustCompile(`^([0-9]{10}|[0-9]{12})$`)

func (c *Contacts) clean() error {
	c.Phone = strings.TrimSpace(c.Phone)
	c.Telegram = strings.TrimPrefix(strings.TrimSpace(c.Telegram), "@")
	if c.Phone != "" && !phonePattern.MatchString(c.Phone) {
		return fmt.Errorf("Проверьте телефон: укажите номер с кодом страны")
	}
	digits := 0
	for _, r := range c.Phone {
		if r >= '0' && r <= '9' {
			digits++
		}
	}
	if c.Phone != "" && (digits < 7 || digits > 15) {
		return fmt.Errorf("Телефон должен содержать от 7 до 15 цифр")
	}
	if c.Telegram != "" && !telegramPattern.MatchString(c.Telegram) {
		return fmt.Errorf("Укажите имя пользователя Telegram, например username, без ссылки")
	}
	return nil
}

func cleanWebURL(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	u, err := url.Parse(value)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Hostname() == "" || u.User != nil || utf8.RuneCountInString(value) > 500 {
		return "", fmt.Errorf("Укажите полную ссылку с https:// или http:// без логина и пароля")
	}
	return u.String(), nil
}

func cleanCompanyList(values []string, limit int) ([]string, error) {
	result := []string{}
	seen := map[string]bool{}
	for _, v := range values {
		v = strings.TrimSpace(v)
		if utf8.RuneCountInString(v) > 100 {
			return nil, fmt.Errorf("Элемент списка не должен превышать 100 символов")
		}
		key := strings.ToLower(v)
		if v != "" && !seen[key] {
			result = append(result, v)
			seen[key] = true
		}
	}
	if len(result) > limit {
		return nil, fmt.Errorf("В списке допускается не более %d элементов", limit)
	}
	return result, nil
}

func (d *CompanyDetails) clean() error {
	d.INN = strings.TrimSpace(d.INN)
	if d.INN != "" && !innPattern.MatchString(d.INN) {
		return fmt.Errorf("ИНН должен содержать 10 или 12 цифр")
	}
	if d.FoundedYear != 0 && (d.FoundedYear < 1900 || d.FoundedYear > time.Now().Year()) {
		return fmt.Errorf("Проверьте год основания компании")
	}
	contacts := Contacts{Phone: d.Phone, Telegram: d.Telegram}
	if err := contacts.clean(); err != nil {
		return err
	}
	d.Phone, d.Telegram = contacts.Phone, contacts.Telegram
	d.ContactEmail = strings.TrimSpace(d.ContactEmail)
	if d.ContactEmail != "" && !validEmail(d.ContactEmail) {
		return fmt.Errorf("Проверьте рабочую почту для связи")
	}
	var err error
	if d.CareersURL, err = cleanWebURL(d.CareersURL); err != nil {
		return err
	}
	if d.Technologies, err = cleanCompanyList(d.Technologies, 30); err != nil {
		return err
	}
	if d.Benefits, err = cleanCompanyList(d.Benefits, 15); err != nil {
		return err
	}
	if d.WorkFormats, err = cleanCompanyList(d.WorkFormats, 3); err != nil {
		return err
	}
	for _, v := range d.WorkFormats {
		if v != "remote" && v != "hybrid" && v != "office" {
			return fmt.Errorf("Неизвестный формат работы")
		}
	}
	return nil
}
