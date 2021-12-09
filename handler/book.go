package handler

import (
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/gorilla/mux"
)

type Book struct{
	ID int `db:"id"`
	Cat_ID int `db:"cat_id"`
	Name string `db:"name"`
	Author_Name string `db:"author_name"`
	Details string `db:"details"`
	Status bool `db:"status"`
	Image string `db:"image"`
	Cat_name string
	ReleaseDate string
}

func (b *Book) Validate() error {
	return validation.ValidateStruct(b,
		validation.Field(&b.Name,
			validation.Required.Error("The Name Field is Required"),
			validation.Length(3, 0).Error("The Name field must be greater than or equals 3"),
		),
		validation.Field(&b.Author_Name,
			validation.Required.Error("The Author Name Field is Required"),
			validation.Length(3, 0).Error("The Author Name field must be greater than or equals 3"),
		),
		validation.Field(&b.Details,
			validation.Required.Error("The Details Field is Required"),
			validation.Length(3, 1000).Error("The Details field must be greater than or equals 3 and less than 1000"),
		),
		validation.Field(&b.Cat_ID,
			validation.Required.Error("Atleast one Category Should be Selected"),
		),
	)
}

type BooksData struct {
	ID int
	Book Book
	Category []Category
	Errors map[string]string
}
type Listbook struct{
	Books []Book
	Search string
	Is_login interface{}
	Messege string
	Categories []Category
	Pagination []Pagination
	TotalPage int
	CurrentPage int
	PrePageURL string
	NextPageURL string
}

