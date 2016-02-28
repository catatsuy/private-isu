require 'sinatra/base'
require 'sinatra/reloader'
require 'mysql2'
require 'rack-flash'
require 'digest/md5'
require 'pp'

module Isuconp
  class App < Sinatra::Base
    use Rack::Session::Cookie, secret: ENV['ISUCONP_SESSION_SECRET'] || 'sendagaya'
    use Rack::Flash
    set :public_folder, File.expand_path('../../public', __FILE__)

    helpers do
      def config
        @config ||= {
          db: {
            host: ENV['ISUCONP_DB_HOST'] || 'localhost',
            port: ENV['ISUCONP_DB_PORT'] && ENV['ISUCON5_DB_PORT'].to_i,
            username: ENV['ISUCONP_DB_USER'] || 'root',
            password: ENV['ISUCONP_DB_PASSWORD'],
            database: ENV['ISUCONP_DB_NAME'] || 'isuconp',
          },
        }
      end

      def db
        return Thread.current[:isuconp_db] if Thread.current[:isuconp_db]
        client = Mysql2::Client.new(
          host: config[:db][:host],
          port: config[:db][:port],
          username: config[:db][:username],
          password: config[:db][:password],
          database: config[:db][:database],
          encoding: 'utf8mb4',
          reconnect: true,
        )
        client.query_options.merge!(symbolize_keys: true)
        Thread.current[:isuconp_db] = client
        client
      end

      def db_initialize
        sql = []
        sql << 'DROP TABLE IF EXISTS users;'
        sql << <<'EOS'
CREATE TABLE IF NOT EXISTS users (
  `id` int NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `account_name` varchar(64) NOT NULL UNIQUE,
  `passhash` varchar(128) NOT NULL, -- SHA2 512 non-binary (hex)
  `authority` tinyint(1) NOT NULL DEFAULT 0,
  `del_flg` tinyint(1) NOT NULL DEFAULT 0,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
) DEFAULT CHARSET=utf8mb4;
EOS
        sql << 'DROP TABLE IF EXISTS posts;'
        sql << <<'EOS'
CREATE TABLE IF NOT EXISTS posts (
  `id` int NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `user_id` int NOT NULL,
  `mime` varchar(64) NOT NULL,
  `imgdata` mediumblob NOT NULL,
  `body` text NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
) DEFAULT CHARSET=utf8mb4;
EOS
        sql << 'DROP TABLE IF EXISTS comments;'
        sql << <<'EOS'
