# role :db,  %w{deploy@example.com}
role :web, %w{shanai-isucon-portal shanai-isucon-bench}, user: fetch(:user)
role :app, %w{shanai-isucon-portal shanai-isucon-bench}, user: fetch(:user)

# set :ssh_options, {
#   keys: %w(/home/rlisowski/.ssh/id_rsa),
#   forward_agent: false,
#   auth_methods: %w(password)
# }
