use std::{
    any,
    collections::HashMap,
    env,
    io::{self, Write},
    path::Path,
    time::Duration,
};

use actix_cors::Cors;
use actix_files::Files;
use actix_redis::RedisSession;
use actix_session::Session;
use actix_web::{
    cookie::time::UtcOffset,
    dev::ResourceDef,
    error, get,
    http::{
        header::{self, ContentType},
        Method, StatusCode,
    },
    middleware, post,
    web::{self, Data, Form},
    App, HttpRequest, HttpResponse, HttpServer, Result,
};
use anyhow::{bail, Context};
use chrono::{DateTime, FixedOffset, Local, Utc};
use derive_more::Constructor;
use duct::cmd;
use handlebars::{handlebars_helper, to_json, Handlebars};
use log::LevelFilter;
use once_cell::sync::Lazy;
use rand::{
    prelude::{SliceRandom, StdRng},
    thread_rng, SeedableRng,
};
use regex::Regex;
use serde::{Deserialize, Serialize};
use serde_json::Map;
use simplelog::{
    ColorChoice, CombinedLogger, ConfigBuilder, SharedLogger, TermLogger, TerminalMode, WriteLogger,
};
use sqlx::{MySql, Pool};

const POSTS_PER_PAGE: usize = 20;
static AGGREGATION_LOWER_CASE_NUM: Lazy<Vec<char>> = Lazy::new(|| {
    let mut az09 = Vec::new();
    for az in 'a' as u32..('z' as u32 + 1) {
        az09.push(char::from_u32(az).unwrap());
    }
    for s09 in '0' as u32..('9' as u32 + 1) {
        az09.push(char::from_u32(s09).unwrap());
    }

    az09
});

#[derive(Debug, Serialize, Deserialize, Constructor)]
struct User {
    id: i32,
    account_name: String,
    passhash: String,
    authority: i8,
    del_flg: i8,
    created_at: chrono::DateTime<Utc>,
}

impl Default for User {
    fn default() -> Self {
        Self {
            id: Default::default(),
            account_name: Default::default(),
            passhash: Default::default(),
            authority: Default::default(),
            del_flg: Default::default(),
            created_at: Utc::now(),
        }
    }
}

#[derive(Debug, Serialize, Deserialize, Constructor)]
struct Post {
    id: i32,
    user_id: i32,
    imgdata: Vec<u8>,
    body: String,
    mime: String,
    created_at: chrono::DateTime<Utc>,
}

#[derive(Debug, Serialize, Deserialize, Constructor)]
struct GrantedInfoPost {
    post: Post,
    comment_count: i64,
    comments: Vec<GrantedUserComment>,
    user: User,
    csrf_token: String,
}

#[derive(Debug, Serialize, Deserialize, Constructor)]
struct Comment {
    id: i32,
    post_id: i32,
    user_id: i32,
    comment: String,
    created_at: chrono::DateTime<Utc>,
}

#[derive(Debug, Serialize, Deserialize, Constructor)]
struct GrantedUserComment {
    comment: Comment,
    user: User,
}

#[derive(Debug, Serialize, Deserialize)]
struct LoginRegisterParams {
    account_name: String,
    password: String,
}

async fn db_initialize(pool: &Pool<MySql>) -> anyhow::Result<()> {
    sqlx::query!("DELETE FROM users WHERE id > 1000")
        .execute(pool)
        .await
        .context("Failed to db_initialize")?;
    sqlx::query!("DELETE FROM posts WHERE id > 10000",)
        .execute(pool)
        .await
        .context("Failed to db_initialize")?;
    sqlx::query!("DELETE FROM comments WHERE id > 100000")
        .execute(pool)
        .await
        .context("Failed to db_initialize")?;
    sqlx::query!("UPDATE users SET del_flg = 0")
        .execute(pool)
        .await
        .context("Failed to db_initialize")?;
    sqlx::query!("UPDATE users SET del_flg = 1 WHERE id % 50 = 0")
        .execute(pool)
        .await
        .context("Failed to db_initialize")?;

    Ok(())
}

