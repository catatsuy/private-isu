# This file should contain all the record creation needed to seed the database with its default values.
# The data can then be loaded with the rake db:seed (or created alongside the db with db:setup).
#
# Examples:
#
#   cities = City.create([{ name: 'Chicago' }, { name: 'Copenhagen' }])
#   Mayor.create(name: 'Emanuel', city: cities.first)

Team.create([
  { name: 'あんこうチーム', users: [User.create(name: 'foo', password: 'hogehoge', email: 'foo@foobar.local')], app_host: 'localhost:8080' },
  { name: 'カメさんチーム', users: [User.create(name: 'bar', password: 'hogehoge', email: 'bar@foobar.local')], app_host: 'localhost:8080' }
])

