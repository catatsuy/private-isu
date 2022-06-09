use std::{env, io, path::Path, time::Duration};

use actix_cors::Cors;
use actix_redis::RedisSession;
use actix_session::Session;
use actix_web::{
    cookie::time::UtcOffset,
    error, get,
    http::{Method, StatusCode},
    middleware,
    web::{self, Data},
    App, HttpRequest, HttpResponse, HttpServer, Result,
};
use anyhow::Context;
use chrono::{DateTime, FixedOffset, Local, Utc};
use derive_more::Constructor;
use handlebars::{to_json, Handlebars};
use log::LevelFilter;
use serde::{Deserialize, Serialize};
use simplelog::{
    ColorChoice, CombinedLogger, ConfigBuilder, SharedLogger, TermLogger, TerminalMode, WriteLogger,
};
use sqlx::{MySql, Pool};

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
            created_at: Utc::now(),
            ..Default::default()
        }
    }
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

#[get("/initialize")]
async fn get_initialize(pool: Data<Pool<MySql>>) -> Result<HttpResponse> {
    if let Err(e) = db_initialize(&pool).await {
        log::error!("{:?}", &e);
    }
    Ok(HttpResponse::Ok().finish())
}

async fn get_session_user(session: &Session, pool: &Pool<MySql>) -> anyhow::Result<Option<User>> {
    let uid = match session.get::<i32>("user_id") {
        Ok(Some(uid)) => uid,
        Err(e) => anyhow::bail!("Failed to session.get {}", &e),
        _ => return Ok(None),
    };

    let user = sqlx::query_as!(User, "SELECT * FROM `users` WHERE `id` = ?", &uid)
        .fetch_optional(pool)
        .await
        .context("Failed to get_session_user")?;

    Ok(user)
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

async fn post_login() -> Result<HttpResponse> {
    todo!();
    Ok(HttpResponse::Ok().finish())
}

#[get("/register")]
async fn get_register() -> Result<HttpResponse> {
    // TODO:
    // if isLogin(getSessionUser(r)) {
    // 	http.Redirect(w, r, "/", http.StatusFound)
    // 	return
    // }
    todo!()
}

async fn post_register() -> Result<HttpResponse> {
    todo!()
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