async fn try_login(account_name: &str, password: &str, pool: &Pool<MySql>) -> anyhow::Result<User> {
    let user = sqlx::query_as!(
        User,
        "SELECT * FROM users WHERE account_name = ? AND del_flg = 0",
        account_name
    )
    .fetch_optional(pool)
    .await
    .context("Failed to query try_login")?;

    if let Some(user) = user {
        if calculate_passhash(&user.account_name, password)? == user.passhash {
            Ok(user)
        } else {
            bail!("Incorrect password");
        }
    } else {
        bail!("User does not exist");
    }
}

fn escapeshellarg(arg: &str) -> String {
    format!("'{}'", arg.replace("'", "'\\''"))
}

fn digest(src: &str) -> anyhow::Result<String> {
    let output = cmd!(
        "/bin/bash",
        "-c",
        format!(
            r#"printf "%s" "+{}+" | openssl dgst -sha512 | sed 's/^.*= //'"#,
            escapeshellarg(src)
        )
    )
    .read()
    .context("Failed to cmd")?;

    Ok(output.trim_end_matches("\n").to_string())
}

fn validate_user(account_name: &str, password: &str) -> bool {
    let name_regex = Regex::new(r"\A[0-9a-zA-Z_]{3,}\z").unwrap();
    let pass_regex = Regex::new(r"\A[0-9a-zA-Z_]{6,}\z").unwrap();

    name_regex.is_match(account_name) && pass_regex.is_match(password)
}

#[get("/initialize")]
async fn get_initialize(pool: Data<Pool<MySql>>) -> Result<HttpResponse> {
    if let Err(e) = db_initialize(&pool).await {
        log::error!("{:?}", &e);
    }
    Ok(HttpResponse::Ok().finish())
}

fn calculate_salt(account_name: &str) -> anyhow::Result<String> {
    digest(account_name)
}

fn calculate_passhash(account_name: &str, password: &str) -> anyhow::Result<String> {
    digest(&format!("{}:{}", password, calculate_salt(account_name)?))
}

async fn get_session_user(session: &Session, pool: &Pool<MySql>) -> anyhow::Result<Option<User>> {
    let uid = match session.get::<i32>("user_id") {
        Ok(Some(uid)) => uid,
        Err(e) => bail!("Failed to get_session_user {}", &e),
        _ => return Ok(None),
    };

    let user = sqlx::query_as!(User, "SELECT * FROM `users` WHERE `id` = ?", &uid)
        .fetch_optional(pool)
        .await
        .context("Failed to get_session_user")?;
    log::debug!("query user");

    Ok(user)
}

fn get_flash(session: &Session, key: &str) -> Option<String> {
    match session.get(key) {
        Ok(Some(value)) => {
            session.remove(key);
            value
        }
        Err(e) => {
            log::error!("{:?}", &e);
            None
        }
        _ => None,
    }
}

