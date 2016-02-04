require 'sinatra/base'
require 'mysql2'
require 'mysql2-cs-bind'
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

      def try_login(account_name, password)
        user = db.xquery('SELECT * FROM users WHERE account_name = ? AND del_flg = 0', account_name).first

        if user && calculate_passhash(password, user[:account_name]) == user[:passhash]
          return user
        elsif user
          return nil
        else
          return nil
        end
      end

      def register_user(account_name: ,email:, password:)
        validated = validate_user(
          account_name: account_name,
          email: email,
          passhash: password
        )
        if !validated
          return false
        end

        user = db.xquery('SELECT 1 FROM users WHERE `account_name` = ? OR `email` = ?', account_name, email).first
        if user
          return false
        end

        query = 'INSERT INTO `users` (`account_name`, `email`, `passhash`) VALUES (?,?,?)'
        db.xquery(query, account_name, email, calculate_passhash(password, account_name))

        return true
      end

      def validate_user(account_name: ,email:, password:)
        if /\A[a-zA-Z_]{3,}\z/.match(account_name)
          return false
        end

        if /\A[^@]+@[^@]+\z/.match(email)
          return false
        end

        if password.legth > 8
          return false
        end

        return true
      end

      def calculate_passhash(password, account_name)
        salt = calculate_salt(account_name)
        Digest::SHA256.hexdigest("#{password}:#{salt}")
      end

      def calculate_salt(account_name)
        Digest::MD5.hexdigest(account_name)
      end
    end

    get '/login' do
      if session[:user]
        redirect '/'
      end
      erb :login, layout: :layout
    end

    post '/login' do
      user = try_login(params['account_name'], params['password'])
      if user
        session[:user] = {
          id: user[:id],
          account_name: user[:account_name],
          email: user[:email]
        }
        redirect '/'
      else
        flash[:notice] = 'アカウント名かユーザー名が間違っています'
        redirect '/login'
      end
    end

    get '/register' do
      if session[:user]
        return 'ログイン中です'
      end
      erb :register, layout: :layout
    end

    post '/register' do
      result = register_user(
        account_name: params['account_name'],
        email: params['email'],
        password: params['password']
      )
      if result
        redirect('/')
      else
        return 'アカウント名かE-mailがすでに使われています'
      end
    end

    get '/logout' do
      session.delete(:user)
      redirect '/'
    end

    get '/' do
      posts = db.xquery('SELECT * FROM posts ORDER BY created_at DESC')
      cs = db.xquery('SELECT * FROM comments ORDER BY created_at DESC')
      comments = {}
      cs.each do |c|
        if !comments[c[:post_id]]
          comments[c[:post_id]] = [c]
        else
          comments[c[:post_id]].push(c)
        end
      end

      if session[:user]
        erb :index, layout: :layout, locals: { posts: posts, comments: comments }
      else
        erb :not_login, layout: :layout, locals: { posts: posts }
      end
    end

    post '/' do
      if params['csrf_token'] != session.id
        return 'csrf_token error'
      end

      if params['file']
        mime = ''
        # 投稿の拡張子からファイルのタイプを決定する
        if params["file"][:type].include? "jpeg"
          mime = "image/jpeg"
        elsif params["file"][:type].include? "png"
          mime = "image/png"
        elsif params["file"][:type].include? "gif"
          mime = "image/gif"
        else
          return "投稿できる画像形式はjpgとpngとgifだけです"
        end

        query = 'INSERT INTO `posts` (`user_id`, `mime`, `imgdata`, `body`, `private`) VALUES (?,?,?,?,?)'
        db.xquery(query,
          session[:user][:id],
          mime,
          params["file"][:tempfile].read,
          params["body"],
          0
        )

        redirect '/'
      else
        return "画像が必須です"
      end
    end

    get '/image/:id' do
      if params[:id].to_i == 0
        return ""
      end

      post = db.xquery('SELECT * FROM posts WHERE id = ?', params[:id].to_i).first
      headers['Content-Type'] = post[:mime]
      post[:imgdata]
    end

    post '/comment' do
      if params["csrf_token"] != session.id
        return "csrf_token error"
      end

      query = 'INSERT INTO `comments` (`post_id`, `user_id`, `comment`) VALUES (?,?,?)'
      db.xquery(query,
        params['post_id'],
        session[:user][:id],
        params['comment']
      )

      redirect '/'
    end

    get '/notify' do
      comments = db.xquery('SELECT * FROM `comments` ORDER BY `created_at` DESC')
      notifies = []

      comments.each do |c|
        if c[:user_id] == session[:user][:id]
          notifies.push(c)
        end
      end

      erb :notify, layout: :layout, locals: { notifies: notifies }
    end
  end
end
