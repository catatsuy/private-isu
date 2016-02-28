#!/usr/bin/env ruby

require 'mysql2'

# imgディレクトリに画像を展開
`curl -O https://github.com/catatsuy/private-isu/releases/download/img/img.zip`
`unzip img.zip`

kaomoji = open('ime_std.txt'){|f| f.each_line.drop(4).map{|line| line.split("\t")[1]}}

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

#CREATE TABLE IF NOT EXISTS users (
  #`id` int NOT NULL AUTO_INCREMENT PRIMARY KEY,
  #`account_name` varchar(64) NOT NULL UNIQUE,
  #`passhash` varchar(128) NOT NULL, -- SHA2 512 non-binary (hex)
  #`authority` tinyint(1) NOT NULL DEFAULT 0,
  #`del_flg` tinyint(1) NOT NULL DEFAULT 0,
  #`created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
#) DEFAULT CHARSET=utf8mb4;

puts "users"

query = db.prepare('INSERT INTO users (`id`,`account_name`,`passhash`,`authority`,`created_at`) VALUES (?,?,?,?,?)')
open('names.txt') do |f|
  f.each_line.with_index(1) do |line,i|
    account_name = line.chomp
    password = account_name * 2

    salt = Digest::MD5.hexdigest(account_name)
    passhash = Digest::SHA256.hexdigest("#{password}:#{salt}")

    authority = i == 1 ? 1 : 0
    created_at = DateTime.parse('2016-01-01 00:00:00') + (1.to_r / 24 / 60 / 60 * i) # 毎秒1アカウント作られたことにする

    query.execute(i, account_name, passhash, authority, created_at.to_time)
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

Dir.glob("img/*").shuffle(random: Random.new(1)).each.with_index(1) do |image, i|
  user_id = (36011 * i) % 1000 + 1 # 36011 は適当に大きな素数
  mime = image.end_with?('.jpg') ? 'image/jpeg' : image.end_with?('.png') ? 'image/png' : 'image/gif'
  created_at = DateTime.parse('2016-01-02 00:00:00') + (1.to_r / 24 / 60 / 60 * i) # 毎秒1投稿されたことにする
  bodies = ['様子','様子です','今日の様子です','ラーメン','うまい','飯テロ']
  body = kaomoji[(26183 * i) % kaomoji.length] # 26183 は適当に大きな素数

  open(image) do |f|
    query.execute(i, user_id, mime, f.read, bodies[i % bodies.length], created_at.to_time)
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
  post_id = (71237 * i) % 10000 + 1 # 71237 は適当に大きな素数
  user_id = (22229 * i) % 1000 + 1 # 22229 は適当に大きな素数
  comment = kaomoji[(9323 * i) % kaomoji.length] # 9323 は適当に大きな素数
  created_at = DateTime.parse('2016-01-03 00:00:00') + (1.to_r / 24 / 60 / 60 * i) # 毎秒1コメントされたことにする

  query.execute(i, post_id, user_id, comment, created_at.to_time)
end

# mysqldumpを出力して圧縮
`mysqldump -u root -h localhost --hex-blob --no-create-info isuconp > dump.sql`
`cat dump.sql | bzip2 > dump.sql.bz2`
