package main

import (
	"log"
	"net/http"

	"library/handler"

	_ "github.com/lib/pq"
	"github.com/gorilla/schema"
    "github.com/jmoiron/sqlx"
)



func main() {
	var tableMigration = `
		CREATE TABLE IF NOT EXISTS users (
			id serial,
			name text,
			email text,
			password text,
			is_admin boolean,
			status boolean,
			verify_key text,

			primary key(id)
		);

		CREATE TABLE IF NOT EXISTS categories (
			id serial,
			name text,
			status boolean,

			primary key(id)
		);

		CREATE TABLE IF NOT EXISTS books (
			id serial,
			cat_id integer,
			name text,
			author_name text,
			details text,
			status boolean,
			image text,

			primary key(id)
		);

		CREATE TABLE IF NOT EXISTS bookings (
			id serial,
			book_id integer,
			user_id integer,
			start_time timestamp,
			end_time timestamp,

			primary key(id)
		);
	`
	db, err := sqlx.Connect("postgres", "user=postgres password=P@ssw0rd dbname=library sslmode=disable")
    if err != nil {
        log.Fatalln(err)
    }

	db.MustExec(tableMigration)
	decoder := schema.NewDecoder()
	decoder.IgnoreUnknownKeys(true)
	r := handler.New(db, decoder)
	
	log.Println("Server Starting....")
	if err := http.ListenAndServe("127.0.0.1:3000", r); err != nil {
		log.Fatal(err)
	}
}
