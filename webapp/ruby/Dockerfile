#syntax=docker/dockerfile:1
FROM ruby:4.0-slim

RUN \
  --mount=type=cache,target=/var/lib/apt,sharing=locked \
  --mount=type=cache,target=/var/cache/apt,sharing=locked \
  apt-get update -qq \
  && apt-get install -y build-essential default-libmysqlclient-dev git ragel \
  && rm -rf /var/lib/apt/lists/*

WORKDIR /home/webapp
COPY Gemfile Gemfile.lock ./
RUN bundle config set --local path 'vendor/bundle' \
  && bundle install \
  && ruby -e "require 'fileutils'; \
    gs = Dir['vendor/bundle/**/unicorn.gemspec']; \
    abort('unicorn.gemspec not found') if gs.empty?; \
    gs.each do |g| \
      spec = Gem::Specification.load(g); \
      dir = File.dirname(g); \
      ver_path = File.join(dir, 'lib', 'unicorn', 'version.rb'); \
      FileUtils.mkdir_p(File.dirname(ver_path)); \
      File.write(ver_path, \"# frozen_string_literal: true\\nmodule Unicorn\\n  module Const\\n    UNICORN_VERSION = '#{spec.version}'\\n  end\\n  VERSION = Const::UNICORN_VERSION\\nend\\n\"); \
      ext_dir = File.join(dir, 'ext', 'unicorn_http'); \
      Dir.chdir(ext_dir) do \
        system('rake', 'ragel') || system('ragel', '-G2', 'unicorn_http.rl', '-o', 'unicorn_http.c') or abort('ragel step failed'); \
        system(Gem.ruby, 'extconf.rb') or abort('extconf.rb failed'); \
        system('make', 'clean'); \
        system('make') or abort('make failed'); \
        system('make', 'install') or abort('make install failed'); \
      end; \
      so = Dir[File.join(ext_dir, 'unicorn_http.so'), File.join(dir, 'lib', 'unicorn_http.so')].first; \
      FileUtils.cp(so, File.join(dir, 'lib', 'unicorn_http.so')) if so; \
    end"
COPY . .

ENTRYPOINT ["bundle", "exec", "unicorn", "-c", "unicorn_config.rb"]
