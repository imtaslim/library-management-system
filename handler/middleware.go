package handler

import "net/http"

func (h *Handler) authMiddleware (next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, "library")
		if err != nil {
			http.Redirect(rw, r, "/login", http.StatusTemporaryRedirect)
			return
		}

		// Check if user is authenticated
		ok := session.Values["authenticated"]
		if ok == nil || ok == false {
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
		next.ServeHTTP(rw, r)
	})
}

func (h *Handler) loginMiddleware (next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, "library")
		if err != nil {
			http.Redirect(rw, r, "/login", http.StatusTemporaryRedirect)
			return
		}

		userID := session.Values["id"]

		if userID != 0{
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

			if user.IsAdmin {
				http.Redirect(rw, r, "/admin", http.StatusTemporaryRedirect)
				return
			}else if !user.IsAdmin {
				http.Redirect(rw, r, "/", http.StatusTemporaryRedirect)
				return
			}
		}
		
		next.ServeHTTP(rw, r)
	})
}

func (h *Handler) adminMiddleware (next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, "library")
		if err != nil {
			http.Redirect(rw, r, "/login", http.StatusTemporaryRedirect)
			return
		}

		// Check if user is authenticated
		ok := session.Values["authenticated"]
		
		if ok == nil || ok == false {
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

		if !user.IsAdmin {
			http.Error(rw, "Unauthorised Access", http.StatusInternalServerError)
			return
		}
		next.ServeHTTP(rw, r)
	})
}