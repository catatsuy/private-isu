package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"text/template"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/sessions"
	"github.com/jmoiron/sqlx"
	"github.com/zenazn/goji"
)

var (
	db    *sqlx.DB
	store *sessions.CookieStore
)

type user struct {
	ID          int       `db:"id"`
	AccountName string    `db:"account_name"`
	Passhash    string    `db:"passhash"`
	Authority   int       `db:"authority"`
	DelFlg      int       `db:"del_flg"`
	CreatedAt   time.Time `db:"created_at"`
}

type post struct {
	ID        int       `db:"id"`
	UserID    int       `db:"user_id"`
	Imgdata   []byte    `db:"imagdata"`
	Body      string    `db:"body"`
	Mime      string    `db:"mime"`
	CreatedAt time.Time `db:"created_at"`
}

type comment struct {
	ID        int       `db:"id"`
	PostID    int       `db:"post_id"`
	UserID    int       `db:"user_id"`
	Comment   string    `db:"comment"`
	CreatedAt time.Time `db:"created_at"`
}

func init() {
	store = sessions.NewCookieStore([]byte("Iscogram"))
}

func getSession(r *http.Request) *sessions.Session {
	session, err := store.Get(r, "isuconp-go.session")
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return session
}

func getSessionUser(r *http.Request) *user {
	session := getSession(r)
	uid, ok := session.Values["user_id"]
	if !ok || uid == nil {
		return nil
	}

	u := &user{}

	err := db.Get(&u, "SELECT * FROM `users` WHERE `id` = ?", uid)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	return u
}

func tryLogin(accountName, password string) *user {
	u := user{}
	err := db.Get(&u, "SELECT * FROM users WHERE account_name = ? AND del_flg = 0", accountName)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	if &u != nil && calculatePasshash(u.AccountName, password) == u.Passhash {
		return &u
	} else if &u == nil {
		return nil
	} else {
		return nil
	}

	return &u
}

func digest(src string) string {
	fmt.Println(src)
	// TODO: escape
	out, err := exec.Command("/bin/bash", "-c", `printf "%s" "`+src+`" | openssl dgst -sha512 | sed 's/^.*= //'`).Output()
	if err != nil {
		fmt.Println(err)
		return ""
	}

	fmt.Println(strings.TrimSuffix(string(out), "\n"))
	return strings.TrimSuffix(string(out), "\n")
}

func calculateSalt(accountName string) string {
	return digest(accountName)
}

func calculatePasshash(accountName, password string) string {
	return digest(password + ":" + calculateSalt(accountName))
}

func getTemplPath(filename string) string {
	return path.Join("templates", filename)
}

func getIndex(w http.ResponseWriter, r *http.Request) {
	// fmt.Fprintf(w, "Hello, %s!", c.URLParams["name"])
	_ = getSessionUser(r)

	posts := []post{}

	err := db.Select(&posts, "SELECT `id`, `user_id`, `body`, `mime`, `created_at` FROM `posts` ORDER BY `created_at` DESC")
	if err != nil {
		fmt.Println(err)
		return
	}

	template.Must(template.ParseFiles(
		getTemplPath("layout.html"),
		getTemplPath("index.html"),
		getTemplPath("posts.html"),
		getTemplPath("post.html")),
	).Execute(w, struct{ Posts []post }{posts})
}

func postLogin(w http.ResponseWriter, r *http.Request) {
	if getSessionUser(r) != nil {
		w.Header().Set("Location", "/")
		w.WriteHeader(http.StatusFound)
		return
	}

	u := tryLogin(r.FormValue("account_name"), r.FormValue("password"))

	if u != nil {
		session := getSession(r)
		session.Values["user_id"] = u.ID
		session.Save(r, w)

		w.Header().Set("Location", "/")
		w.WriteHeader(http.StatusFound)
	} else {
		w.Header().Set("Location", "/login")
		w.WriteHeader(http.StatusFound)
	}
}

func main() {
	host := os.Getenv("ISUCONP_DB_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("ISUCONP_DB_PORT")
	if port == "" {
		port = "3306"
	}
	_, err := strconv.Atoi(port)
	if err != nil {
		log.Fatalf("Failed to read DB port number from an environment variable ISUCONP_DB_PORT.\nError: %s", err.Error())
	}
	user := os.Getenv("ISUCONP_DB_USER")
	if user == "" {
		user = "root"
	}
	password := os.Getenv("ISUCONP_DB_PASSWORD")
	dbname := os.Getenv("ISUCONP_DB_NAME")
	if dbname == "" {
		dbname = "isucon5q"
	}

	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=Local",
		user,
		password,
		host,
		port,
		dbname,
	)

	db, err = sqlx.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %s.", err.Error())
	}
	defer db.Close()

	goji.Get("/", getIndex)
	goji.Post("/login", postLogin)
	goji.Serve()
}
