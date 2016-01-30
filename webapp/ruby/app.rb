require 'sinatra/base'
require 'mysql2'
require 'mysql2-cs-bind'
require 'pp'
require 'digest/md5'

module Isuconp
  class App < Sinatra::Base
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

      def try_login(login, password)
        user = db.xquery('SELECT * FROM users WHERE login = ?', login).first
      end

      def register_user(account_name: ,email:, password:)
        user = db.xquery('SELECT 1 FROM users WHERE `account_name` = ? OR `email` = ?', account_name, email).first
        if user
          return false
        end

        query = 'INSERT INTO `users` (`account_name`, `email`, `passhash`) VALUES (?,?,?)'
        db.xquery(query, account_name, email, calculate_passhash(password, calculate_salt(account_name)))
        return true
      end

      def calculate_passhash(password, salt)
        Digest::SHA256.hexdigest("#{password}:#{salt}")
      end

      def calculate_salt(account_name)
        Digest::MD5.hexdigest(account_name)
      end
    end

    get '/login' do
      register_user(account_name: "catatsuy", email: "catatsuy@catatsuy.org", password: "aa")
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
