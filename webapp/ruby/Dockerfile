FROM ruby:3.3-slim

RUN apt-get update -qq && apt-get install -y build-essential

RUN apt-get install -y default-libmysqlclient-dev

RUN mkdir -p /home/webapp
COPY Gemfile /home/webapp
COPY Gemfile.lock /home/webapp
WORKDIR /home/webapp
RUN bundle config set --local path 'vendor/bundle'
RUN bundle install
COPY . /home/webapp
CMD bundle exec foreman start
