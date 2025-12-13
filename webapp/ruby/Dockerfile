#syntax=docker/dockerfile:1
FROM ruby:3.4-slim

RUN \
  --mount=type=cache,target=/var/lib/apt,sharing=locked \
  --mount=type=cache,target=/var/cache/apt,sharing=locked \
  apt-get update -qq \
  && apt-get install -y build-essential default-libmysqlclient-dev \
  && rm -rf /var/lib/apt/lists/*

WORKDIR /home/webapp
COPY Gemfile Gemfile.lock ./
RUN bundle config set --local path 'vendor/bundle' && bundle install
COPY . .

ENTRYPOINT ["bundle", "exec", "unicorn", "-c", "unicorn_config.rb"]
