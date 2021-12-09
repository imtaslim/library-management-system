package handler

import (
	"fmt"
	"net/http"

	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/go-ozzo/ozzo-validation/is"
)

func (a *Auth) Validate() error {
	return validation.ValidateStruct(a,
		validation.Field(&a.Name,
			validation.Required.Error("The Name Field is Required"),
			validation.Length(3, 0).Error("The Name field must be greater than or equals 3"),
		),
		validation.Field(&a.Email,
			validation.Required.Error("The Email Field is Required"),
			is.Email.Error("The Field Must be an Email"),
		),
	)
}

func (a *Auth) PassValidate() error {
	return validation.ValidateStruct(a,
		validation.Field(&a.Old_password,
			validation.Required.Error("The Old Password Field is Required"),
			validation.Length(8, 0).Error("The Old Password must have 8 Characters"),
			is.Alphanumeric.Error("This Field Must contains Uppercase, Lowercase and Number"),
		),
		validation.Field(&a.Password,
			validation.Required.Error("The New Password Field is Required"),
			validation.Length(8, 0).Error("The New Password must have 8 Characters"),
			is.Alphanumeric.Error("This Field Must contains Uppercase, Lowercase and Number"),
		),
		validation.Field(&a.Confirm,
			validation.Required.Error("The Confirm Password Field is Required"),
			validation.Length(8, 0).Error("The Password must have 8 Characters"),
			is.Alphanumeric.Error("This Field Must contains Uppercase, Lowercase and Number"),
		),
	)
}

func (h *Handler) profile (rw http.ResponseWriter, r *http.Request) {
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
	
	if err := h.templates.ExecuteTemplate(rw, "my-profile.html", user); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *Handler) updateProfile (rw http.ResponseWriter, r *http.Request) {
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

	if err := users.Validate(); err != nil {
		valError, ok := err.(validation.Errors)
		if ok {
			vErrs := make(map[string]string)
			for key, value := range valError {
				vErrs[key] =value.Error()
			}
			users.Error = vErrs
			if err := h.templates.ExecuteTemplate(rw, "my-profile.html", users); err != nil {
				http.Error(rw, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	const updateStatusBooks = `UPDATE users SET name = $1, email = $2 WHERE id=$3`
	res := h.db.MustExec(updateStatusBooks, users.Name, users.Email, userID)	
	
	if ok, err := res.RowsAffected(); err != nil || ok == 0 {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	session.AddFlash("Profile Update Successfully")
	err = session.Save(r, rw)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	
	http.Redirect(rw, r, "/profile", http.StatusTemporaryRedirect)
}

func (h *Handler) changePassword (rw http.ResponseWriter, r *http.Request) {
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
			if err := h.templates.ExecuteTemplate(rw, "my-profile.html", user); err != nil {
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
		if err := h.templates.ExecuteTemplate(rw, "my-profile.html", user); err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	}

	if users.Password != users.Confirm {
		user.Error = map[string]string{
			"Confirm": "The Password Confirmation Doesn't Match",
		}
		if err := h.templates.ExecuteTemplate(rw, "my-profile.html", user); err != nil {
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
	
	http.Redirect(rw, r, "/profile", http.StatusTemporaryRedirect)
}