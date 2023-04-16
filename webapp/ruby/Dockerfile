FROM ruby:3.2-buster

RUN mkdir -p /home/webapp
COPY Gemfile /home/webapp
COPY Gemfile.lock /home/webapp
WORKDIR /home/webapp
RUN bundle config set --local path 'vendor/bundle'
RUN bundle install
COPY . /home/webapp
CMD bundle exec foreman start
