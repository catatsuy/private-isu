FROM rust:1.61.0

WORKDIR /home/webapp

COPY Cargo.toml Cargo.toml
RUN mkdir src \
    && echo "fn main(){}" > src/main.rs \
    && echo "DATABASE_URL=mysql://root:root@mysql:3306/isuconp" > .env \
    && cargo build --release

COPY src src
COPY static static
RUN rm -f target/release/deps/rust*
CMD cargo run --release
