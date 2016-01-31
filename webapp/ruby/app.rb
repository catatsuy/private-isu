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
        pp session[:user]
        redirect '/'
      end
      erb :login, layout: :layout
    end

    post '/login' do
      user = try_login(params['account_name'], params['password'])
      if user
        pp user
        session[:user] = {
          id: user[:id],
          account_name: user[:account_name],
          email: user[:email]
        }
        redirect '/'
      else
        flash[:notice] = "アカウント名かユーザー名が間違っています"
        redirect '/login'
      end
    end

    get '/register' do
      if session[:user]
        return "ログイン中です"
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
        redirect("/")
      else
        return "アカウント名かE-mailがすでに使われています"
      end
    end

    get '/logout' do
      session.delete(:user)
      redirect '/'
    end

    get '/' do
      if session[:user]
        erb :index, layout: :layout
      else
        erb :not_login, layout: :layout
      end
    end

  end
end
