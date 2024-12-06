package main

import (
	crand "crypto/rand"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	gsm "github.com/bradleypeabody/gorilla-sessions-memcache"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/template/html/v2"
	"github.com/gorilla/sessions"
	"github.com/jmoiron/sqlx"
)

var (
	db    *sqlx.DB
	store *gsm.MemcacheStore
)

const (
	postsPerPage  = 20
	ISO8601Format = "2006-01-02T15:04:05-07:00"
	UploadLimit   = 10 * 1024 * 1024 // 10mb
)

type User struct {
	ID          int       `db:"id"`
	AccountName string    `db:"account_name"`
	Passhash    string    `db:"passhash"`
	Authority   int       `db:"authority"`
	DelFlg      int       `db:"del_flg"`
	CreatedAt   time.Time `db:"created_at"`
}

type Post struct {
	ID           int       `db:"id"`
	UserID       int       `db:"user_id"`
	Imgdata      []byte    `db:"imgdata"`
	Body         string    `db:"body"`
	Mime         string    `db:"mime"`
	CreatedAt    time.Time `db:"created_at"`
	CommentCount int
	Comments     []Comment
	User         User
	CSRFToken    string
}

type Comment struct {
	ID        int       `db:"id"`
	PostID    int       `db:"post_id"`
	UserID    int       `db:"user_id"`
	Comment   string    `db:"comment"`
	CreatedAt time.Time `db:"created_at"`
	User      User
}