async fn make_post(
    results: Vec<Post>,
    csrf_token: String,
    all_comments: bool,
    pool: &Pool<MySql>,
) -> anyhow::Result<Vec<GrantedInfoPost>> {
    let mut granted_info_posts = Vec::new();

    for p in results {
        let comment_count = sqlx::query!(
            "SELECT COUNT(*) AS `count` FROM `comments` WHERE `post_id` = ?",
            p.id
        )
        .fetch_one(pool)
        .await
        .context("Failed to query comment_count")?
        .count;

        let mut comments = if all_comments {
            sqlx::query_as!(
                Comment,
                "SELECT * FROM `comments` WHERE `post_id` = ? ORDER BY `created_at` DESC",
                p.id
            )
            .fetch_all(pool)
            .await
        } else {
            sqlx::query_as!(
                Comment,
                "SELECT * FROM `comments` WHERE `post_id` = ? ORDER BY `created_at` DESC LIMIT 3",
                p.id
            )
            .fetch_all(pool)
            .await
        }
        .context("Failed to query comments")?;

        let mut granted_comments = Vec::new();

        for comment in comments {
            let user = sqlx::query_as!(
                User,
                "SELECT * FROM `users` WHERE `id` = ?",
                comment.user_id
            )
            .fetch_optional(pool)
            .await
            .context("Failed to query user")?
            .context("Not found user")?;
            log::debug!("comment user {:?}", &user);

            granted_comments.push(GrantedUserComment::new(comment, user));
        }

        granted_comments.reverse();

        let user = sqlx::query_as!(User, "SELECT * FROM `users` WHERE `id` = ?", p.user_id)
            .fetch_optional(pool)
            .await
            .context("Failed to query user")?
            .context("Not found user")?;
        log::debug!("user {:?}", &user);

        if user.del_flg == 0 {
            granted_info_posts.push(GrantedInfoPost::new(
                p,
                comment_count,
                granted_comments,
                user,
                csrf_token.clone(),
            ))
        }
        if granted_info_posts.len() >= POSTS_PER_PAGE {
            break;
        }
    }

    Ok(granted_info_posts)
}

// fn image_url(p: &GrantedInfoPost) -> String {
//     let ext = match p.post.mime.as_str() {
//         "image/jpeg" => ".jpg",
//         "image/png" => ".png",
//         "image/gif" => ".gif",
//         _ => "",
//     };

//     format!("/image/{}{}", p.post.id, ext)
// }

handlebars_helper!(image_url: |p: GrantedInfoPost| {
    let ext = match p.post.mime.as_str() {
            "image/jpeg" => ".jpg",
            "image/png" => ".png",
            "image/gif" => ".gif",
            _ => "",
        };

    format!("/image/{}{}", p.post.id, ext)
});

handlebars_helper!(date_time_format: |create_at: DateTime<Utc>| {
    create_at.format("%Y-%m-%dT%H:%M:%S-07:00").to_string()
});

fn is_login(u: Option<&User>) -> bool {
    match u {
        Some(u) => u.id != 0,
        None => false,
    }
}

fn get_csrf_token(session: &Session) -> Option<String> {
    session.get("csrf_token").unwrap_or_default()
}

// goと違い文字数指定
fn secure_random_str(b: u32) -> String {
    let mut rng = StdRng::from_rng(thread_rng()).unwrap();

    let mut rnd_str = Vec::new();
    for _ in 0..b {
        rnd_str.push(AGGREGATION_LOWER_CASE_NUM.choose(&mut rng).unwrap());
    }

    let rnd_str = rnd_str.iter().map(|c| *c).collect();

    rnd_str
}

#[get("/login")]
async fn get_login(
    session: Session,
    pool: Data<Pool<MySql>>,
    handlebars: Data<Handlebars<'_>>,
) -> Result<HttpResponse> {
    let user = match get_session_user(&session, pool.as_ref()).await {
        Ok(user) => {
            if is_login(user.as_ref()) {
                return Ok(HttpResponse::Found()
                    .insert_header((header::LOCATION, "/"))
                    .finish());
            }

            if let Some(user) = user {
                user
            } else {
                User::default()
            }
        }
        Err(e) => {
            log::error!("{:?}", &e);
            User::default()
        }
    };

    let body = {
        let mut map = Map::new();

        map.insert("user".to_string(), to_json(user));
        map.insert("flash".to_string(), to_json(get_flash(&session, "notice")));
        map.insert("parent".to_string(), to_json("layout"));
        log::debug!("{:?}", &map);

        handlebars.render("login", &map).unwrap()
    };
    log::debug!("{:?}", &body);

    Ok(HttpResponse::Ok().body(body))
}

