package handler

import (
	"fmt"
	"math"
	"net/http"
	"strconv"

	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/gorilla/mux"
)

type Category struct{
	ID int `db:"id"`
	Name string `db:"name"`
	Status bool `db:"status"`
}

func (c *Category) Validate() error {
	return validation.ValidateStruct(c,
		validation.Field(&c.Name,
			validation.Required.Error("The Name Field is Required"),
			validation.Length(3, 0).Error("The Name field must be greater than or equals 3"),
		),
	)
}

type CategoryData struct {
	ID int
	Name string
	Errors map[string]string
}

type ListCategory struct{
	Categories []Category
	Search string
	Messege string
	Pagination []Pagination
	TotalPage int
	CurrentPage int
	PrePageURL string
	NextPageURL string
}

type Pagination struct{
	URL string
	PageNo int
}

func (h *Handler) categoriesHome (rw http.ResponseWriter, r *http.Request) {
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

	categories := []Category{}

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
	h.db.Get(&total, "SELECT count(*) FROM categories WHERE name ILIKE '%%' || $1 || '%%'", search)
    h.db.Select(&categories, "SELECT * FROM categories WHERE name ILIKE '%%' || $1 || '%%' OFFSET $2 LIMIT $3", search, offset, limit)

	totalPage := int(math.Ceil(float64(total)/float64(limit)))
	pagination := make([]Pagination,totalPage)
	for i:=0; i<totalPage; i++ {
		pagination[i] = Pagination{
			URL: fmt.Sprintf("http://localhost:3000/categories?page=%d%s", i +1, pageQuery),
			PageNo: i +1,
		}
		if i + 1 == p {
			if i != 0 {
				prePageURL = fmt.Sprintf("http://localhost:3000/categories?page=%d%s", i, pageQuery)
			}
			if i + 1 != totalPage {
				nextPageURL = fmt.Sprintf("http://localhost:3000/categories?page=%d%s", i + 2, pageQuery)
			}
		}
	}

	lt := ListCategory{
		Categories: categories,
		Search: search,
		Messege: msg,
		Pagination: pagination,
		TotalPage: totalPage,
		CurrentPage: p,
		PrePageURL: prePageURL,
		NextPageURL: nextPageURL,
	}
	
	if err := h.templates.ExecuteTemplate(rw, "list-categories.html", lt); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
}

// func (h *Handler) categoriesSearch (rw http.ResponseWriter, r *http.Request) {
// 	if err := r.ParseForm(); err != nil {
// 		http.Error(rw, err.Error(), http.StatusInternalServerError)
// 		return
// 	}
// 	ser := r.FormValue("search")
// 	if ser == "" {
// 		http.Redirect(rw, r, "/categories", http.StatusTemporaryRedirect)
// 		return
// 	}

// 	const getSrc = `SELECT * FROM categories WHERE name ILIKE '%%' || $1 || '%%'`
// 	var categories []Category
// 	h.db.Select(&categories, getSrc,ser)

// 	lt := ListCategory {
// 		Categories: categories,
// 		Search: ser,
// 	}
// 	if err := h.templates.ExecuteTemplate(rw, "list-categories.html", lt); err != nil {
// 		http.Error(rw, err.Error(), http.StatusInternalServerError)
// 		return
// 	}
// }

func (h *Handler) categoriesCreate (rw http.ResponseWriter, r *http.Request) {
	vErrs := map[string]string{"name": ""}
	Name := ""
	h.createCategoryData(rw, Name, vErrs)
}