func init() {
	memdAddr := os.Getenv("ISUCONP_MEMCACHED_ADDRESS")
	if memdAddr == "" {
		memdAddr = "localhost:11211"
	}
	memcacheClient := memcache.New(memdAddr)
	store = gsm.NewMemcacheStore(memcacheClient, "iscogram_", []byte("sendagaya"))
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

func dbInitialize() {
	sqls := []string{
		"DELETE FROM users WHERE id > 1000",
		"DELETE FROM posts WHERE id > 10000",
		"DELETE FROM comments WHERE id > 100000",
		"UPDATE users SET del_flg = 0",
		"UPDATE users SET del_flg = 1 WHERE id % 50 = 0",
	}

	for _, sql := range sqls {
		db.Exec(sql)
	}
}

func tryLogin(accountName, password string) *User {
	u := User{}
	err := db.Get(&u, "SELECT * FROM users WHERE account_name = ? AND del_flg = 0", accountName)
	if err != nil {
		return nil
	}

	if calculatePasshash(u.AccountName, password) == u.Passhash {
		return &u
	} else {
		return nil
	}
}

func validateUser(accountName, password string) bool {
	return regexp.MustCompile(`\A[0-9a-zA-Z_]{3,}\z`).MatchString(accountName) &&
		regexp.MustCompile(`\A[0-9a-zA-Z_]{6,}\z`).MatchString(password)
}

// 今回のGo実装では言語側のエスケープの仕組みが使えないのでOSコマンドインジェクション対策できない
// 取り急ぎPHPのescapeshellarg関数を参考に自前で実装
// cf: http://jp2.php.net/manual/ja/function.escapeshellarg.php
func escapeshellarg(arg string) string {
	return "'" + strings.Replace(arg, "'", "'\\''", -1) + "'"
}

func digest(src string) string {
	// opensslのバージョンによっては (stdin)= というのがつくので取る
	out, err := exec.Command("/bin/bash", "-c", `printf "%s" `+escapeshellarg(src)+` | openssl dgst -sha512 | sed 's/^.*= //'`).Output()
	if err != nil {
		log.Print(err)
		return ""
	}

	return strings.TrimSuffix(string(out), "\n")
}

func calculateSalt(accountName string) string {
	return digest(accountName)
}

func calculatePasshash(accountName, password string) string {
	return digest(password + ":" + calculateSalt(accountName))
}

func getSession(c *fiber.Ctx) *sessions.Session {
    // FastHTTPをnet/httpに変換
    var req http.Request
    
    // URLの設定
    req.URL = &url.URL{
        Path: string(c.Path()),
        RawQuery: string(c.Context().QueryArgs().QueryString()),
    }
    
    // ヘッダーの設定
    req.Header = make(http.Header)
    c.Context().Request.Header.VisitAll(func(key, value []byte) {
        req.Header.Add(string(key), string(value))
    })
    
    // Cookieの設定
    var cookies []http.Cookie
    c.Context().Request.Header.VisitAllCookie(func(key, value []byte) {
        cookies = append(cookies, http.Cookie{Name: string(key), Value: string(value)})
    })
    
    // Cookieヘッダーを正しく設定
    var cookieStrings []string
    for _, cookie := range cookies {
        cookieStrings = append(cookieStrings, cookie.String())
    }
    req.Header.Set("Cookie", strings.Join(cookieStrings, "; "))
    
    session, _ := store.Get(&req, "isuconp-go.session")

	return session
}

func getSessionUser(c *fiber.Ctx) User {
	session := getSession(c)
	uid, ok := session.Values["user_id"]
	if !ok || uid == nil {
		return User{}
	}

	u := User{}

	err := db.Get(&u, "SELECT * FROM `users` WHERE `id` = ?", uid)
	if err != nil {
		return User{}
	}

	return u
}

func converFiberRequestToHttpRequest(c *fiber.Ctx) *http.Request {
	return &http.Request{
		Method: string(c.Method()),
		URL: &url.URL{
			Path: string(c.Path()),
			RawQuery: string(c.Context().QueryArgs().String()),
		},
		Header: make(http.Header),
	}
}

type responseWriter struct {
    c *fiber.Ctx
}

func (rw *responseWriter) Header() http.Header {
    h := make(http.Header)
    rw.c.Context().Response.Header.VisitAll(func(key, value []byte) {
        h.Add(string(key), string(value))
    })
    return h
}

func (rw *responseWriter) Write(b []byte) (int, error) {
    return rw.c.Write(b)
}

func (rw *responseWriter) WriteHeader(statusCode int) {
    rw.c.Status(statusCode)
}

// セッション保存のヘルパー関数
func saveSession(c *fiber.Ctx, session *sessions.Session) error {
    req := &http.Request{
        Header: make(http.Header),
    }
    c.Context().Request.Header.VisitAll(func(key, value []byte) {
        req.Header.Add(string(key), string(value))
    })
    
    rw := &responseWriter{c: c}
    err := session.Save(req, rw)
    if err != nil {
        return err
    }
    
    // セッションクッキーをFiberのレスポンスに設定
    for _, cookie := range req.Response.Cookies() {
        c.Cookie(&fiber.Cookie{
            Name:     cookie.Name,
            Value:    cookie.Value,
            Path:     cookie.Path,
            Domain:   cookie.Domain,
            MaxAge:   cookie.MaxAge,
            Expires:  cookie.Expires,
            Secure:   cookie.Secure,
            HTTPOnly: cookie.HttpOnly,
        })
    }
    
    return nil
}

func getFlash(c *fiber.Ctx, key string) string {
	session := getSession(c)
	value, ok := session.Values[key]

	if !ok || value == nil {
		return ""
	} else {
		delete(session.Values, key)
		saveSession(c, session)
		return value.(string)
	}
}

func makePosts(results []Post, csrfToken string, allComments bool) ([]Post, error) {
	var posts []Post

	for _, p := range results {
		err := db.Get(&p.CommentCount, "SELECT COUNT(*) AS `count` FROM `comments` WHERE `post_id` = ?", p.ID)
		if err != nil {
			return nil, err
		}

		query := "SELECT * FROM `comments` WHERE `post_id` = ? ORDER BY `created_at` DESC"
		if !allComments {
			query += " LIMIT 3"
		}
		var comments []Comment
		err = db.Select(&comments, query, p.ID)
		if err != nil {
			return nil, err
		}

		for i := 0; i < len(comments); i++ {
			err := db.Get(&comments[i].User, "SELECT * FROM `users` WHERE `id` = ?", comments[i].UserID)
			if err != nil {
				return nil, err
			}
		}

		// reverse
		for i, j := 0, len(comments)-1; i < j; i, j = i+1, j-1 {
			comments[i], comments[j] = comments[j], comments[i]
		}

		p.Comments = comments

		err = db.Get(&p.User, "SELECT * FROM `users` WHERE `id` = ?", p.UserID)
		if err != nil {
			return nil, err
		}

		p.CSRFToken = csrfToken

		if p.User.DelFlg == 0 {
			posts = append(posts, p)
		}
		if len(posts) >= postsPerPage {
			break
		}
	}

	return posts, nil
}

func imageURL(p Post) string {
	ext := ""
	if p.Mime == "image/jpeg" {
		ext = ".jpg"
	} else if p.Mime == "image/png" {
		ext = ".png"
	} else if p.Mime == "image/gif" {
		ext = ".gif"
	}

	return "/image/" + strconv.Itoa(p.ID) + ext
}

func isLogin(u User) bool {
	return u.ID != 0
}

func getCSRFToken(c *fiber.Ctx) string {
	session := getSession(c)
	csrfToken, ok := session.Values["csrf_token"]
	if !ok {
		return ""
	}
	return csrfToken.(string)
}

func secureRandomStr(b int) string {
	k := make([]byte, b)
	if _, err := crand.Read(k); err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", k)
}

func getInitialize(c *fiber.Ctx) error {
	dbInitialize()
	return c.SendStatus(fiber.StatusOK)
}

func getLogin(c *fiber.Ctx) error {
	me := getSessionUser(c)

	if isLogin(me) {
		return c.Redirect("/", fiber.StatusFound)
	}

	return c.Render("login", fiber.Map{
		"Me":    me,
		"Flash": getFlash(c, "notice"),
	})
}

func postLogin(c *fiber.Ctx) error {
	if isLogin(getSessionUser(c)) {
		return c.Redirect("/", fiber.StatusFound)
	}

	u := tryLogin(c.FormValue("account_name"), c.FormValue("password"))

	if u != nil {
		session := getSession(c)
		session.Values["user_id"] = u.ID
		session.Values["csrf_token"] = secureRandomStr(16)
		saveSession(c, session)

		return c.Redirect("/", fiber.StatusFound)
	} else {
		session := getSession(c)
		session.Values["notice"] = "アカウント名かパスワードが間違っています"
		saveSession(c, session)

		return c.Redirect("/login", fiber.StatusFound)
	}
}

func getRegister(c *fiber.Ctx) error {
	if isLogin(getSessionUser(c)) {
		return c.Redirect("/", fiber.StatusFound)
	}

	return c.Render("register", fiber.Map{
		"Me":    User{},
		"Flash": getFlash(c, "notice"),
	})
}

func postRegister(c *fiber.Ctx) error {
	if isLogin(getSessionUser(c)) {
		return c.Redirect("/", fiber.StatusFound)
	}

	accountName, password := c.FormValue("account_name"), c.FormValue("password")

	validated := validateUser(accountName, password)
	if !validated {
		session := getSession(c)
		session.Values["notice"] = "アカウント名は3文字以上、パスワードは6文字以上である必要があります"
		saveSession(c, session)

		return c.Redirect("/register", fiber.StatusFound)
	}

	exists := 0
	// ユーザーが存在しない場合はエラーになるのでエラーチェックはしない
	db.Get(&exists, "SELECT 1 FROM users WHERE `account_name` = ?", accountName)

	if exists == 1 {
		session := getSession(c)
		session.Values["notice"] = "アカウント名がすでに使われています"
		saveSession(c, session)

		return c.Redirect("/register", fiber.StatusFound)
	}

	query := "INSERT INTO `users` (`account_name`, `passhash`) VALUES (?,?)"
	result, err := db.Exec(query, accountName, calculatePasshash(accountName, password))
	if err != nil {
		log.Print(err)
		return err
	}

	uid, err := result.LastInsertId()
	if err != nil {
		log.Print(err)
		return err
	}

	session := getSession(c)
	session.Values["user_id"] = uid
	session.Values["csrf_token"] = secureRandomStr(16)
	saveSession(c, session)
	return c.Redirect("/", fiber.StatusFound)
}

func getLogout(c *fiber.Ctx) error {
	session := getSession(c)
	delete(session.Values, "user_id")
	session.Options = &sessions.Options{MaxAge: -1}
	saveSession(c, session)

	return c.Redirect("/", fiber.StatusFound)
}

func getIndex(c *fiber.Ctx) error {
	me := getSessionUser(c)

	results := []Post{}

	err := db.Select(&results, "SELECT `id`, `user_id`, `body`, `mime`, `created_at` FROM `posts` ORDER BY `created_at` DESC")
	if err != nil {
		log.Print(err)
		return err
	}

	posts, err := makePosts(results, getCSRFToken(c), false)
	if err != nil {
		log.Print(err)
		return err
	}

	return c.Render("layout", fiber.Map{
		"Posts":     posts,
		"Me":        me,
		"CSRFToken": getCSRFToken(c),
		"Flash":     getFlash(c, "notice"),
	})
}

func getAccountName(c *fiber.Ctx) error {
	accountName := c.Params("accountName")
	user := User{}

	err := db.Get(&user, "SELECT * FROM `users` WHERE `account_name` = ? AND `del_flg` = 0", accountName)
	if err != nil {
		log.Print(err)
		return err
	}

	if user.ID == 0 {
		return c.SendStatus(fiber.StatusNotFound)
	}

	results := []Post{}

	err = db.Select(&results, "SELECT `id`, `user_id`, `body`, `mime`, `created_at` FROM `posts` WHERE `user_id` = ? ORDER BY `created_at` DESC", user.ID)
	if err != nil {
		log.Print(err)
		return err
	}

	posts, err := makePosts(results, getCSRFToken(c), false)
	if err != nil {
		log.Print(err)
		return err
	}

	commentCount := 0
	err = db.Get(&commentCount, "SELECT COUNT(*) AS count FROM `comments` WHERE `user_id` = ?", user.ID)
	if err != nil {
		log.Print(err)
		return err
	}

	postIDs := []int{}
	err = db.Select(&postIDs, "SELECT `id` FROM `posts` WHERE `user_id` = ?", user.ID)
	if err != nil {
		log.Print(err)
		return err
	}
	postCount := len(postIDs)

	commentedCount := 0
	if postCount > 0 {
		s := []string{}
		for range postIDs {
			s = append(s, "?")
		}
		placeholder := strings.Join(s, ", ")

		// convert []int -> []interface{}
		args := make([]interface{}, len(postIDs))
		for i, v := range postIDs {
			args[i] = v
		}

		err = db.Get(&commentedCount, "SELECT COUNT(*) AS count FROM `comments` WHERE `post_id` IN ("+placeholder+")", args...)
		if err != nil {
			log.Print(err)
			return err
		}
	}

	me := getSessionUser(c)

	return c.Render("layout", fiber.Map{
		"Posts":          posts,
		"User":           user,
		"PostCount":      postCount,
		"CommentCount":   commentCount,
		"CommentedCount": commentedCount,
		"Me":             me,
	})
}

func getPosts(c *fiber.Ctx) error {
	m, err := url.ParseQuery(string(c.Request().URI().QueryString()))
	if err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}
	maxCreatedAt := m.Get("max_created_at")
	if maxCreatedAt == "" {
		return nil
	}

	t, err := time.Parse(ISO8601Format, maxCreatedAt)
	if err != nil {
		log.Print(err)
		return err
	}

	results := []Post{}
	err = db.Select(&results, "SELECT `id`, `user_id`, `body`, `mime`, `created_at` FROM `posts` WHERE `created_at` <= ? ORDER BY `created_at` DESC", t.Format(ISO8601Format))
	if err != nil {
		log.Print(err)
		return err
	}

	posts, err := makePosts(results, getCSRFToken(c), false)
	if err != nil {
		log.Print(err)
		return err
	}

	if len(posts) == 0 {
		return c.SendStatus(fiber.StatusNotFound)
	}

	return c.Render("posts", fiber.Map{
		"Posts": posts,
	})
}