#[post("/login")]
async fn post_login(
    session: Session,
    pool: Data<Pool<MySql>>,
    params: Form<LoginRegisterParams>,
) -> Result<HttpResponse> {
    match get_session_user(&session, pool.as_ref()).await {
        Ok(user) => {
            if is_login(user.as_ref()) {
                return Ok(HttpResponse::Found()
                    .insert_header((header::LOCATION, "/"))
                    .finish());
            }
        }
        Err(e) => log::error!("{:?}", &e),
    };

    match try_login(&params.account_name, &params.password, pool.as_ref()).await {
        Ok(user) => {
            session.insert("user_id", user.id).unwrap();
            session.insert("csrf_token", secure_random_str(32)).unwrap();

            Ok(HttpResponse::Found()
                .insert_header((header::LOCATION, "/"))
                .finish())
        }
        Err(e) => {
            log::error!("{:?}", &e);
            session
                .insert("notice", "アカウント名かパスワードが間違っています")
                .unwrap();

            Ok(HttpResponse::Found()
                .insert_header((header::LOCATION, "/login"))
                .finish())
        }
    }
}

#[get("/register")]
async fn get_register(
    session: Session,
    pool: Data<Pool<MySql>>,
    handlebars: Data<Handlebars<'_>>,
) -> Result<HttpResponse> {
    log::debug!("call get_register");
    match get_session_user(&session, pool.as_ref()).await {
        Ok(user) => {
            if is_login(user.as_ref()) {
                return Ok(HttpResponse::Found()
                    .insert_header((header::LOCATION, "/"))
                    .finish());
            }
        }
        Err(e) => log::error!("{:?}", &e),
    };

    log::debug!("render template");
    let body = {
        let user = User::default();

        let mut map = Map::new();

        map.insert("user".to_string(), to_json(user));
        map.insert("flash".to_string(), to_json(get_flash(&session, "notice")));
        map.insert("parent".to_string(), to_json("layout"));
        log::debug!("map {:?}", &map);

        handlebars.render("register", &map).unwrap()
    };

    log::debug!("return ok");
    Ok(HttpResponse::Ok().body(body))
}

#[post("/register")]
async fn post_register(
    session: Session,
    pool: Data<Pool<MySql>>,
    params: Form<LoginRegisterParams>,
) -> Result<HttpResponse> {
    match get_session_user(&session, pool.as_ref()).await {
        Ok(user) => {
            if is_login(user.as_ref()) {
                return Ok(HttpResponse::Found()
                    .insert_header((header::LOCATION, "/"))
                    .finish());
            }
        }
        Err(e) => log::error!("{:?}", &e),
    };

    let validated = validate_user(&params.account_name, &params.password);
    if !validated {
        if let Err(e) = session.insert(
            "notice",
            "アカウント名は3文字以上、パスワードは6文字以上である必要があります",
        ) {
            log::error!("{:?}", &e);
            return Ok(HttpResponse::InternalServerError().body(e.to_string()));
        } else {
            return Ok(HttpResponse::Found()
                .insert_header((header::LOCATION, "/register"))
                .finish());
        }
    }

    let exists = match sqlx::query!(
        "SELECT 1 AS _exists FROM users WHERE `account_name` = ?",
        &params.account_name
    )
    .fetch_optional(pool.as_ref())
    .await
    {
        Ok(exists) => exists,
        Err(e) => {
            log::error!("{:?}", &e);
            return Ok(HttpResponse::InternalServerError().body(e.to_string()));
        }
    };

    if let Some(_) = exists {
        if let Err(e) = session.insert("notice", "アカウント名がすでに使われています")
        {
            log::error!("{:?}", &e);
            return Ok(HttpResponse::InternalServerError().body(e.to_string()));
        } else {
            return Ok(HttpResponse::Found()
                .insert_header((header::LOCATION, "/register"))
                .finish());
        }
    }

    let pass_hash = match calculate_passhash(&params.account_name, &params.password) {
        Ok(p) => p,
        Err(e) => {
            log::error!("{:?}", &e);
            return Ok(HttpResponse::InternalServerError().body(e.to_string()));
        }
    };
    let uid = match sqlx::query!(
        "INSERT INTO `users` (`account_name`, `passhash`) VALUES (?,?)",
        &params.account_name,
        pass_hash
    )
    .execute(pool.as_ref())
    .await
    {
        Ok(r) => r.last_insert_id(),
        Err(e) => {
            log::error!("{:?}", &e);
            return Ok(HttpResponse::InternalServerError().body(e.to_string()));
        }
    };
    log::debug!("last insert id {}", &uid);

    if let Err(e) = session.insert("user_id", uid) {
        log::error!("{:?}", &e);
        return Ok(HttpResponse::InternalServerError().body(e.to_string()));
    }
    if let Err(e) = session.insert("csrf_token", secure_random_str(32)) {
        log::error!("{:?}", &e);
        return Ok(HttpResponse::InternalServerError().body(e.to_string()));
    }

    Ok(HttpResponse::Found()
        .insert_header((header::LOCATION, "/"))
        .finish())
}