func (h *Handler) categoriesStore (rw http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	var category Category
	if err := h.decoder.Decode(&category, r.PostForm); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := category.Validate(); err != nil {
		valError, ok := err.(validation.Errors)
		if ok {
			vErrs := make(map[string]string)
			for key, value := range valError {
				vErrs[key] =value.Error()
			}
			h.createCategoryData(rw, category.Name, vErrs)
			return
		}
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	
	const insertCategories = `INSERT INTO categories(name, status) VALUES ($1, $2)`
	res := h.db.MustExec(insertCategories, category.Name, true)

	if ok, err := res.RowsAffected(); err != nil || ok == 0 {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	session, err := store.Get(r, "library")
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	session.AddFlash("New Category Added Successfully!")
	err = session.Save(r, rw)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(rw, r, "/categories", http.StatusTemporaryRedirect)
}

func (h *Handler) categoriesComplete (rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	Id := vars["id"]
	St := vars["status"]

	if Id == "" {
		http.Error(rw, "Invalid URL", http.StatusInternalServerError)
		return
	}

	const getTodo = `SELECT * FROM categories WHERE id=$1`
	var category Category
	h.db.Get(&category, getTodo, Id)

	if category.ID == 0 {
		http.Error(rw, "Invalid URL", http.StatusInternalServerError)
		return
	}

	if St == "1" {
		const updateStatusCategories = `UPDATE categories SET status = true WHERE id=$1`
		res := h.db.MustExec(updateStatusCategories, Id)

		if ok, err := res.RowsAffected(); err != nil || ok == 0 {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		const updateStatusCategories = `UPDATE categories SET status = false WHERE id=$1`
		res := h.db.MustExec(updateStatusCategories, Id)

		if ok, err := res.RowsAffected(); err != nil || ok == 0 {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	http.Redirect(rw, r, "/categories", http.StatusTemporaryRedirect)
}

func (h *Handler) categoriesEdit (rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	Id := vars["id"]

	if Id == "" {
		http.Error(rw, "Invalid URL", http.StatusInternalServerError)
		return
	}

	const getTodo = `SELECT * FROM categories WHERE id=$1`
	var category Category
	h.db.Get(&category, getTodo, Id)

	if category.ID == 0 {
		http.Error(rw, "Invalid URL", http.StatusInternalServerError)
		return
	}
	
	rerrs := map[string]string{"title": "This field is required"}
		h.editCategoryData(rw, category.ID, category.Name, rerrs)
}

func (h *Handler) categoriesUpdate (rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	Id := vars["id"]

	if Id == "" {
		http.Error(rw, "Invalid URL", http.StatusInternalServerError)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	var category Category
	if err := h.decoder.Decode(&category, r.PostForm); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	id, err := strconv.Atoi(Id)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := category.Validate(); err != nil {
		valError, ok := err.(validation.Errors)
		if ok {
			vErrs := make(map[string]string)
			for key, value := range valError {
				vErrs[key] =value.Error()
			}
			h.editCategoryData(rw, id, category.Name, vErrs)
			return
		}
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	const getTodo = `SELECT * FROM categories WHERE id=$1`
	var categories Category
	h.db.Get(&categories, getTodo, Id)

	if categories.ID == 0 {
		http.Error(rw, "Invalid URL", http.StatusInternalServerError)
		return
	}

	const updateStatusCategories = `UPDATE categories SET name=$1 WHERE id=$2`
	res := h.db.MustExec(updateStatusCategories, category.Name, Id)

	if ok, err := res.RowsAffected(); err != nil || ok == 0 {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	session, err := store.Get(r, "library")
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	session.AddFlash("Category Updated Successfully!")
	err = session.Save(r, rw)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(rw, r, "/categories", http.StatusTemporaryRedirect)
}

func (h *Handler) categoriesDelete (rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	Id := vars["id"]

	if Id == "" {
		http.Error(rw, "Invalid URL", http.StatusInternalServerError)
		return
	}

	const getTodo = `SELECT * FROM categories WHERE id=$1`
	var category Category
	h.db.Get(&category, getTodo, Id)

	if category.ID == 0 {
		http.Error(rw, "Invalid URL", http.StatusInternalServerError)
		return
	}

	const deleteCategories = `DELETE FROM categories WHERE id=$1`
	res := h.db.MustExec(deleteCategories, Id)

	if ok, err := res.RowsAffected(); err != nil || ok == 0 {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	session, err := store.Get(r, "library")
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	session.AddFlash("Category Deleted Successfully!")
	err = session.Save(r, rw)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(rw, r, "/categories", http.StatusTemporaryRedirect)
}

func (h *Handler) createCategoryData (rw http.ResponseWriter, name string, errs map[string]string) {
	form := CategoryData{
		Name: name,
		Errors: errs,
	}
	if err := h.templates.ExecuteTemplate(rw, "create-categories.html", form); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *Handler) editCategoryData (rw http.ResponseWriter, id int, name string, errs map[string]string) {
	form := CategoryData{
		ID: id,
		Name: name,
		Errors: errs,
	}
	if err := h.templates.ExecuteTemplate(rw, "edit-categories.html", form); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
}