func getPostsID(c *fiber.Ctx) error {
	pidStr := c.Params("id")
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return c.SendStatus(fiber.StatusNotFound)
	}

	results := []Post{}
	err = db.Select(&results, "SELECT * FROM `posts` WHERE `id` = ?", pid)
	if err != nil {
		log.Print(err)
		return err
	}

	posts, err := makePosts(results, getCSRFToken(c), true)
	if err != nil {
		log.Print(err)
		return err
	}

	if len(posts) == 0 {
		return c.SendStatus(fiber.StatusNotFound)
	}
	me := getSessionUser(c)

	return c.Render("layout", fiber.Map{
		"Posts": posts,
		"imageURL": imageURL,
		"Me":    me,
	})
}

func postIndex(c *fiber.Ctx) error {
	me := getSessionUser(c)
	if !isLogin(me) {
		return c.Redirect("/login", fiber.StatusFound)
	}

	if c.FormValue("csrf_token") != getCSRFToken(c) {
		return c.SendStatus(fiber.StatusUnprocessableEntity)
	}

	file, err := c.FormFile("file")
	if err != nil {
		session := getSession(c)

		session.Values["notice"] = "画像が必須です"
		saveSession(c, session)

		return c.Redirect("/", fiber.StatusFound)
	}

	mime := ""
	if file != nil {
		// 投稿のContent-Typeからファイルのタイプを決定する
		contentType := file.Header["Content-Type"][0]
		if strings.Contains(contentType, "jpeg") {
			mime = "image/jpeg"
		} else if strings.Contains(contentType, "png") {
			mime = "image/png"
		} else if strings.Contains(contentType, "gif") {
			mime = "image/gif"
		} else {
			session := getSession(c)
			session.Values["notice"] = "投稿できる画像形式はjpgとpngとgifだけです"
			saveSession(c, session)

			return c.Redirect("/", fiber.StatusFound)
		}
	}

	filedata, err := file.Open()
	if err != nil {
		log.Print(err)
		return err
	}

	fileInfo := file.Size

	if fileInfo > UploadLimit {
		session := getSession(c)

		session.Values["notice"] = "ファイルサイズが大きすぎます"
		saveSession(c, session)

		return c.Redirect("/", fiber.StatusFound)
	}

	query := "INSERT INTO `posts` (`user_id`, `mime`, `imgdata`, `body`) VALUES (?,?,?,?)"
	result, err := db.Exec(
		query,
		me.ID,
		mime,
		filedata,
		c.FormValue("body"),
	)
	if err != nil {
		log.Print(err)
		return err
	}

	pid, err := result.LastInsertId()
	if err != nil {
		log.Print(err)
		return err
	}

	return c.Redirect("/posts/" + strconv.FormatInt(pid, 10))
}

