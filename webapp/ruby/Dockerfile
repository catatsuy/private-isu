FROM ruby:3.1-buster

RUN mkdir -p /home/webapp
COPY . /home/webapp
WORKDIR /home/webapp
RUN bundle config set --local path 'vendor/bundle'
RUN bundle install
CMD bundle exec foreman start
