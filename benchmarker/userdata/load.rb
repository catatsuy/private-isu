#!/usr/bin/env ruby

require 'mysql2'
require 'digest'

rand = Random.new(1)

def digest(src)
  Digest::SHA512.hexdigest(src)
end

def calculate_salt(account_name)
  digest account_name
end

def calculate_passhash(password, account_name)
  digest "#{password}:#{calculate_salt(account_name)}"
end


puts "imgディレクトリに画像を展開"
`curl -L -O https://github.com/catatsuy/private-isu/releases/download/img/img.zip`
`unzip img.zip`

kaomoji = File.read('kaomoji.txt').strip().split("\n")

db = Mysql2::Client.new(
  host: 'localhost',
  port: 3306,
  username: 'root',
  password: '',
  database: 'isuconp',
  encoding: 'utf8mb4',
  reconnect: true,
)
db.query_options.merge!(symbolize_keys: true)


puts "schema.sqlを読み込む"
File.read('../sql/schema.sql').split(';').each do |sql|
  puts sql
  db.query(sql) unless sql.strip == ''
end


#CREATE TABLE IF NOT EXISTS users (
  #`id` int NOT NULL AUTO_INCREMENT PRIMARY KEY,
  #`account_name` varchar(64) NOT NULL UNIQUE,
  #`passhash` varchar(128) NOT NULL, -- SHA2 512 non-binary (hex)
  #`authority` tinyint(1) NOT NULL DEFAULT 0,
  #`del_flg` tinyint(1) NOT NULL DEFAULT 0,
  #`created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
#) DEFAULT CHARSET=utf8mb4;

puts "users"

query = db.prepare('INSERT INTO users (`id`,`account_name`,`passhash`,`authority`,`del_flg`,`created_at`) VALUES (?,?,?,?,?,?)')
open('names.txt') do |f|
  f.each_line.with_index(1) do |line,i|
    account_name = line.chomp
    password = account_name * 2

    passhash = calculate_passhash(password, account_name)

    authority = i < 10 ? 1 : 0
    del_flg = ((i >= 10) && (i % 50 == 0)) ? 1 : 0
    created_at = DateTime.parse('2016-01-01 00:00:00') + (1.to_r / 24 / 60 / 60 * i) # 毎秒1アカウント作られたことにする

    query.execute(i, account_name, passhash, authority, del_flg, created_at.to_time)

    p i if i % 20 == 0
  end
end

#CREATE TABLE IF NOT EXISTS posts (
  #`id` int NOT NULL AUTO_INCREMENT PRIMARY KEY,
  #`user_id` int NOT NULL,
  #`mime` varchar(64) NOT NULL,
  #`imgdata` mediumblob NOT NULL,
  #`body` text NOT NULL,
  #`created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
#) DEFAULT CHARSET=utf8mb4;

puts "posts"

query = db.prepare('INSERT INTO posts (`id`,`user_id`,`mime`,`imgdata`,`body`,`created_at`) VALUES (?,?,?,?,?,?)')

Dir.glob("img/*").shuffle(random: rand).each.with_index(1) do |image, i|
  user_id = rand.rand(1..1000)
  mime = image.end_with?('.jpg') ? 'image/jpeg' : image.end_with?('.png') ? 'image/png' : 'image/gif'
  created_at = DateTime.parse('2016-01-02 00:00:00') + (1.to_r / 24 / 60 / 60 * i) # 毎秒1投稿されたことにする
  body = kaomoji[rand.rand(0...kaomoji.length)]

  open(image) do |f|
    query.execute(i, user_id, mime, f.read, body, created_at.to_time)
  end

  p i if i % 100 == 0
end

#CREATE TABLE IF NOT EXISTS comments (
  #`id` int NOT NULL AUTO_INCREMENT PRIMARY KEY,
  #`post_id` int NOT NULL,
  #`user_id` int NOT NULL,
  #`comment` text NOT NULL,
  #`created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
#) DEFAULT CHARSET=utf8mb4;

puts "comments"

query = db.prepare('INSERT INTO comments (`id`,`post_id`,`user_id`,`comment`,`created_at`) VALUES (?,?,?,?,?)')

1.upto(100_000).each do |i|
  post_id = rand.rand(1..10000)
  user_id = rand.rand(1..1000)
  comment = kaomoji[rand.rand(0...kaomoji.length)]
  created_at = DateTime.parse('2016-01-03 00:00:00') + (1.to_r / 24 / 60 / 60 * i) # 毎秒1コメントされたことにする

  query.execute(i, post_id, user_id, comment, created_at.to_time)
end

puts "mysqldumpを出力して圧縮"
`mysqldump -u root -h localhost --hex-blob --add-drop-database --databases isuconp | bzip2 > dump.sql.bz2`