func getImage(c *fiber.Ctx) error {
	pidStr := c.Params("id")
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return c.SendStatus(fiber.StatusNotFound)
	}

	post := Post{}
	err = db.Get(&post, "SELECT * FROM `posts` WHERE `id` = ?", pid)
	if err != nil {
		log.Print(err)
		return err
	}

	ext := c.Params("ext")

	if ext == "jpg" && post.Mime == "image/jpeg" ||
		ext == "png" && post.Mime == "image/png" ||
			ext == "gif" && post.Mime == "image/gif" {
		c.Set("Content-Type", post.Mime)
		err = c.Send(post.Imgdata)
		if err != nil {
			log.Print(err)
			return err
		}
		return nil
	}

	return c.SendStatus(fiber.StatusNotFound)
}

func postComment(c *fiber.Ctx) error {
	me := getSessionUser(c)
	if !isLogin(me) {
		return c.Redirect("/login", fiber.StatusFound)
	}

	if c.FormValue("csrf_token") != getCSRFToken(c) {
		return c.SendStatus(fiber.StatusUnprocessableEntity)
	}

	postID, err := strconv.Atoi(c.FormValue("post_id"))
	if err != nil {
		log.Print("post_idは整数のみです")
		return err
	}

	query := "INSERT INTO `comments` (`post_id`, `user_id`, `comment`) VALUES (?,?,?)"
	_, err = db.Exec(query, postID, me.ID, c.FormValue("comment"))
	if err != nil {
		log.Print(err)
		return err
	}

	return c.Redirect("/posts/" + strconv.Itoa(postID), fiber.StatusFound)
}