CREATE TABLE IF NOT EXISTS comments (
  `id` int NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `post_id` int NOT NULL,
  `user_id` int NOT NULL,
  `comment` text NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
) DEFAULT CHARSET=utf8mb4;
EOS
        sql.each do |s|
          db.prepare(s).execute
        end
        load "#{File.dirname(__dir__)}/scripts/create_user.rb"
        ""
      end

      def try_login(account_name, password)
        user = db.prepare('SELECT * FROM users WHERE account_name = ? AND del_flg = 0').execute(account_name).first

        if user && calculate_passhash(password, user[:account_name]) == user[:passhash]
          return user
        elsif user
          return nil
        else
          return nil
        end
      end

      def register_user(account_name:, password:)
        validated = validate_user(
          account_name: account_name,
          password: password
        )
        if !validated
          return false
        end

        user = db.prepare('SELECT 1 FROM users WHERE `account_name` = ?').execute(account_name).first
        if user
          return false
        end

        query = 'INSERT INTO `users` (`account_name`, `passhash`) VALUES (?,?)'
        db.prepare(query).execute(
          account_name,
          calculate_passhash(password, account_name)
        )

        return true
      end

      def validate_user(account_name:, password:)
        unless /\A[0-9a-zA-Z_]{3,}\z/.match(account_name)
          return false
        end

        if password.length <= 7
          return false
        end

        return true
      end

      def digest(src)
        `echo '#{src}' | openssl dgst -sha512`.strip
      end

      def calculate_salt(account_name)
        digest account_name
      end

      def calculate_passhash(password, account_name)
        digest "#{password}:#{calculate_salt(account_name)}"
      end
    end

    get '/initialize' do
      db_initialize
    end

    get '/login' do
      if session[:user]
        redirect '/', 302
      end
      erb :login, layout: :layout
    end

    post '/login' do
      if session[:user] && session[:user][:id]
        # ログイン済みはリダイレクト
        redirect '/', 302
      end

      user = try_login(params['account_name'], params['password'])
      if user
        session[:user] = {
          id: user[:id]
        }
        redirect '/', 302
      else
        flash[:notice] = 'アカウント名かパスワードが間違っています'
        redirect '/login', 302
      end
    end

    get '/register' do
      if session[:user]
        redirect '/', 302
      end
      erb :register, layout: :layout
    end

    post '/register' do
      if session[:user] && session[:user][:id]
        # ログイン済みはリダイレクト
        redirect '/', 302
      end

      result = register_user(
        account_name: params['account_name'],
        password: params['password']
      )
      if result
        redirect '/', 302
      else
        flash[:notice] = 'アカウント名がすでに使われています'
        redirect '/register', 302
      end
    end

    get '/logout' do
      session.delete(:user)
      redirect '/', 302
    end

    get '/' do
      posts = db.query('SELECT * FROM posts ORDER BY created_at DESC')
      cs = db.query('SELECT * FROM comments ORDER BY created_at DESC')
      comments = {}
      cs.each do |c|
        if !comments[c[:post_id]]
          comments[c[:post_id]] = [c]
        else
          comments[c[:post_id]].push(c)
        end
      end

      user = {}
      if session[:user]
        user = db.prepare('SELECT * FROM `users` WHERE `id` = ?').execute(
          session[:user][:id]
        ).first
      else
        user = { id: 0 }
      end

      users_raw = db.query('SELECT * FROM `users`')
      users = {}
      users_raw.each do |u|
        users[u[:id]] = u
      end

      erb :index, layout: :layout, locals: { posts: posts, comments: comments, users: users, user: user }
    end

    post '/' do
      unless session[:user] && session[:user][:id]
        # 未ログインはリダイレクト
        redirect '/login', 302
      end

      if params['csrf_token'] != session.id
        return 'csrf_token error'
      end

      if params['file']
        mime = ''
        # 投稿のContent-Typeからファイルのタイプを決定する
        if params["file"][:type].include? "jpeg"
          mime = "image/jpeg"
        elsif params["file"][:type].include? "png"
          mime = "image/png"
        elsif params["file"][:type].include? "gif"
          mime = "image/gif"
        else
          flash[:notice] = '投稿できる画像形式はjpgとpngとgifだけです'
          redirect '/', 302
        end

        query = 'INSERT INTO `posts` (`user_id`, `mime`, `imgdata`, `body`) VALUES (?,?,?,?)'
        db.prepare(query).execute(
          session[:user][:id],
          mime,
          params["file"][:tempfile].read,
          params["body"],
        )

        redirect '/', 302
      else
        flash[:notice] = '画像が必須です'
        redirect '/', 302
      end
    end

    get '/image/:id' do
      if params[:id].to_i == 0
        return ""
      end

      post = db.prepare('SELECT * FROM posts WHERE id = ?').execute(params[:id].to_i).first

      headers['Content-Type'] = post[:mime]
      post[:imgdata]
    end

    post '/comment' do
      unless session[:user] && session[:user][:id]
        # 未ログインはリダイレクト
        redirect '/login', 302
      end

      if params["csrf_token"] != session.id
        return "csrf_token error"
      end

      query = 'INSERT INTO `comments` (`post_id`, `user_id`, `comment`) VALUES (?,?,?)'
      db.prepare(query).execute(
        params['post_id'],
        session[:user][:id],
        params['comment']
      )

      redirect '/', 302
    end

    get '/admin/banned' do
      if !session[:user]
        redirect '/login', 302
      end

      user = db.prepare('SELECT * FROM `users` WHERE `id` = ?').execute(
        session[:user][:id]
      ).first

      if user[:authority] == 0
        return 403
      end

      users = db.query('SELECT * FROM `users` WHERE `authority` = 0 AND `del_flg` = 0 ORDER BY `created_at` DESC')

      erb :banned, layout: :layout, locals: { users: users }
    end

    post '/admin/banned' do
      unless session[:user] && session[:user][:id]
        # 未ログインはリダイレクト
        redirect '/', 302
      end

      user = db.prepare('SELECT * FROM `users` WHERE `id` = ?').execute(
        session[:user][:id]
      ).first

      if user[:authority] == 0
        return 403
      end

      if params['csrf_token'] != session.id
        return 403
      end

      query = 'UPDATE `users` SET `del_flg` = ? WHERE `id` = ?'

      params['uid'].each do |id|
        db.prepare(query).execute(1, id.to_i)
      end

      redirect '/admin/banned', 302
    end

    get '/mypage' do
      unless session[:user] && session[:user][:id]
        # 未ログインはリダイレクト
        redirect '/', 302
      end

      posts_all = db.query('SELECT * FROM `posts` ORDER BY `created_at` DESC')
      comments_all = db.query('SELECT * FROM `comments` ORDER BY `created_at` DESC')
      posts = []
      comments = []

      posts_all.each do |p|
        if p[:user_id] == session[:user][:id]
          posts.push(p)
        end
      end

      comments_all.each do |c|
        if c[:user_id] == session[:user][:id]
          comments.push(c)
        end
      end

      mixed = []
      index = 0

      (0..(posts.length - 1)).each do |pi|
        if comments.length > 0 && comments[index][:created_at] > posts[pi][:created_at]
          mixed.push({type: :comment, value: comments[index]})
        else
          mixed.push({type: :post, value: posts[pi]})
        end

        if index < comments.length - 1
          index += 1
        else
          (pi..(posts.length - 1)).each do |i|
            mixed.push({type: :post, value: posts[i]})
          end
        end
      end

      user = db.prepare('SELECT * FROM `users` WHERE `id` = ?').execute(
        session[:user][:id]
      ).first

      erb :mypage, layout: :layout, locals: { mixed: mixed, user: user }
    end

  end
end
