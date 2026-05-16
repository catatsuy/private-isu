#syntax=docker/dockerfile:1
FROM ruby:4.0-slim

RUN \
  --mount=type=cache,target=/var/lib/apt,sharing=locked \
  --mount=type=cache,target=/var/cache/apt,sharing=locked \
  apt-get update -qq \
  && apt-get install -y build-essential default-libmysqlclient-dev git ragel \
  && rm -rf /var/lib/apt/lists/*

WORKDIR /home/webapp
COPY Gemfile Gemfile.lock prepare_unicorn.rb ./
RUN bundle config set --local path 'vendor/bundle' \
  && bundle install \
  && bundle exec ruby prepare_unicorn.rb
COPY . .

ENTRYPOINT ["bundle", "exec", "unicorn", "-c", "unicorn_config.rb"]