func getAdminBanned(c *fiber.Ctx) error {
	me := getSessionUser(c)
	if !isLogin(me) {
		return c.Redirect("/", fiber.StatusFound)
	}

	if me.Authority == 0 {
		return c.SendStatus(fiber.StatusForbidden)
	}

	users := []User{}
	err := db.Select(&users, "SELECT * FROM `users` WHERE `authority` = 0 AND `del_flg` = 0 ORDER BY `created_at` DESC")
	if err != nil {
		log.Print(err)
		return err
	}

	return c.Render("banned", fiber.Map{
		"Users":     users,
		"Me":        me,
		"CSRFToken": getCSRFToken(c),
	})
}

type UidsBody struct {
	Uids []string `query:"uid" json:"uid" xml:"uid" form:"uid"`
}

func postAdminBanned(c *fiber.Ctx) error {
	me := getSessionUser(c)
	if !isLogin(me) {
		return c.Redirect("/", fiber.StatusFound)
	}

	if me.Authority == 0 {
		return c.SendStatus(fiber.StatusForbidden)
	}

	if c.FormValue("csrf_token") != getCSRFToken(c) {
		return c.SendStatus(fiber.StatusUnprocessableEntity)
	}

	query := "UPDATE `users` SET `del_flg` = ? WHERE `id` = ?"


	uids := UidsBody{}
	if err := c.BodyParser(&uids); err != nil {
		log.Print(err)
		return err
	}

	for _, id := range uids.Uids {
		db.Exec(query, 1, id)
	}

	return c.Redirect("/admin/banned", fiber.StatusFound)
}