func (h *Handler) booksHome (rw http.ResponseWriter, r *http.Request) {

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
    h.db.Select(&books, "SELECT * FROM books WHERE name ILIKE '%%' || $1 || '%%'  OR author_name ILIKE '%%' || $1 || '%%' ORDER BY status OFFSET $2 LIMIT $3", search, offset, limit)

	totalPage := int(math.Ceil(float64(total)/float64(limit)))
	pagination := make([]Pagination,totalPage)
	for i:=0; i<totalPage; i++ {
		pagination[i] = Pagination{
			URL: fmt.Sprintf("http://localhost:3000/books?page=%d%s", i +1, pageQuery),
			PageNo: i +1,
		}
		if i + 1 == p {
			if i != 0 {
				prePageURL = fmt.Sprintf("http://localhost:3000/books?page=%d%s", i, pageQuery)
			}
			if i + 1 != totalPage {
				nextPageURL = fmt.Sprintf("http://localhost:3000/books?page=%d%s", i + 2, pageQuery)
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
		Search: search,
		Messege: msg,
		Pagination: pagination,
		TotalPage: totalPage,
		CurrentPage: p,
		PrePageURL: prePageURL,
		NextPageURL: nextPageURL,
	}
	if err := h.templates.ExecuteTemplate(rw, "list-books.html", lc); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
}

// func (h *Handler) booksSearch (rw http.ResponseWriter, r *http.Request) {
// 	if err := r.ParseForm(); err != nil {
// 		http.Error(rw, err.Error(), http.StatusInternalServerError)
// 		return
// 	}
// 	ser := r.FormValue("search")
// 	if ser == "" {
// 		http.Redirect(rw, r, "/books", http.StatusTemporaryRedirect)
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
// 		Search: ser,
// 	}
// 	if err := h.templates.ExecuteTemplate(rw, "list-books.html", lc); err != nil {
// 		http.Error(rw, err.Error(), http.StatusInternalServerError)
// 		return
// 	}
// }

func (h *Handler) booksCreate (rw http.ResponseWriter, r *http.Request) {
	categories := []Category{}
    h.db.Select(&categories, "SELECT * FROM categories")
	
	vErrs := map[string]string{"name": ""}
	book := Book{}
	h.createBookData(rw, categories, book, vErrs)
}

const MAX_UPLOAD_SIZE = 1024 * 10024 // 1MB

func (h *Handler) booksStore (rw http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	r.Body = http.MaxBytesReader(rw, r.Body, MAX_UPLOAD_SIZE)
	if err := r.ParseMultipartForm(MAX_UPLOAD_SIZE); err != nil {
		http.Error(rw, "The uploaded file is too big. Please choose an file that's less than 1MB in size", http.StatusBadRequest)
		return
	}

	file, fileHeader, err := r.FormFile("Image")
	
    if err != nil {
        http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
    }
    defer file.Close()
   
	now := strconv.Itoa(int(time.Now().UnixNano()))
	img := "upload-*"+now+filepath.Ext(fileHeader.Filename)
    tempFile, err := ioutil.TempFile("assets/uploads", img)
	
    if err != nil {
        http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
    }
    defer tempFile.Close()

	fileBytes, err := ioutil.ReadAll(file)
    if err != nil {
        http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
    }
	
    tempFile.Write(fileBytes)
	imgName := tempFile.Name()

	categories := []Category{}
    h.db.Select(&categories, "SELECT * FROM categories")

	var book Book
	if err := h.decoder.Decode(&book, r.PostForm); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := book.Validate(); err != nil {
		valError, ok := err.(validation.Errors)
		if ok {
			vErrs := make(map[string]string)
			for key, value := range valError {
				vErrs[key] =value.Error()
			}
			h.createBookData(rw, categories, book, vErrs)
			return
		}
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	
	const insertCategories = `INSERT INTO books(cat_id, name, author_name, details, status, image) VALUES ($1, $2, $3, $4, $5, $6)`
	res := h.db.MustExec(insertCategories, book.Cat_ID, book.Name, book.Author_Name, book.Details, true, imgName)

	if ok, err := res.RowsAffected(); err != nil || ok == 0 {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	session, err := store.Get(r, "library")
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	session.AddFlash("Book Added Succesfully!")
	err = session.Save(r, rw)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(rw, r, "/books", http.StatusTemporaryRedirect)
}

func (h *Handler) createBookData (rw http.ResponseWriter, cat []Category, book Book, errs map[string]string) {
	form := BooksData{
		Category: cat,
		Book: book,
		Errors: errs,
	}
	if err := h.templates.ExecuteTemplate(rw, "create-books.html", form); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *Handler) bookComplete (rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	Id := vars["id"]
	St := vars["status"]

	if Id == "" {
		http.Error(rw, "Invalid URL", http.StatusInternalServerError)
		return
	}

	const getbook = `SELECT * FROM books WHERE id=$1`
	var book Book
	h.db.Get(&book, getbook, Id)

	if book.ID == 0 {
		http.Error(rw, "Invalid URL", http.StatusInternalServerError)
		return
	}

	if St == "1" {
		const updateStatusBooks = `UPDATE books SET status = true WHERE id=$1`
		res := h.db.MustExec(updateStatusBooks, Id)

		if ok, err := res.RowsAffected(); err != nil || ok == 0 {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		session, err := store.Get(r, "library")
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		session.AddFlash(book.Name+" is available!")
		err = session.Save(r, rw)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		const updateStatusBooks = `UPDATE books SET status = false WHERE id=$1`
		res := h.db.MustExec(updateStatusBooks, Id)

		if ok, err := res.RowsAffected(); err != nil || ok == 0 {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		session, err := store.Get(r, "library")
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		session.AddFlash(book.Name+" is unavailable!")
		err = session.Save(r, rw)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	http.Redirect(rw, r, "/books", http.StatusTemporaryRedirect)
}

func (h *Handler) bookEdit (rw http.ResponseWriter, r *http.Request) {
	categories := []Category{}
    h.db.Select(&categories, "SELECT * FROM categories")

	vars := mux.Vars(r)
	Id := vars["id"]

	if Id == "" {
		http.Error(rw, "Invalid URL", http.StatusInternalServerError)
		return
	}

	const getBook = `SELECT * FROM books WHERE id=$1`
	var book Book
	h.db.Get(&book, getBook, Id)

	if book.ID == 0 {
		http.Error(rw, "Invalid URL", http.StatusInternalServerError)
		return
	}
	
	rerrs := map[string]string{"title": "This field is required"}
		h.editBookData(rw, book.ID, book, categories,  rerrs)
}

func (h *Handler) bookUpdate (rw http.ResponseWriter, r *http.Request) {
	categories := []Category{}
    h.db.Select(&categories, "SELECT * FROM categories")

	vars := mux.Vars(r)
	Id := vars["id"]

	if err := r.ParseForm(); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	imgName := ""
	file, fileHeader, err := r.FormFile("Image")

	if file != nil {
		r.Body = http.MaxBytesReader(rw, r.Body, MAX_UPLOAD_SIZE)
		if err := r.ParseMultipartForm(MAX_UPLOAD_SIZE); err != nil {
			http.Error(rw, "The uploaded file is too big. Please choose an file that's less than 10MB in size", http.StatusBadRequest)
			return
		}
		
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
		defer file.Close()
	
		now := strconv.Itoa(int(time.Now().UnixNano()))
		img := "upload-*"+now+filepath.Ext(fileHeader.Filename)
		tempFile, err := ioutil.TempFile("assets/uploads", img)
		
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
		defer tempFile.Close()

		fileBytes, err := ioutil.ReadAll(file)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
		
		tempFile.Write(fileBytes)
	}

	var book Book
	if err := h.decoder.Decode(&book, r.PostForm); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := book.Validate(); err != nil {
		valError, ok := err.(validation.Errors)
		if ok {
			vErrs := make(map[string]string)
			for key, value := range valError {
				vErrs[key] =value.Error()
			}
			h.editBookData(rw, book.ID, book, categories, vErrs)
			return
		}
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	const getTodo = `SELECT * FROM books WHERE id=$1`
	var books Book
	h.db.Get(&books, getTodo, Id)

	if books.ID == 0 {
		http.Error(rw, "Invalid URL", http.StatusInternalServerError)
		return
	}

	if file == nil {
		imgName = books.Image
	}else{
		if err := os.Remove(books.Image); err != nil {
			http.Error(rw, "Invalid URL", http.StatusInternalServerError)
			return
		}
	}

	const updateStatusCategories = `UPDATE books SET name=$1, author_name=$2, details=$3, cat_id=$4, image=$5 WHERE id=$6`
	res := h.db.MustExec(updateStatusCategories, book.Name, book.Author_Name, book.Details, book.Cat_ID, imgName, Id)

	if ok, err := res.RowsAffected(); err != nil || ok == 0 {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	session, err := store.Get(r, "library")
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	session.AddFlash(book.Name+" is updated successfully!")
	err = session.Save(r, rw)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(rw, r, "/books", http.StatusTemporaryRedirect)
}

func (h *Handler) editBookData (rw http.ResponseWriter, id int, book Book, cat []Category, errs map[string]string) {
	form := BooksData{
		ID: id,
		Book: book,
		Category: cat,
		Errors: errs,
	}
	if err := h.templates.ExecuteTemplate(rw, "edit-books.html", form); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *Handler) bookDelete (rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	Id := vars["id"]

	const getBook = `SELECT * FROM books WHERE id=$1`
	var book Book
	h.db.Get(&book, getBook, Id)

	if book.ID == 0 {
		http.Error(rw, "Invalid URL", http.StatusInternalServerError)
		return
	}

	if err := os.Remove(book.Image); err != nil {
        http.Error(rw, "Invalid URL", http.StatusInternalServerError)
		return
    }

	const deleteBooks = `DELETE FROM books WHERE id=$1`
	res := h.db.MustExec(deleteBooks, Id)

	if ok, err := res.RowsAffected(); err != nil || ok == 0 {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	session, err := store.Get(r, "library")
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	session.AddFlash(book.Name+" is deleted successfully!")
	err = session.Save(r, rw)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(rw, r, "/books", http.StatusTemporaryRedirect)
}

func (h *Handler) bookDetails (rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	Id := vars["id"]

	const getBook = `SELECT * FROM books WHERE id=$1`
	var book Book
	h.db.Get(&book, getBook, Id)

	const getTodo = `SELECT name FROM categories WHERE id=$1`
	var category Category
	h.db.Get(&category, getTodo, book.Cat_ID)
	book.Cat_name = category.Name

	book.ReleaseDate = randate().Format("_2 January 2006")
	
	if err := h.templates.ExecuteTemplate(rw, "book-details.html", book); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
}

func randate() time.Time {
    min := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
    max := time.Now().Unix()
    delta := max - min

    sec := rand.Int63n(delta) - min
    return time.Unix(sec, 0)
}