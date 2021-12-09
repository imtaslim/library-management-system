package handler

import (
	"encoding/base64"
	"fmt"
	"net/http"

	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/go-ozzo/ozzo-validation/is"
	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
)

type Auth struct {
	ID       int    `db:"id"`
	Name     string `db:"name"`
	Email    string `db:"email"`
	Password string `db:"password"`
	Confirm  string
	Old_password  string
	IsAdmin bool `db:"is_admin"`
	Status bool `db:"status"`
	VerifyKey string `db:"verify_key"`
	Error map[string]string
	Messege string
}

func (a *Auth) RegValidate() error {
	return validation.ValidateStruct(a,
		validation.Field(&a.Name,
			validation.Required.Error("The Name Field is Required"),
			validation.Length(3, 0).Error("The Name field must be greater than or equals 3"),
		),
		validation.Field(&a.Email,
			validation.Required.Error("The Email Field is Required"),
			is.Email.Error("The Field Must be an Email"),
		),
		validation.Field(&a.Password,
			validation.Required.Error("The Password Field is Required"),
			validation.Length(8, 0).Error("The Password must have 8 Characters"),
			is.Alphanumeric.Error("The Field Must contains Uppercase, Lowercase and Number"),
		),
		validation.Field(&a.Confirm,
			validation.Required.Error("The Confirm Password Field is Required"),
			validation.Length(8, 0).Error("The Password must have 8 Characters"),
			is.Alphanumeric.Error("The Field Must contains Uppercase, Lowercase and Number"),
		),
	)
}

func (a *Auth) LogValidate() error {
	return validation.ValidateStruct(a,
		validation.Field(&a.Email,
			validation.Required.Error("The Email Field is Required"),
			is.Email.Error("The Field Must be an Email"),
		),
		validation.Field(&a.Password,
			validation.Required.Error("The Password Field is Required"),
			validation.Length(8, 0).Error("The Password must have 8 Characters"),
			is.Alphanumeric.Error("The Field Must contains Uppercase, Lowercase and Number"),
		),
	)
}

func (a *Auth) forValidate() error {
	return validation.ValidateStruct(a,
		validation.Field(&a.Email,
			validation.Required.Error("The Email Field is Required"),
			is.Email.Error("The Field Must be an Email"),
		),
	)
}

func (a *Auth) ResetValidate() error {
	return validation.ValidateStruct(a,
		validation.Field(&a.Password,
			validation.Required.Error("The Password Field is Required"),
			validation.Length(8, 0).Error("The Password must have 8 Characters"),
			is.Alphanumeric.Error("The Field Must contains Uppercase, Lowercase and Number"),
		),
		validation.Field(&a.Confirm,
			validation.Required.Error("The Confirm Password Field is Required"),
			validation.Length(8, 0).Error("The Password must have 8 Characters"),
			is.Alphanumeric.Error("The Field Must contains Uppercase, Lowercase and Number"),
		),
	)
}

type register struct {
	Name     string
	Email    string
	Password string
	Messege string
	Errors   map[string]string
}

var (
	// key must be 16, 24 or 32 bytes long (AES-128, AES-192 or AES-256)
	key   = []byte("super-secret-key")
	store = sessions.NewCookieStore(key)
)

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (h *Handler) register(rw http.ResponseWriter, r *http.Request) {

	vErrs := map[string]string{}

	h.createAuthData(rw, "", "", vErrs)
}

