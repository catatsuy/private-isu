#syntax=docker/dockerfile:1
FROM ruby:3.3-slim

RUN \
  --mount=type=cache,target=/var/lib/apt,sharing=locked \
  --mount=type=cache,target=/var/cache/apt,sharing=locked \
  apt-get update -qq && apt-get install -y build-essential default-libmysqlclient-dev

RUN mkdir -p /home/webapp
COPY Gemfile /home/webapp
COPY Gemfile.lock /home/webapp
WORKDIR /home/webapp
RUN bundle config set --local path 'vendor/bundle'
RUN bundle install
COPY . /home/webapp

ENTRYPOINT [ "bundle", "exec", "foreman", "start" ]
