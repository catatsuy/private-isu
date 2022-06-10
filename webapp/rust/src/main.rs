use std::{any, env, io, path::Path, time::Duration};

use actix_cors::Cors;
use actix_files::Files;
use actix_redis::RedisSession;
use actix_session::Session;
use actix_web::{
    cookie::time::UtcOffset,
    dev::ResourceDef,
    error, get,
    http::{header, Method, StatusCode},
    middleware, post,
    web::{self, Data, Form},
    App, HttpRequest, HttpResponse, HttpServer, Result,
};
use anyhow::{bail, Context};
use chrono::{DateTime, FixedOffset, Local, Utc};
use derive_more::Constructor;
use duct::cmd;
use handlebars::{to_json, Handlebars};
use log::LevelFilter;
use once_cell::sync::Lazy;
use rand::{
    prelude::{SliceRandom, StdRng},
    thread_rng, SeedableRng,
};
use regex::Regex;
use serde::{Deserialize, Serialize};
use simplelog::{
    ColorChoice, CombinedLogger, ConfigBuilder, SharedLogger, TermLogger, TerminalMode, WriteLogger,
};
use sqlx::{MySql, Pool};

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

fn is_login(u: Option<&User>) -> bool {
    match u {
        Some(u) => u.id != 0,
        None => false,
    }
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
async fn get_login(handlebars: Data<Handlebars<'_>>) -> Result<HttpResponse> {
    let body = {
        let user = User::new(0, "test".to_string(), "pass".to_string(), 0, 0, Utc::now());

        let mut json = serde_json::to_value(user).unwrap();
        let map = json.as_object_mut().unwrap();
        map.insert("flash".to_string(), to_json("notice"));
        map.insert("parent".to_string(), to_json("layout"));
        log::debug!("{:?}", &map);

        handlebars.render("login", map).unwrap()
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

        let mut json = serde_json::to_value(user).unwrap();
        let map = json.as_object_mut().unwrap();
        map.insert("flash".to_string(), to_json(get_flash(&session, "notice")));
        map.insert("parent".to_string(), to_json("layout"));
        log::debug!("map {:?}", &map);

        handlebars.render("register", map).unwrap()
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

async fn get_logout() -> Result<HttpResponse> {
    todo!()
}

async fn get_index() -> Result<HttpResponse> {
    todo!()
}

async fn get_posts() -> Result<HttpResponse> {
    todo!()
}

async fn get_posts_id() -> Result<HttpResponse> {
    todo!()
}

async fn post_index() -> Result<HttpResponse> {
    todo!()
}

async fn get_image() -> Result<HttpResponse> {
    todo!()
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
        handlebars
            .register_templates_directory(".html", "./static")
            .unwrap();

        App::new()
            .wrap(middleware::Logger::default())
            .wrap(if cfg!(debug_assertions) {
                Cors::permissive()
            } else {
                Cors::default().supports_credentials()
            })
            .wrap(RedisSession::new(redis_url.clone(), private_key.master()))
            .app_data(Data::new(db.clone()))
            .app_data(Data::new(handlebars))
            .service(get_initialize)
            .service(get_login)
            .service(post_login)
            .service(get_register)
            .service(post_register)
            // .service(ResourceDef::new("/{tail}*").)
            .service(Files::new("/", "../public"))
            .service(
                web::resource("/test").to(|req: HttpRequest| match *req.method() {
                    Method::GET => HttpResponse::Ok(),
                    Method::POST => HttpResponse::MethodNotAllowed(),
                    _ => HttpResponse::NotFound(),
                }),
            )
            .service(web::resource("/").to(|| async {
                error::InternalError::new(
                    io::Error::new(io::ErrorKind::Other, "test"),
                    StatusCode::INTERNAL_SERVER_ERROR,
                )
            }))
    })
    .bind(("0.0.0.0", 8080))?
    .run()
    .await
}

static LAYOUT: &str = include_str!("../templates/layout.html");
static LOGIN: &str = include_str!("../templates/login.html");