func (h *Handler) registerpro(rw http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	var register Auth
	if err := h.decoder.Decode(&register, r.PostForm); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := register.RegValidate(); err != nil {
		valError, ok := err.(validation.Errors)
		if ok {
			vErrs := make(map[string]string)
			for key, value := range valError {
				vErrs[key] = value.Error()
			}
			h.createAuthData(rw, register.Name, register.Email, vErrs)
			return
		}
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	if register.Password != register.Confirm {
		vErrs := map[string]string{
			"Confirm": "The Password Confirmation Doesn't Match",
		}
		h.createAuthData(rw, register.Name, register.Email, vErrs)
		return
	}

	HP, err := HashPassword(register.Password)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	s := register.Email
	se := base64.StdEncoding.EncodeToString([]byte(s))

	email_verified := se

	const insertCategories = `INSERT INTO users(name, email, password, is_admin, status, verify_key) VALUES ($1, $2, $3, $4, $5, $6)`
	res := h.db.MustExec(insertCategories, register.Name, register.Email, HP, false, false, email_verified)

	if ok, err := res.RowsAffected(); err != nil || ok == 0 {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	link := (fmt.Sprintf("http://localhost:3000/verify?token=%s", email_verified))
	email(register.Email, register.Name, link, "Verify Your Account")

	session, err := store.Get(r, "library")
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	session.AddFlash("Registration Successful, Check Your Email to Verify and Login!")
	err = session.Save(r, rw)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(rw, r, "/login", http.StatusTemporaryRedirect)
}

func (h *Handler) createAuthData(rw http.ResponseWriter, name string, email string, errs map[string]string) {
	form := register{
		Name:   name,
		Email:  email,
		Errors: errs,
	}
	if err := h.templates.ExecuteTemplate(rw, "register.html", form); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *Handler) login(rw http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "library")
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	var msg string
	if flashes := session.Flashes(); len(flashes) > 0 {
		if val, ok := flashes[0].(string); ok {
			msg = val
		}
	}
	err = session.Save(r, rw)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	vErrs := map[string]string{}

	h.createLoginData(rw, "", msg, vErrs)
}

func (h *Handler) loginpro(rw http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	var login Auth
	if err := h.decoder.Decode(&login, r.PostForm); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := login.LogValidate(); err != nil {
		valError, ok := err.(validation.Errors)
		if ok {
			vErrs := make(map[string]string)
			for key, value := range valError {
				vErrs[key] = value.Error()
			}
			h.createLoginData(rw, login.Email, "", vErrs)
			return
		}
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	const getuser = `SELECT * FROM users WHERE email=$1`
	var user Auth
	h.db.Get(&user, getuser, login.Email)

	if user.ID == 0 {
		vErrs := map[string]string{
			"Email": "Login Credencial Doesn't Match",
		}
		h.createLoginData(rw, login.Email, "", vErrs)
		return
	}

	if !user.Status {
		vErrs := map[string]string{
			"Email": "Email is not Verified, Please check email",
		}
		h.createLoginData(rw, login.Email, "", vErrs)
		return
	}

	HP := CheckPasswordHash(login.Password, user.Password)

	if !HP {
		vErrs := map[string]string{
			"Email": "Login Credencial Doesn't Match",
		}
		h.createLoginData(rw, login.Email, "", vErrs)
		return
	}

	session, _ := store.Get(r, "library")

	session.Options.HttpOnly = true

	session.Values["authenticated"] = true
	session.Values["id"] = user.ID
	session.Save(r, rw)

	if user.IsAdmin {
		http.Redirect(rw, r, "/admin", http.StatusTemporaryRedirect)
	}else{
		http.Redirect(rw, r, "/", http.StatusTemporaryRedirect)
	}
}

func (h *Handler) createLoginData(rw http.ResponseWriter, email string, msg string, errs map[string]string) {
	form := register{
		Email:  email,
		Errors: errs,
		Messege: msg,
	}
	if err := h.templates.ExecuteTemplate(rw, "login.html", form); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *Handler) logout(rw http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "library")

	// Revoke users authentication
	session.Values["authenticated"] = false
	session.Values["id"] = 0
	session.Save(r, rw)

	http.Redirect(rw, r, "/login", http.StatusTemporaryRedirect)
}

func (h *Handler) verify(rw http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(rw, "Server Error", http.StatusInternalServerError)
	}

	var user Auth

	h.db.Get(&user, "SELECT * FROM users WHERE verify_key=$1", token)

	if user.ID == 0 {
		http.Redirect(rw, r, "/login", http.StatusTemporaryRedirect)
		return
	}

	const updateStatusBooks = `UPDATE users SET status = true WHERE id=$1`
	res := h.db.MustExec(updateStatusBooks, user.ID)

	if ok, err := res.RowsAffected(); err != nil || ok == 0 {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	session, _ := store.Get(r, "library")

	session.Options.HttpOnly = true

	session.Values["authenticated"] = true
	session.Values["id"] = user.ID
	session.Save(r, rw)

	if user.IsAdmin {
		http.Redirect(rw, r, "/admin", http.StatusTemporaryRedirect)
	}else{
		http.Redirect(rw, r, "/", http.StatusTemporaryRedirect)
	}
}

func (h *Handler) sendEmail (rw http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "library")
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	var msg string
	if flashes := session.Flashes(); len(flashes) > 0 {
		if val, ok := flashes[0].(string); ok {
			msg = val
		}
	}
	err = session.Save(r, rw)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	var forget Auth
	forget.Messege = msg
	if err := h.templates.ExecuteTemplate(rw, "forget-password.html", forget); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *Handler) sendEmailProcess (rw http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	var forget Auth
	if err := h.decoder.Decode(&forget, r.PostForm); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := forget.forValidate(); err != nil {
		valError, ok := err.(validation.Errors)
		if ok {
			vErrs := make(map[string]string)
			for key, value := range valError {
				vErrs[key] = value.Error()
			}
			forget.Error = vErrs
			if err := h.templates.ExecuteTemplate(rw, "forget-password.html", forget); err != nil {
				http.Error(rw, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		return
	}

	var user Auth
	h.db.Get(&user, "SELECT * FROM users WHERE email=$1", forget.Email)

	if user.ID == 0 {
		http.Redirect(rw, r, "/login", http.StatusTemporaryRedirect)
		return
	}

	link := (fmt.Sprintf("http://localhost:3000/forget-password?token=%s", user.VerifyKey))
	email(user.Email, user.Name, link, "Change Your Password")

	session, err := store.Get(r, "library")
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	session.AddFlash("Email successfully send, if You don't get it, Send Again!")
	err = session.Save(r, rw)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(rw, r, "/send-email", http.StatusTemporaryRedirect)
}

func (h *Handler) forgetPassword (rw http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(rw, "Server Error", http.StatusInternalServerError)
	}

	var user Auth

	h.db.Get(&user, "SELECT id FROM users WHERE verify_key=$1", token)

	if user.ID == 0 {
		http.Redirect(rw, r, "/login", http.StatusTemporaryRedirect)
		return
	}

	if err := h.templates.ExecuteTemplate(rw, "reset-password.html", user); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *Handler) resetPassword (rw http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	var reset Auth
	if err := h.decoder.Decode(&reset, r.PostForm); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := reset.ResetValidate(); err != nil {
		valError, ok := err.(validation.Errors)
		if ok {
			vErrs := make(map[string]string)
			for key, value := range valError {
				vErrs[key] = value.Error()
			}
			reset.Error = vErrs
			if err := h.templates.ExecuteTemplate(rw, "reset-password.html", reset); err != nil {
				http.Error(rw, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		return
	}

	const getuser = `SELECT * FROM users WHERE id=$1`
	var user Auth
	h.db.Get(&user, getuser, reset.ID)

	if user.ID == 0 {
		http.Redirect(rw, r, "/", http.StatusTemporaryRedirect)
		return
	}

	if reset.Password != reset.Confirm {
		reset.Error = map[string]string{
			"Confirm": "The Password Confirmation Doesn't Match",
		}
		if err := h.templates.ExecuteTemplate(rw, "reset-password.html", reset); err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	HP, err := HashPassword(reset.Password)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	const updateStatusBooks = `UPDATE users SET password = $2 WHERE id=$1`
	res := h.db.MustExec(updateStatusBooks, user.ID, HP)

	if ok, err := res.RowsAffected(); err != nil || ok == 0 {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	session, _ := store.Get(r, "library")

	session.Options.HttpOnly = true

	session.Values["authenticated"] = true
	session.Values["id"] = user.ID
	session.Save(r, rw)

	if user.IsAdmin {
		http.Redirect(rw, r, "/admin", http.StatusTemporaryRedirect)
	}else{
		http.Redirect(rw, r, "/", http.StatusTemporaryRedirect)
	}
}