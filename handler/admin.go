package handler

import (
	"fmt"
	"math"
	"net/http"
	"strconv"

	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/gorilla/mux"
)

type Userlist struct{
	Users []Auth
	Messege string
	Pagination []Pagination
	TotalPage int
	CurrentPage int
	PrePageURL string
	NextPageURL string
	Search string
}

func (h *Handler) adminUsers(rw http.ResponseWriter, r *http.Request) {
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

	userlist := []Auth{}
	var p int = 1
	var errr error
	var nextPageURL string
	var prePageURL string

	page := r.URL.Query().Get("page")
	search := r.URL.Query().Get("search")
	if page != "" {
		p, errr = strconv.Atoi(page)
	}

	pageQuery := fmt.Sprintf("&search=%s", search)

	if errr != nil {
		http.Error(rw, errr.Error(), http.StatusInternalServerError)
		return
	}
	offset := 0
	limit := 7

	if p > 0 {
		offset = limit*p - limit
	}

	total := 0
	h.db.Get(&total, "SELECT count(*) FROM users WHERE NOT id = 1 AND (name ILIKE '%%' || $1 || '%%'  OR email ILIKE '%%' || $1 || '%%')", search)
	h.db.Select(&userlist, "SELECT * FROM users WHERE NOT id = 1 AND (name ILIKE '%%' || $1 || '%%'  OR email ILIKE '%%' || $1 || '%%') ORDER BY id OFFSET $2 LIMIT $3", search, offset, limit)

	totalPage := int(math.Ceil(float64(total) / float64(limit)))
	pagination := make([]Pagination, totalPage)
	for i := 0; i < totalPage; i++ {
		pagination[i] = Pagination{
			URL:    fmt.Sprintf("http://localhost:3000/users?page=%d%s", i+1, pageQuery),
			PageNo: i + 1,
		}
		if i+1 == p {
			if i != 0 {
				prePageURL = fmt.Sprintf("http://localhost:3000/users?page=%d%s", i, pageQuery)
			}
			if i+1 != totalPage {
				nextPageURL = fmt.Sprintf("http://localhost:3000/users?page=%d%s", i+2, pageQuery)
			}
		}
	}

	lb := Userlist{
		Users: userlist,
		Pagination:  pagination,
		TotalPage:   totalPage,
		CurrentPage: p,
		PrePageURL:  prePageURL,
		NextPageURL: nextPageURL,
		Messege: msg,
		Search: search,
	}
	if err := h.templates.ExecuteTemplate(rw, "user-list.html", lb); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *Handler) adminBooking(rw http.ResponseWriter, r *http.Request) {
	bookings := []Bookings{}
	var p int = 1
	var errr error
	var nextPageURL string
	var prePageURL string

	page := r.URL.Query().Get("page")
	if page != "" {
		p, errr = strconv.Atoi(page)
	}

	userID := r.URL.Query().Get("user")

	if errr != nil {
		http.Error(rw, errr.Error(), http.StatusInternalServerError)
		return
	}
	offset := 0
	limit := 7

	if p > 0 {
		offset = limit*p - limit
	}

	total := 0
	if userID != "" {
		h.db.Get(&total, "SELECT count(*) FROM bookings WHERE user_id=$1", userID)
		h.db.Select(&bookings, "SELECT * FROM bookings WHERE user_id=$3 ORDER BY id DESC OFFSET $1 LIMIT $2", offset, limit, userID)
	}else{
		h.db.Get(&total, "SELECT count(*) FROM bookings")
		h.db.Select(&bookings, "SELECT * FROM bookings ORDER BY id DESC OFFSET $1 LIMIT $2", offset, limit)	
	}

	totalPage := int(math.Ceil(float64(total) / float64(limit)))
	pagination := make([]Pagination, totalPage)
	for i := 0; i < totalPage; i++ {
		pagination[i] = Pagination{
			URL:    fmt.Sprintf("http://localhost:3000/bookinglist?page=%d", i+1),
			PageNo: i + 1,
		}
		if i+1 == p {
			if i != 0 {
				prePageURL = fmt.Sprintf("http://localhost:3000/bookinglist?page=%d", i)
			}
			if i+1 != totalPage {
				nextPageURL = fmt.Sprintf("http://localhost:3000/bookinglist?page=%d", i+2)
			}
		}
	}

	for key, value := range bookings {
		const getTodo = `SELECT name, status FROM books WHERE id=$1`
		var book Book
		h.db.Get(&book, getTodo, value.Book_id)

		const getUsers = `SELECT name FROM users WHERE id=$1`
		var user Auth
		h.db.Get(&user, getUsers, value.User_id)

		start_time := value.Start_time.Format("Mon Jan _2 2006 15:04 AM")
		end_time := value.End_time.Format("Mon Jan _2 2006 15:04 AM")

		bookings[key].Book_name = book.Name
		bookings[key].User_name = user.Name
		bookings[key].ST = start_time
		bookings[key].ET = end_time
	}

	lb := ListBooking{
		Bookings:    bookings,
		Pagination:  pagination,
		TotalPage:   totalPage,
		CurrentPage: p,
		PrePageURL:  prePageURL,
		NextPageURL: nextPageURL,
	}
	if err := h.templates.ExecuteTemplate(rw, "booking-list.html", lb); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *Handler) changeRoles (rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	Id := vars["id"]
	St := vars["utype"]

	if Id == "" {
		http.Error(rw, "Invalid URL", http.StatusInternalServerError)
		return
	}

	const getuser = `SELECT * FROM users WHERE id=$1`
	var user Auth
	h.db.Get(&user, getuser, Id)

	if user.ID == 0 {
		http.Error(rw, "Invalid URL", http.StatusInternalServerError)
		return
	}

	if St == "user" {
		const updateStatususers = `UPDATE users SET is_admin = false WHERE id=$1`
		res := h.db.MustExec(updateStatususers, Id)

		if ok, err := res.RowsAffected(); err != nil || ok == 0 {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		session, err := store.Get(r, "library")
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		session.AddFlash(user.Name+" is a Normal User Now!")
		err = session.Save(r, rw)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		const updateStatususers = `UPDATE users SET is_admin = true WHERE id=$1`
		res := h.db.MustExec(updateStatususers, Id)

		if ok, err := res.RowsAffected(); err != nil || ok == 0 {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		session, err := store.Get(r, "library")
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		session.AddFlash(user.Name+" is an Admin now!")
		err = session.Save(r, rw)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	http.Redirect(rw, r, "/users", http.StatusTemporaryRedirect)
}

func (h *Handler) adminPassword (rw http.ResponseWriter, r *http.Request) {
	var user Auth

	session, err := store.Get(r, "library")
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	if flashes := session.Flashes(); len(flashes) > 0 {
		if val, ok := flashes[0].(string); ok {
			user.Messege = val
			fmt.Println(val)
		}
	}
	err = session.Save(r, rw)

	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	if err != nil {
		http.Redirect(rw, r, "/login", http.StatusTemporaryRedirect)
		return
	}

	userID := session.Values["id"]
	const getuser = `SELECT * FROM users WHERE id=$1`
	h.db.Get(&user, getuser, userID)

	if user.ID == 0 {
		session.Values["authenticated"] = false
		session.Values["id"] = 0
		session.Save(r, rw)
		http.Redirect(rw, r, "/login", http.StatusTemporaryRedirect)
		return
	}

	user.Error = map[string]string{}
	
	if err := h.templates.ExecuteTemplate(rw, "change-password.html", user); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *Handler) adminPasswordPro (rw http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	session, err := store.Get(r, "library")
	if err != nil {
		http.Redirect(rw, r, "/login", http.StatusTemporaryRedirect)
		return
	}

	userID := session.Values["id"]
	const getuser = `SELECT * FROM users WHERE id=$1`
	var user Auth
	h.db.Get(&user, getuser, userID)

	if user.ID == 0 {
		session.Values["authenticated"] = false
		session.Values["id"] = 0
		session.Save(r, rw)
		http.Redirect(rw, r, "/login", http.StatusTemporaryRedirect)
		return
	}

	var users Auth
	if err := h.decoder.Decode(&users, r.PostForm); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	
	if err := users.PassValidate(); err != nil {
		valError, ok := err.(validation.Errors)
		if ok {
			vErrs := make(map[string]string)
			for key, value := range valError {
				vErrs[key] =value.Error()
			}
			user.Error = vErrs
			if err := h.templates.ExecuteTemplate(rw, "change-password.html", user); err != nil {
				http.Error(rw, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	HP := CheckPasswordHash(users.Old_password, user.Password)
	

	if !HP {
		user.Error = map[string]string{
			"Old_password": "Old Password Doesn't Match",
		}
		if err := h.templates.ExecuteTemplate(rw, "change-password.html", user); err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	}

	if users.Password != users.Confirm {
		user.Error = map[string]string{
			"Confirm": "The Password Confirmation Doesn't Match",
		}
		if err := h.templates.ExecuteTemplate(rw, "change-password.html", user); err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	}

	HashPass, err := HashPassword(users.Password)

	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	const updateStatusBooks = `UPDATE users SET password = $1 WHERE id=$2`
	res := h.db.MustExec(updateStatusBooks, HashPass, userID)

	if ok, err := res.RowsAffected(); err != nil || ok == 0 {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	session.AddFlash("Password Changed Successfully")
	err = session.Save(r, rw)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	
	http.Redirect(rw, r, "/change-admin-password", http.StatusTemporaryRedirect)
}