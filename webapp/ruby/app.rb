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
        user = db.xquery('SELECT * FROM users WHERE account_name = ?', account_name).first

        if user && calculate_passhash(password, user[:account_name]) == user[:passhash]
          return user
        elsif user
          return nil
        else
          return nil
        end
      end

      def register_user(account_name: ,email:, password:)
        user = db.xquery('SELECT 1 FROM users WHERE `account_name` = ? OR `email` = ?', account_name, email).first
        if user
          return false
        end

        query = 'INSERT INTO `users` (`account_name`, `email`, `passhash`) VALUES (?,?,?)'
        db.xquery(query, account_name, email, calculate_passhash(password, account_name))
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
      if session[:user_id]
        pp session
        redirect '/'
      end
      erb :login, layout: :layout
    end

    post '/login' do
      user = try_login(params['account_name'], params['password'])
      pp user
      if user
        session[:user_id] = user['id']
        redirect '/'
      else
        flash[:notice] = "アカウント名かユーザー名が間違っています"
        redirect '/login'
      end
    end

    get '/register' do
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

  end
end
