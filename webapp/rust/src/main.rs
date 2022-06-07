use std::{env, io};

use actix_cors::Cors;
use actix_web::{
    error,
    http::{Method, StatusCode},
    middleware, web, App, HttpRequest, HttpResponse, HttpServer,
};

#[actix_web::main]
async fn main() -> io::Result<()> {
    let host = env::var("ISUCONP_DB_HOST").unwrap_or("localhost".to_string());
    let port: u32 = env::var("ISUCONP_DB_PORT")
        .unwrap_or("3306".to_string())
        .parse()
        .unwrap_or(3306);

    let user = env::var("ISUCONP_DB_USER").unwrap_or("root".to_string());
    let password = env::var("ISUCONP_DB_PASSWORD").expect("Failed to ISUCONP_DB_PASSWORD");
    let dbname = env::var("ISUCONP_DB_NAME").unwrap_or("isuconp".to_string());

    let dsn = format!(
        "{}:{}@tcp({}:{})/{}?charset=utf8mb4&parseTime=true&loc=Local",
        &user, &password, &host, &port, &dbname
    );

    HttpServer::new(move || {
        App::new()
            .wrap(middleware::Logger::default())
            .wrap(if cfg!(debug_assertions) {
                Cors::permissive()
            } else {
                Cors::default().supports_credentials()
            })
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
