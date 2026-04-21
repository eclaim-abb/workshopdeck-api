package email

import (
	"bytes"
	"fmt"
	"html/template"
	"time"

	"eclaim-workshop-deck-api/internal/config"

	gomail "gopkg.in/gomail.v2"
)

type EmailService struct{}
type PermissionStruct struct {
	ActionPage        string
	ActionName        string
	ActionDescription string
}

func NewEmailService() *EmailService {
	return &EmailService{}
}

func emailSender(body bytes.Buffer, subject string, toEmail string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", fmt.Sprintf("%s <%s>", config.EmailData.SMTP.Name, config.EmailData.SMTP.User))
	m.SetHeader("To", toEmail)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body.String())

	d := gomail.NewDialer(
		config.EmailData.SMTP.Server,
		int(config.EmailData.SMTPPort),
		config.EmailData.SMTP.User,
		config.EmailData.SMTP.Pass,
	)

	return d.DialAndSend(m)
}

func (s *EmailService) SendResetEmail(to, username, newPassword string) error {
	tmpl, err := template.ParseFiles("templates/reset_password.html")
	if err != nil {
		return err
	}

	data := struct {
		Username string
		Password string
		Year     int
	}{
		Username: username,
		Password: newPassword,
		Year:     time.Now().Year(),
	}

	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		return err
	}

	err = emailSender(body, "Workshop Deck - Your New Password", to)

	return err
}

func (s *EmailService) SendCreatedUser(toEmail, username, email, newPassword string) error {
	tmpl, err := template.ParseFiles("templates/new_user.html")
	if err != nil {
		return err
	}

	data := struct {
		Username string
		Email    string
		Password string
		Year     int
	}{
		Username: username,
		Email:    email,
		Password: newPassword,
		Year:     time.Now().Year(),
	}

	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		return err
	}

	err = emailSender(body, "Workshop Deck - An account has been created for you", toEmail)
	return err
}

func (s *EmailService) SendChangedPassword(toEmail, username string) error {
	tmpl, err := template.ParseFiles("templates/change_password.html")
	if err != nil {
		return err
	}

	now := time.Now().Local()

	data := struct {
		Username string
		Email    string
		Date     string
		Time     string
	}{
		Username: username,
		Email:    toEmail,
		Date:     now.Format("02 Jan 2006"),
		Time:     now.Format("15:04"),
	}

	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		return err
	}

	err = emailSender(body, "Workshop Deck - Notice on changed password", toEmail)

	return err
}

func (s *EmailService) SendUpdatedAccount(toEmail, username, userId string, emailChanged, usernameChanged, passwordChanged bool) error {
	tmpl, err := template.ParseFiles("templates/updated_account.html")
	if err != nil {
		return err
	}

	now := time.Now().Local()

	data := struct {
		Username        string
		EmailChanged    bool
		NewEmail        string
		UsernameChanged bool
		NewUsername     string
		PasswordChanged bool
		Date            string
		Time            string
	}{
		Username:        username,
		EmailChanged:    emailChanged,
		NewEmail:        toEmail,
		UsernameChanged: usernameChanged,
		NewUsername:     userId,
		PasswordChanged: passwordChanged,
		Date:            now.Format("02 Jan 2006"),
		Time:            now.Format("15:04"),
	}

	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		return err
	}

	err = emailSender(body, "Notice on updated account", toEmail)

	return err
}

func (s *EmailService) SendWelcome(name, email, username string) error {
	tmpl, err := template.ParseFiles("templates/welcome.html")
	if err != nil {
		return err
	}

	now := time.Now().Local()

	data := struct {
		Username string
		Name     string
		Email    string
		Date     string
		Year     int
	}{
		Username: username,
		Name:     name,
		Email:    email,
		Date:     now.Format("02 Jan 2006"),
		Year:     now.Year(),
	}

	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		return err
	}

	err = emailSender(body, "Workshop Deck - Welcome!", email)

	return err
}

func (s *EmailService) SendRoleChange(toEmail, oldRole, newRole, name, modifierName string, permissions []PermissionStruct) error {
	tmpl, err := template.ParseFiles("templates/changed_role.html")
	if err != nil {
		return err
	}

	now := time.Now().Local()

	data := struct {
		Username    string
		OldRole     string
		NewRole     string
		Permissions []PermissionStruct
		Date        string
		Time        string
		Year        int
		ModifiedBy  string
	}{
		Username:    name,
		OldRole:     oldRole,
		NewRole:     newRole,
		Permissions: permissions,
		Date:        now.Format("02 Jan 2006"),
		Time:        now.Format("15:04"),
		Year:        now.Year(),
		ModifiedBy:  modifierName,
	}

	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		return err
	}

	err = emailSender(body, "Workshop Deck - Role Change Notification!", toEmail)

	return err
}

func (s *EmailService) Send2FA(toEmail, username, token string) error {
	tmpl, err := template.ParseFiles("templates/send2fa.html")
	if err != nil {
		return err
	}

	now := time.Now().Local()

	data := struct {
		Username string
		Email    string
		Date     string
		Time     string
		Token    string
		Year     int
	}{
		Username: username,
		Email:    toEmail,
		Date:     now.Format("02 Jan 2006"),
		Time:     now.Format("15:04"),
		Token:    token,
		Year:     now.Year(),
	}

	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		return err
	}

	err = emailSender(body, "Workshop Deck - Your 2FA Token", toEmail)

	return err
}
