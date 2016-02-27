require 'mysql2'
require 'yaml'

@client = Mysql2::Client.new(
  host: ENV['ISUCONP_DB_HOST'] || 'localhost',
  port: ENV['ISUCONP_DB_PORT'] && ENV['ISUCON5_DB_PORT'].to_i,
  username: ENV['ISUCONP_DB_USER'] || 'root',
  password: ENV['ISUCONP_DB_PASSWORD'],
  database: ENV['ISUCONP_DB_NAME'] || 'isuconp',
)

def calculate_salt(account_name)
  Digest::MD5.hexdigest(account_name)
end

def calculate_passhash(password, account_name)
  salt = calculate_salt(account_name)
  Digest::SHA256.hexdigest("#{password}:#{salt}")
end

def register_user(account_name:, password:, authority: 0)
  validated = validate_user(
    account_name: account_name,
    password: password,
  )
  if !validated
    print "validate error"
    return false
  end

  user = @client.prepare('SELECT 1 FROM users WHERE `account_name` = ?').execute(account_name).first
  if user
    return false
  end

  query = 'INSERT INTO `users` (`account_name`, `passhash`, `authority`) VALUES (?,?,?)'
  @client.prepare(query).execute(account_name, calculate_passhash(password, account_name), authority)

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

users = YAML::load(IO::read('users.yml'))

users.each do |u|
  result = register_user(
    account_name: u['account_name'],
    password: u['password'],
    authority: u['authority'],
  )

  if !result
    print "#{u['account_name']}は作成できませんでした\n"
  end
end
