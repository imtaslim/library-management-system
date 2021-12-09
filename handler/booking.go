package handler

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	validation "github.com/go-ozzo/ozzo-validation"
)

type Bookings struct {
	ID         int    `db:"id"`
	Book_id    int    `db:"book_id"`
	User_id    int    `db:"user_id"`
	Start_time time.Time `db:"start_time"`
	End_time   time.Time `db:"end_time"`
	ST string
	ET   string
	Book_name	string
	User_name	string
	Status bool
}

func (b *Bookings) Validate() error {
	return validation.ValidateStruct(b,
		validation.Field(&b.ID,
			validation.Required.Error("No book selected"),
		),
		validation.Field(&b.ST,
			validation.Required.Error("The Start Time Field is Required"),
		),
		validation.Field(&b.ET,
			validation.Required.Error("The End Time Field is Required"),
		),
	)
}

type BookingsData struct {
	ID      string
	Booking Bookings
	Now string
	Errors  map[string]string
}

type ListBooking struct {
	Bookings []Bookings
	User_name string
	Pagination []Pagination
	TotalPage int
	CurrentPage int
	PrePageURL string
	NextPageURL string
}

func (h *Handler) bookingProcess(rw http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	var booking Bookings
	if err := h.decoder.Decode(&booking, r.PostForm); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	//fmt.Println(booking)
	if err := booking.Validate(); err != nil {
		valError, ok := err.(validation.Errors)
		if ok {
			vErrs := make(map[string]string)
			for key, value := range valError {
				vErrs[key] = value.Error()
			}
			
			http.Redirect(rw, r, "/", http.StatusTemporaryRedirect)
			return
		}
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	const getBook = `SELECT * FROM books WHERE id=$1`
	var books Book
	h.db.Get(&books, getBook, booking.ID)

	if books.ID == 0 {
		http.Error(rw, "Invalid URL", http.StatusInternalServerError)
		return
	}
	const updateStatusTodo = `UPDATE books SET status = false WHERE id=$1`
	ress := h.db.MustExec(updateStatusTodo, booking.ID)

	if ok, err := ress.RowsAffected(); err != nil || ok == 0 {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	session, _ := store.Get(r, "library")
	user := session.Values["id"]

	const insertBooking = `INSERT INTO bookings(book_id, user_id, start_time, end_time) VALUES ($1, $2, $3, $4)`
	res := h.db.MustExec(insertBooking, booking.ID, user, booking.ST, booking.ET)

	if ok, err := res.RowsAffected(); err != nil || ok == 0 {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	session, err := store.Get(r, "library")
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	session.AddFlash(books.Name+" is booked successfully!")
	err = session.Save(r, rw)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(rw, r, "/", http.StatusTemporaryRedirect)
}

func (h *Handler) bookings (rw http.ResponseWriter, r *http.Request) {
	bookings := []Bookings{}
	session, _ := store.Get(r, "library")
	user := session.Values["id"]
	var p int = 1
	var errr error
	var nextPageURL string
	var prePageURL string

	page := r.URL.Query().Get("page")
	if page != "" {
		p, errr = strconv.Atoi(page)
	}

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
	h.db.Get(&total, "SELECT count(*) FROM bookings WHERE user_id=$1", user)
    h.db.Select(&bookings, "SELECT * FROM bookings WHERE user_id=$1 ORDER BY id DESC OFFSET $2 LIMIT $3", user, offset, limit)

	totalPage := int(math.Ceil(float64(total)/float64(limit)))
	pagination := make([]Pagination,totalPage)
	for i:=0; i<totalPage; i++ {
		pagination[i] = Pagination{
			URL: fmt.Sprintf("http://localhost:3000/my-bookings?page=%d", i +1),
			PageNo: i +1,
		}
		if i + 1 == p {
			if i != 0 {
				prePageURL = fmt.Sprintf("http://localhost:3000/my-bookings?page=%d", i)
			}
			if i + 1 != totalPage {
				nextPageURL = fmt.Sprintf("http://localhost:3000/my-bookings?page=%d", i + 2)
			}
		}
	}

	for key, value := range bookings {
		const getTodo = `SELECT name, status FROM books WHERE id=$1`
		var book Book
		h.db.Get(&book, getTodo, value.Book_id)

		start_time:= value.Start_time.Format("Mon Jan _2 2006 15:04 AM")
		end_time:= value.End_time.Format("Mon Jan _2 2006 15:04 AM")
		
		bookings[key].Book_name = book.Name
		bookings[key].Status = book.Status
		bookings[key].ST = start_time
		bookings[key].ET = end_time
	}
	const getUsers = `SELECT name FROM users WHERE id=$1`
	var userName Auth
	h.db.Get(&userName, getUsers, user)
	
	lb := ListBooking {
		Bookings: bookings,
		User_name: userName.Name,
		Pagination: pagination,
		TotalPage: totalPage,
		CurrentPage: p,
		PrePageURL: prePageURL,
		NextPageURL: nextPageURL,
	}
	if err := h.templates.ExecuteTemplate(rw, "my-booking.html", lb); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
}
