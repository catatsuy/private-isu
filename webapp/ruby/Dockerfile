FROM ruby:3.0.1-buster

RUN mkdir -p /home/webapp
COPY . /home/webapp
WORKDIR /home/webapp
RUN gem install bundler:2.2.16
RUN bundle config set --local path 'vendor/bundle'
RUN bundle install
CMD bundle exec foreman start