#[get("/logout")]
async fn get_logout(session: Session) -> Result<HttpResponse> {
    session.remove("user_id").unwrap_or_default();

    Ok(HttpResponse::Found()
        .insert_header((header::LOCATION, "/"))
        .finish())
}

#[get("/")]
async fn get_index(
    session: Session,
    pool: Data<Pool<MySql>>,
    handlebars: Data<Handlebars<'_>>,
) -> Result<HttpResponse> {
    let me = match get_session_user(&session, pool.as_ref()).await {
        Ok(user) => user.unwrap_or_default(),
        Err(e) => {
            log::error!("{:?}", &e);
            return Ok(HttpResponse::InternalServerError().body(e.to_string()));
        }
    };

    let results = match sqlx::query_as!(Post,"SELECT `id`, `user_id`, `body`, `mime`, `created_at`, b'0' AS imgdata FROM `posts` ORDER BY `created_at` DESC").fetch_all(pool.as_ref()).await {
        Ok(results) => results,
        Err(e) => {
            log::error!("{:?}",&e);
            return Ok(HttpResponse::InternalServerError().body(e.to_string()));
        }
    };

    let csrf_token = get_csrf_token(&session).unwrap_or_default();
    // let csrf_token = if let Some(token) = get_csrf_token(&session) {
    //     token
    // } else {
    //     log::error!("token is None");
    //     return Ok(HttpResponse::InternalServerError().finish());
    // };
    let posts = match make_post(results, csrf_token, false, pool.as_ref()).await {
        Ok(posts) => posts,
        Err(e) => {
            log::error!("{:?}", &e);
            return Ok(HttpResponse::InternalServerError().body(e.to_string()));
        }
    };

    let body = {
        let mut map = Map::new();
        // let posts = serde_json::to_value(posts).unwrap();
        // let map = json.as_object_mut().unwrap();
        map.insert("posts".to_string(), to_json(posts));
        map.insert("user".to_string(), to_json(me));
        map.insert(
            "csrf_token".to_string(),
            to_json(get_csrf_token(&session).unwrap_or_default()),
        );
        map.insert("flash".to_string(), to_json(get_flash(&session, "notice")));

        map.insert("post_parent".to_string(), to_json("posts"));
        map.insert("posts_parent".to_string(), to_json("index"));
        map.insert("content_parent".to_string(), to_json("layout"));

        handlebars.render("post", &map).unwrap()
    };

    Ok(HttpResponse::Ok().body(body))
}

#[get("/posts")]
async fn get_posts() -> Result<HttpResponse> {
    todo!()
}

