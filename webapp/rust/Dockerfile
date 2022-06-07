FROM rust:1.61.0

RUN mkdir -p /home/webapp
COPY . /home/webapp
WORKDIR /home/webapp
RUN cargo build --release
CMD ./target/release/rust
