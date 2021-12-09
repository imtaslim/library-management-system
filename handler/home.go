package handler

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	//"time"
)

type Count struct {
	User int
	Booking int
	Book int
}

func (h *Handler) home (rw http.ResponseWriter, r *http.Request) {
	// Now:= time.Now()
	// const getbooking = `SELECT * FROM bookings WHERE end_time < $1`
	// var booking []Bookings
	// h.db.Select(&booking, getbooking, Now)

	// for _, value := range booking {
	// 	const updateStatusBooks = `UPDATE books SET status=true WHERE id=$1`
	// 	res := h.db.MustExec(updateStatusBooks, value.Book_id)

	// 	if ok, err := res.RowsAffected(); err != nil || ok == 0 {
	// 		http.Error(rw, err.Error(), http.StatusInternalServerError)
	// 		return
	// 	}	
	// }

	session, err := store.Get(r, "library")

	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	auth := session.Values["authenticated"]
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

	var categories []Category
	h.db.Select(&categories, "SELECT * FROM categories ORDER BY id DESC")

	books := []Book{}
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
	limit := 5

	if p > 0 {
		offset = limit * p - limit
	}
	
	total := 0
	h.db.Get(&total, "SELECT count(*) FROM books WHERE name ILIKE '%%' || $1 || '%%'  OR author_name ILIKE '%%' || $1 || '%%'", search)
    h.db.Select(&books, "SELECT * FROM books WHERE name ILIKE '%%' || $1 || '%%'  OR author_name ILIKE '%%' || $1 || '%%' ORDER BY status DESC OFFSET $2 LIMIT $3", search, offset, limit)

	totalPage := int(math.Ceil(float64(total)/float64(limit)))
	pagination := make([]Pagination,totalPage)
	for i:=0; i<totalPage; i++ {
		pagination[i] = Pagination{
			URL: fmt.Sprintf("http://localhost:3000?page=%d%s", i +1, pageQuery),
			PageNo: i +1,
		}
		if i + 1 == p {
			if i != 0 {
				prePageURL = fmt.Sprintf("http://localhost:3000?page=%d%s", i, pageQuery)
			}
			if i + 1 != totalPage {
				nextPageURL = fmt.Sprintf("http://localhost:3000?page=%d%s", i + 2, pageQuery)
			}
		}
	}
	
	for key, value := range books {
		const getTodo = `SELECT name FROM categories WHERE id=$1`
		var category Category
		h.db.Get(&category, getTodo, value.Cat_ID)
		books[key].Cat_name = category.Name
	}
	lc := Listbook {
		Books: books,
		Is_login: auth,
		Search: search,
		Messege: msg,
		Categories: categories,
		Pagination: pagination,
		TotalPage: totalPage,
		CurrentPage: p,
		PrePageURL: prePageURL,
		NextPageURL: nextPageURL,
	}
	if err := h.templates.ExecuteTemplate(rw, "home.html", lc); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
}

// func (h *Handler) search (rw http.ResponseWriter, r *http.Request) {
// 	session, _ := store.Get(r, "library")
// 	auth := session.Values["authenticated"]

// 	var categories []Category
// 	h.db.Select(&categories, "SELECT * FROM categories ORDER BY id DESC")

// 	if err := r.ParseForm(); err != nil {
// 		http.Error(rw, err.Error(), http.StatusInternalServerError)
// 		return
// 	}
// 	ser := r.FormValue("search")
// 	if ser == "" {
// 		http.Redirect(rw, r, "/", http.StatusTemporaryRedirect)
// 		return
// 	}

// 	const getSrc = `SELECT * FROM books WHERE name ILIKE '%%' || $1 || '%%' OR author_name ILIKE '%%' || $1 || '%%'`
// 	var books []Book
// 	h.db.Select(&books, getSrc,ser)

// 	for key, value := range books {
// 		const getTodo = `SELECT name FROM categories WHERE id=$1`
// 		var category Category
// 		h.db.Get(&category, getTodo, value.Cat_ID)
// 		books[key].Cat_name = category.Name
// 	}

// 	lc := Listbook {
// 		Books: books,
// 		Is_login: auth,
// 		Search: ser,
// 		Categories: categories,
// 	}
// 	if err := h.templates.ExecuteTemplate(rw, "home.html", lc); err != nil {
// 		http.Error(rw, err.Error(), http.StatusInternalServerError)
// 		return
// 	}
// }

func (h *Handler) adminHome (rw http.ResponseWriter, r *http.Request) {
	
	var user int
	var book int
	var booking int

	h.db.Get(&user, "SELECT count(*) FROM users WHERE is_admin = false")
	h.db.Get(&book, "SELECT count(*) FROM books")
	h.db.Get(&booking, "SELECT count(*) FROM bookings")

	lt := Count{
		User: user,
		Book: book,
		Booking: booking,
	}

	if err := h.templates.ExecuteTemplate(rw, "admin.html", lt); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
}