func main() {
	// DB設定
	host := os.Getenv("ISUCONP_DB_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("ISUCONP_DB_PORT")
	if port == "" {
		port = "3306"
	}
	user := os.Getenv("ISUCONP_DB_USER")
	if user == "" {
		user = "root"
	}
	password := os.Getenv("ISUCONP_DB_PASSWORD")
	dbname := os.Getenv("ISUCONP_DB_NAME")
	if dbname == "" {
		dbname = "isuconp"
	}

	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&loc=Local",
		user,
		password,
		host,
		port,
		dbname,
	)

	var err error
	db, err = sqlx.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %s.", err.Error())
	}
	defer db.Close()

	// Fiberアプリケーションの設定
	engine := html.New("./templates", ".html")
	engine.AddFunc("imageURL", imageURL)
	engine.Layout("layout") // レイアウトの設定
	engine.Load()
	app := fiber.New(fiber.Config{
		Views: engine,
		Prefork: true,
	})
	app.Use(logger.New())

	// ルーティング
	app.Get("/initialize", getInitialize)
	app.Get("/login", getLogin)
	app.Post("/login", postLogin)
	app.Get("/register", getRegister)
	app.Post("/register", postRegister)
	app.Get("/logout", getLogout)
	app.Get("/", getIndex)
	app.Get("/posts", getPosts)
	app.Get("/posts/:id", getPostsID)
	app.Post("/", postIndex)
	app.Get("/image/:id.:ext", getImage)
	app.Post("/comment", postComment)
	app.Get("/admin/banned", getAdminBanned)
	app.Post("/admin/banned", postAdminBanned)
	app.Get("/@:accountName", getAccountName)
	app.Static("/*", "../public")

	log.Fatal(app.Listen(":8080"))
}