#[get("/posts/{id}")]
async fn get_posts_id(
    pid: web::Path<(u64,)>,
    session: Session,
    pool: Data<Pool<MySql>>,
    handlebars: Data<Handlebars<'_>>,
) -> Result<HttpResponse> {
    let results = match sqlx::query_as!(Post, "SELECT * FROM `posts` WHERE `id` = ?", pid.0)
        .fetch_all(pool.as_ref())
        .await
    {
        Ok(r) => r,
        Err(e) => {
            log::error!("{:?}", &e);
            return Ok(HttpResponse::InternalServerError().body(e.to_string()));
        }
    };

    let posts = match make_post(
        results,
        get_csrf_token(&session).unwrap_or_default(),
        true,
        pool.as_ref(),
    )
    .await
    {
        Ok(p) => p,
        Err(e) => {
            log::error!("{:?}", &e);
            return Ok(HttpResponse::InternalServerError().body(e.to_string()));
        }
    };

    if posts.is_empty() {
        return Ok(HttpResponse::NotFound().finish());
    }

    let p = &posts[0];

    let me = match get_session_user(&session, pool.as_ref()).await {
        Ok(u) => u.unwrap_or_default(),
        Err(e) => {
            log::error!("{:?}", &e);
            return Ok(HttpResponse::InternalServerError().body(e.to_string()));
        }
    };

    let body = {
        // let mut map = Map::new();
        // let posts = serde_json::to_value(posts).unwrap();
        // let map = json.as_object_mut().unwrap();
        // map.insert("post".to_string(), to_json(p));

        let mut post = serde_json::to_value(p).unwrap();
        let map = post.as_object_mut().unwrap();
        map.insert("user".to_string(), to_json(me));

        map.insert("post_parent".to_string(), to_json("post_id"));
        map.insert("content_parent".to_string(), to_json("layout"));

        // let mut file = std::fs::File::create(".tmp/get_posts_id.json").unwrap();
        // write!(file, "{:#?}", &map).unwrap();

        handlebars.render("post", &map).unwrap()
    };

    Ok(HttpResponse::Ok().body(body))
}

async fn post_index() -> Result<HttpResponse> {
    todo!()
}

#[get("/image/{path}")]
async fn get_image(path: web::Path<(String,)>, pool: Data<Pool<MySql>>) -> Result<HttpResponse> {
    let (pid, ext) = match path.0.rsplit_once(".") {
        Some((pid, ext)) => {
            let pid = match pid.parse::<u64>() {
                Ok(pid) => pid,
                Err(e) => {
                    log::warn!("{:?}", &e);
                    return Ok(HttpResponse::InternalServerError().body(e.to_string()));
                }
            };
            (pid, ext)
        }
        None => {
            let e = "Invalid path";
            log::warn!("{}", e);
            return Ok(HttpResponse::InternalServerError().body(e));
        }
    };

    let post = match sqlx::query_as!(Post, "SELECT * FROM `posts` WHERE `id` = ?", pid)
        .fetch_optional(pool.as_ref())
        .await
    {
        Ok(Some(post)) => post,
        Err(e) => {
            log::warn!("{:?}", &e);
            return Ok(HttpResponse::InternalServerError().body(e.to_string()));
        }
        _ => {
            return Ok(HttpResponse::InternalServerError().finish());
        }
    };

    if ext == "jpg" && post.mime == "image/jpeg"
        || ext == "png" && post.mime == "image/png"
        || ext == "gif" && post.mime == "image/gif"
    {}

    let content_type = match (ext, post.mime.as_str()) {
        ("jpg", "image/jpeg") | ("png", "image/png") | ("gif", "image/gif") => post.mime.as_str(),
        _ => return Ok(HttpResponse::InternalServerError().finish()),
    };

    Ok(HttpResponse::Ok()
        .content_type(content_type)
        .body(post.imgdata))
}

async fn post_comment() -> Result<HttpResponse> {
    todo!()
}

async fn get_admin_banned() -> Result<HttpResponse> {
    todo!()
}

async fn post_admin_banned() -> Result<HttpResponse> {
    todo!()
}

