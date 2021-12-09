package handler

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"net/smtp"

	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"github.com/jmoiron/sqlx"
	// "github.com/gorilla/sessions"
)

type Handler struct{
	templates *template.Template
	db *sqlx.DB
	decoder *schema.Decoder
}

func New(db *sqlx.DB, decoder *schema.Decoder) *mux.Router {
	h := &Handler{
		db: db,
		decoder: decoder,
	}
	
	h.parseTemplates()

	r := mux.NewRouter()
	r.HandleFunc("/logout", h.logout)

	r.HandleFunc("/", h.home)
	// r.HandleFunc("/search", h.search)

	l := r.NewRoute().Subrouter()
	l.HandleFunc("/register", h.register)
	l.HandleFunc("/register-process", h.registerpro)
	l.HandleFunc("/login", h.login)
	l.HandleFunc("/login-process", h.loginpro)
	l.HandleFunc("/verify", h.verify)
	l.HandleFunc("/send-email", h.sendEmail)
	l.HandleFunc("/send-email-process", h.sendEmailProcess)
	l.HandleFunc("/forget-password", h.forgetPassword)
	l.HandleFunc("/reset-password", h.resetPassword)
	l.Use(h.loginMiddleware)

	s := r.NewRoute().Subrouter()
	s.HandleFunc("/bookingProcess", h.bookingProcess)
	s.HandleFunc("/my-bookings", h.bookings)
	s.HandleFunc("/profile", h.profile)
	s.HandleFunc("/updateProfile", h.updateProfile)
	s.HandleFunc("/changePassword", h.changePassword)
	s.HandleFunc("/details/{id:[0-9]+}", h.bookDetails)
	s.Use(h.authMiddleware)

	a := r.NewRoute().Subrouter()
	a.HandleFunc("/admin", h.adminHome)
	a.HandleFunc("/categories", h.categoriesHome)
	a.HandleFunc("/categories/create", h.categoriesCreate)
	a.HandleFunc("/categories/store", h.categoriesStore)
	a.HandleFunc("/categories/{id:[0-9]+}/{status:[0-9]+}/complete", h.categoriesComplete)
	a.HandleFunc("/categories/{id:[0-9]+}/edit", h.categoriesEdit)
	a.HandleFunc("/categories/{id:[0-9]+}/update", h.categoriesUpdate)
	a.HandleFunc("/categories/{id:[0-9]+}/delete", h.categoriesDelete)
	a.HandleFunc("/books", h.booksHome)
	a.HandleFunc("/books/create", h.booksCreate)
	a.HandleFunc("/books/store", h.booksStore)
	a.HandleFunc("/books/{id:[0-9]+}/{status:[0-9]+}/complete", h.bookComplete)
	a.HandleFunc("/books/{id:[0-9]+}/edit", h.bookEdit)
	a.HandleFunc("/books/{id:[0-9]+}/update", h.bookUpdate)
	a.HandleFunc("/books/{id:[0-9]+}/delete", h.bookDelete)
	a.HandleFunc("/users", h.adminUsers)
	a.HandleFunc("/change-admin-password", h.adminPassword)
	a.HandleFunc("/changeAdminPassword", h.adminPasswordPro)
	a.HandleFunc("/bookinglist", h.adminBooking)
	a.HandleFunc("/changeRoles/{id:[0-9]+}/{utype}", h.changeRoles)
	a.Use(h.adminMiddleware)

	r.PathPrefix("/asset/").Handler(http.StripPrefix("/asset/", http.FileServer(http.Dir("./"))))

	r.NotFoundHandler = http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if err := h.templates.ExecuteTemplate(rw, "404.html", nil); err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	return r
}

func (h *Handler) parseTemplates() {
	h.templates = template.Must(template.ParseFiles(
		"templates/admin/admin.html",
		"templates/admin/category/create-categories.html",
		"templates/admin/category/list-categories.html",
		"templates/admin/category/edit-categories.html",
		"templates/admin/book/list-books.html",
		"templates/admin/book/create-books.html",
		"templates/admin/book/edit-books.html",
		"templates/admin/users/booking-list.html",
		"templates/admin/users/user-list.html",
		"templates/admin/change-password.html",

		"templates/book-details.html",
		"templates/404.html",
		"templates/home.html",
		"templates/my-booking.html",
		"templates/my-profile.html",
		"templates/auth/register.html",
		"templates/auth/login.html",
		"templates/auth/forget-password.html",
		"templates/auth/reset-password.html",
	))
}

func email(usermail string, name string, link string, subject string) {
	// Sender data.
	from := "give your Email here"
	password := "give your password here"
  
	// Receiver email address.
	to := []string{
		usermail,
	}
  
	// smtp server configuration.
	smtpHost := "smtp.gmail.com"
	smtpPort := "587"
  
	// Authentication.
	auth := smtp.PlainAuth("Library Management System", from, password, smtpHost)
  
	t, _ := template.ParseFiles("templates/template.html")
  
	var body bytes.Buffer
  
	mimeHeaders := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	body.Write([]byte(fmt.Sprintf("Subject: %s\n%s\n\n", subject, mimeHeaders)))
  
	err := t.Execute(&body, struct {
	  Name    string
	  Link string
	}{
	  Name:    name,
	  Link:		link,
	})

	if err != nil {
		fmt.Println(err)
		return
	  }
  
	// Sending email.
	
	if err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, to, body.Bytes()); err != nil {
	  fmt.Println(err)
	  return
	}
	fmt.Println("Email Sent!")
}