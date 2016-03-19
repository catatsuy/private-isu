# This file should contain all the record creation needed to seed the database with its default values.
# The data can then be loaded with the rake db:seed (or created alongside the db with db:setup).
#
# Examples:
#
#   cities = City.create([{ name: 'Chicago' }, { name: 'Copenhagen' }])
#   Mayor.create(name: 'Emanuel', city: cities.first)

User.create([
    { name: 'foo', password: 'hogehoge', email: 'foo@foobar.local'},
    { name: 'bar', password: 'hogehoge', email: 'bar@foobar.local'}
])
Team.create([
  { name: 'あんこうチーム', users: [User.find(1)], app_host: 'localhost:8080' },
  { name: 'カメさんチーム', users: [User.find(2)], app_host: 'localhost:8080' }
])