fn init_logger<P: AsRef<Path>>(log_path: Option<P>) {
    const JST_UTCOFFSET_SECS: i32 = 9 * 3600;

    let jst_now = {
        let jst = Utc::now();
        jst.with_timezone(&FixedOffset::east(JST_UTCOFFSET_SECS))
    };

    let offset = UtcOffset::from_whole_seconds(JST_UTCOFFSET_SECS).unwrap();

    let mut config = ConfigBuilder::new();
    config.set_time_offset(offset);

    let mut logger: Vec<Box<dyn SharedLogger>> = vec![
        #[cfg(not(feature = "termcolor"))]
        TermLogger::new(
            if cfg!(debug_assertions) {
                LevelFilter::Debug
            } else {
                LevelFilter::Info
            },
            config.build(),
            TerminalMode::Mixed,
            ColorChoice::Always,
        ),
    ];
    if let Some(log_path) = log_path {
        let log_path = log_path.as_ref();
        std::fs::create_dir_all(&log_path).unwrap();
        logger.push(WriteLogger::new(
            LevelFilter::Info,
            config.build(),
            std::fs::File::create(log_path.join(format!("{}.log", jst_now))).unwrap(),
        ));
    }
    CombinedLogger::init(logger).unwrap()
}

#[actix_web::main]
async fn main() -> io::Result<()> {
    init_logger::<&str>(None);

    let host = env::var("ISUCONP_DB_HOST").unwrap_or("localhost".to_string());
    let port: u32 = env::var("ISUCONP_DB_PORT")
        .unwrap_or("3306".to_string())
        .parse()
        .unwrap();

    let user = env::var("ISUCONP_DB_USER").unwrap_or("root".to_string());
    // let password = env::var("ISUCONP_DB_PASSWORD").expect("Failed to ISUCONP_DB_PASSWORD");
    let password = env::var("ISUCONP_DB_PASSWORD").unwrap_or("root".to_string());
    let dbname = env::var("ISUCONP_DB_NAME").unwrap_or("isuconp".to_string());

    let redis_url = env::var("ISUCONP_REDIS_URL").unwrap_or("localhost:6379".to_string());

    let dsn = format!(
        "{}:{}@tcp({}:{})/{}?charset=utf8mb4&parseTime=true&loc=Local",
        &user, &password, &host, &port, &dbname
    );
    let dsn = "mysql://root:root@localhost:3306/isuconp".to_string();

    let num_cpus = num_cpus::get();

    let db = sqlx::mysql::MySqlPoolOptions::new()
        .max_connections(num_cpus as u32)
        .connect_timeout(Duration::from_secs(1))
        .connect(&dsn)
        .await
        .unwrap();

    let private_key = actix_web::cookie::Key::generate();

    HttpServer::new(move || {
        let mut handlebars = Handlebars::new();
        handlebars.register_helper("image_url_helper", Box::new(image_url));
        handlebars.register_helper("date_time_format", Box::new(date_time_format));
        handlebars
            .register_templates_directory(".html", "./static")
            .unwrap();

        App::new()
            .wrap(middleware::Logger::default())
            .wrap(if cfg!(debug_assertions) {
                Cors::permissive()
            } else {
                Cors::default()
                    .supports_credentials()
                    .allowed_origin("http://localhost")
            })
            .wrap(RedisSession::new(redis_url.clone(), private_key.master()))
            .app_data(Data::new(db.clone()))
            .app_data(Data::new(handlebars))
            .service(get_initialize)
            .service(get_login)
            .service(post_login)
            .service(get_register)
            .service(post_register)
            .service(get_logout)
            .service(get_index)
            .service(get_posts)
            .service(get_posts_id)
            .service(get_image)
            // .service(ResourceDef::new("/{tail}*").)
            .service(Files::new("/", "../public"))
            .service(
                web::resource("/test").to(|req: HttpRequest| match *req.method() {
                    Method::GET => HttpResponse::Ok(),
                    Method::POST => HttpResponse::MethodNotAllowed(),
                    _ => HttpResponse::NotFound(),
                }),
            )
    })
    .bind(("0.0.0.0", 8080))?
    .run()
    .await
}
