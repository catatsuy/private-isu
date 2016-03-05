require 'sinatra/base'
require 'mysql2'
require 'rack-flash'

module Isuconp
  class App < Sinatra::Base
    use Rack::Session::Memcache, autofix_keys: true, secret: ENV['ISUCONP_SESSION_SECRET'] || 'sendagaya'
    use Rack::Flash
    set :public_folder, File.expand_path('../../public', __FILE__)

    UPLOAD_LIMIT = 10 * 1024 * 1024 # 10mb

    helpers do
      def config
        @config ||= {
          db: {
            host: ENV['ISUCONP_DB_HOST'] || 'localhost',
            port: ENV['ISUCONP_DB_PORT'] && ENV['ISUCONP_DB_PORT'].to_i,
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
        sql << 'DELETE FROM users WHERE id > 1000'
        sql << 'DELETE FROM posts WHERE id > 10000'
        sql << 'DELETE FROM comments WHERE id > 100000'
        sql.each do |s|
          db.prepare(s).execute
        end
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

        return true
      end

      def digest(src)
        `echo -n #{src} | openssl dgst -sha512 | sed 's/^.*= //'`.strip # opensslのバージョンによっては (stdin)= というのがつくので取る
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
      return 200
    end

    get '/login' do
      if session[:user] && session[:user][:id]
        # ログイン済みはリダイレクト
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
      ps = db.query('SELECT * FROM posts ORDER BY created_at DESC')
      cs = db.query('SELECT * FROM comments ORDER BY created_at ASC')
      cc = db.query('SELECT post_id, COUNT(*) as count FROM comments GROUP BY post_id ORDER BY created_at DESC')
      posts = []
      count = {}
      comments = {}
      cc.each do |c|
        count[c[:post_id]] = c[:count]
      end

      cs.each do |c|
        unless comments[c[:post_id]]
          comments[c[:post_id]] = []
        end
        comments[c[:post_id]].push(c)
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

      ps.each do |p|
        posts << p if users[p[:user_id]][:del_flg] == 0
      end

      erb :index, layout: :layout, locals: { posts: posts, count: count, comments: comments, users: users, user: user }
    end

    get '/posts' do
      max = params['max_created_at']
      posts = []
      count = {}
      comments = {}
      users = {}

      ps = if max
        db.prepare('SELECT * FROM posts WHERE created_at <= ? ORDER BY created_at DESC').execute(Time.parse(max))
      else
        db.query('SELECT * FROM posts ORDER BY created_at DESC')
      end
      cs = db.query('SELECT * FROM comments ORDER BY created_at DESC')
      cc = db.query('SELECT post_id, COUNT(*) as count FROM comments GROUP BY post_id ORDER BY created_at DESC')

      cc.each do |c|
        count[c[:post_id]] = c[:count]
      end

      cs.each do |c|
        unless comments[c[:post_id]]
          comments[c[:post_id]] = []
        end
        comments[c[:post_id]].push(c)
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
      users_raw.each do |u|
        users[u[:id]] = u
      end
      ps.each do |p|
        if users[p[:user_id]][:del_flg] == 0
          p[:imgdata] = "#{request.base_url}/image/#{p[:id]}"
          posts << p
        end
      end
      erb :posts, layout: :layout, locals: { posts: posts, count: count, comments: comments, users: users, user: user }
    end

    get '/posts/:id' do
      post = db.prepare('SELECT * FROM posts WHERE id = ? ORDER BY created_at DESC').execute(
        params[:id]
      ).first

      rs = db.prepare('SELECT * FROM comments WHERE post_id = ? ORDER BY created_at DESC').execute(
        params[:id]
      )
      comments = []
      rs.each do |p|
        comments << p
      end
      count = db.prepare('SELECT COUNT(*) FROM comments WHERE post_id = ? ORDER BY created_at DESC').execute(
        params[:id]
      ).first

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

      erb :posts_id, layout: :layout, locals: { post: post, count: count, comments: comments, users: users, user: user }
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

        if params['file'][:tempfile].read.length > UPLOAD_LIMIT
          flash[:notice] = 'ファイルサイズが大きすぎます'
          redirect '/', 302
        end

        params['file'][:tempfile].rewind
        query = 'INSERT INTO `posts` (`user_id`, `mime`, `imgdata`, `body`) VALUES (?,?,?,?)'
        db.prepare(query).execute(
          session[:user][:id],
          mime,
          params["file"][:tempfile].read,
          params["body"],
        )
        pid = db.last_id

        redirect "/posts/#{pid}", 302
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

      mixed = []
      posts_all = db.query('SELECT * FROM `posts` ORDER BY `created_at` DESC')
      posts_all.each do |p|
        mixed << {type: :post, value: p}
      end
      comments_all = db.query('SELECT * FROM `comments` ORDER BY `created_at` DESC')
      comments_all.each do |c|
        mixed << {type: :comment, value: c}
      end

      mixed = mixed.map do |m|
        if m[:type] == :post
          posts_comments = []
          rs = db.prepare('SELECT * FROM `comments` WHERE `post_id` = ? ORDER BY `created_at` DESC').execute(
            m[:value][:id]
          )
          rs.each do |pc|
            posts_comments << pc
          end
          m.merge!({comments: posts_comments})
        end
        m
      end
      mixed = mixed.select { |m| m[:value][:user_id] == session[:user][:id] }
      mixed.sort! { |a, b| a[:value][:created_at] <=> b[:value][:created_at] }

      user = db.prepare('SELECT * FROM `users` WHERE `id` = ?').execute(
        session[:user][:id]
      ).first

      users_raw = db.query('SELECT * FROM `users`')
      users = {}
      users_raw.each do |u|
        users[u[:id]] = u
      end

      erb :mypage, layout: :layout, locals: { mixed: mixed, user: user, users: users }
    end

  end
end
