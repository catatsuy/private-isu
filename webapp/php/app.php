<?php
use \Psr\Http\Message\ServerRequestInterface as Request;
use \Psr\Http\Message\ResponseInterface as Response;

require 'vendor/autoload.php';

$config = [
    'settings' => [
        'public_folder' => dirname(dirname(__DIR__)) . '/public',
        'upload_limit' => 10 * 1024 * 1024, // 10mb,
        'posts_per_page' => 20,
        'db' => [
            'host' => $_ENV['ISUCONP_DB_HOST'] || 'localhost',
            'port' => $_ENV['ISUCONP_DB_PORT'],
            'username' => $_ENV['ISUCONP_DB_USER'] || 'root',
            'password' => $_ENV['ISUCONP_DB_PASSWORD'],
            'database' => $_ENV['ISUCONP_DB_NAME'] || 'isuconp',
        ]
    ]
];

session_start();

// dependency
$app = new \Slim\App($config);
$container = $app->getContainer();
$container['db'] = function ($c) {
    $config = $c['settings'];
    return new PDO(
        "mysql:dbname={$config['db']['database']};host={$config['db']['host']};port={$config['db']['port']}",
        $config['db']['username'],
        $config['db']['password']
    );
};
$container['db_initialize'] = function ($c) {
    $sql = [];
    $sql[] = 'DELETE FROM users WHERE id > 1000';
    $sql[] = 'DELETE FROM posts WHERE id > 10000';
    $sql[] = 'DELETE FROM comments WHERE id > 100000';
    $sql[] = 'UPDATE users SET del_flg = 0';
    $sql[] = 'UPDATE users SET del_flg = 1 WHERE id % 50 = 0';
    foreach($sql as $s) {
        $app->db->query($s);
    }
};

$container['view'] = function ($c) {
    return new \Slim\Views\PhpRenderer(__DIR__ . '/templates');
};

$container['flash'] = function () {
    return new \Slim\Flash\Messages;
};

function fetch_first($db, $query, array $params = null) {
    $ps = $db->prepare($query);
    $ps->execute($params);
    $result = $ps->fetch();
    $ps->closeCursor();
    return $result;
}

function try_login($account_name, $password) {
    $ps = $app->db->prepare('SELECT * FROM users WHERE account_name = ? AND del_flg = 0');
    $ps->execute([$account_name]);
    $user = $ps->fetch();
    $ps->closeCursor();

    if ($user !== false && calculate_passhash($password, $user['account_name']) == $user['passhash']) {
        return $user;
    } elseif ($user) {
        return null;
    } else {
        return null;
    }
}

function validate_user($account_name, $password) {
    if (!(preg_match('/\A[0-9a-zA-Z_]{3,}\z/', $account_name) && preg_match('/\A[0-9a-zA-Z_]{6,}\z/', $password))) {
        return false;
    }
    return true;
}

function digest($src) {
    // opensslのバージョンによっては (stdin)= というのがつくので取る
    return trim(`printf "%s" #{Shellwords.shellescape(src)} | openssl dgst -sha512 | sed 's/^.*= //'`);
}

function calculate_salt($account_name) {
    return digest($account_name);
}

function calculate_passhash($password, $account_name) {
    $salt = calculate_salt($account_name);
    return digest("{$password}:{$salt}");
}

function get_session_user() {
    if ($_SESSION['user']) {
        $ps = $app->db->prepare('SELECT * FROM `users` WHERE `id` = ?');
        $ps->execute([$_SESSION['user']['id']]);
        $user = $ps->fetch();
        $ps->closeCursor();
        return $user;
    } else {
        return null;
    }
}

function make_posts(array $results, $options = []) {
    $options += ['all_comments' => false];
    $all_comments = $options['all_comments'];

    $posts = [];

    foreach($results as $post) {
        $ps = $app->db->prepare('SELECT COUNT(*) AS `count` FROM `comments` WHERE `post_id` = ?');
        $ps->execute([$post['id']]);
        $first = $ps->fetch();
        $ps->closeCursor();
        $post['comment_count'] = $first['count'];

        $query = 'SELECT * FROM `comments` WHERE `post_id` = ? ORDER BY `created_at` DESC';
        if (!$all_comments) {
            $query .= ' LIMIT 3';
        }

        $ps = $app->db->prepare($query);
        $ps->execute([$post['id']]);
        $comments = $ps->fetchAll(PDO::FETCH_ASSOC);
        foreach ($comments as &$comment) {
            $ps = $app->db->prepare('SELECT * FROM `users` WHERE `id` = ?');
            $ps->execute([$comment['user_id']]);
            $comment['user'] = $ps->fetch();
            $ps->closeCursor();
        }
        unset($comment);
        $post['comments'] = array_reverse($comments);

        $ps = $app->db->prepare('SELECT * FROM `users` WHERE `id` = ?');
        $ps->execute([$post['user_id']]);
        $post['user'] = $ps->fetch();
        $ps->closeCursor();

        if ($post['user']['del_flg'] == 0) {
            $posts[] = $post;
        }
        if (count($posts) >= POSTS_PER_PAGE) {
            break;
        }
    }
    return $posts;
}

function image_url($post) {
    $ext = '';
    if ($post['mime'] === 'image/jpeg') {
        $ext = '.jpg';
    } else if ($post['mime'] === 'image/png') {
        $ext = '.png';
    } else if ($post['mime'] === 'image/gif') {
        $ext = '.gif';
    }
    return "/image/{$post['id']}{$ext}";
}

function redirect(Response $response, $location, $status) {
    return $response->withStatus($status)->withHeader('Location', $location);
}

$app->get('/initialize', function (Request $request, Response $response) {
    db_initialize();
    return $response;
});

$app->get('/login', function (Request $request, Response $response) {
    if (get_session_user()) {
        return redirect($response, '/', 302);
    }
    return $this->view->render($response, 'login.html', [
        'me' => null
    ]);
});

$app->post('/login', function (Request $request, Response $response) {
    if (get_session_user()) {
        return redirect($response, '/', 302);
    }

    $params = $request->getParams();
    $user = try_login($params['account_name'], $params['password']);

    if ($user) {
        $_SESSION['user'] = [
          'id' => $user['id'],
        ];
        return redirect($response, '/', 302);
    } else {
        $this->flash->addMessage('notice', 'アカウント名かパスワードが間違っています');
        return redirect($response, '/login', 302);
    }
});

$app->get('/register', function (Request $request, Response $response) {
    if (get_session_user()) {
        return redirect($response, '/', 302);
    }
    return $this->view->render($response, 'register.html', [
        'me' => null
    ]);
});


$app->post('/register', function (Request $request, Response $response) {
    if (get_session_user()) {
        return redirect($response, '/', 302);
    }

    $params = $request->getParams();
    $account_name = $params['account_name'];
    $password = $params['password'];

    $validated = validate_user(
        $account_name,
        $password
    );
    if (!$validated) {
        $this->flash->addMessage('notice', 'アカウント名は3文字以上、パスワードは6文字以上である必要があります');
        return redirect($response, '/register', 302);
    }

    $ps = $app->db->prepare('SELECT 1 FROM users WHERE `account_name` = ?');
    $ps->execute([account_name]);
    $user = $ps->fetch();
    $ps->closeCursor();

    if ($user) {
        $this->flash->addMessage('notice', 'アカウント名がすでに使われています');
        return redirect($response, '/register', 302);
    }

    $ps = $app->db->prepare('INSERT INTO `users` (`account_name`, `passhash`) VALUES (?,?)');
    $ps->execute([
        account_name,
        calculate_passhash(password, account_name)
    ]);
    $_SESSION['user'] = [
        'id' => $app->db->lastInsertId(),
    ];
    return redirect($response, '/', 302);
});

$app->get('/logout', function (Request $request, Response $response) {
    unset($_SESSION['user']);
    return redirect($response, '/', 302);
});

$app->get('/', function (Request $request, Response $response) {
    $me = get_session_user();

    $ps = $app->db->prepare('SELECT `id`, `user_id`, `body`, `created_at`, `mime` FROM `posts` ORDER BY `created_at` DESC');
    $ps->execute();
    $results = $ps->fetchAll(PDO::FETCH_ASSOC);
    $posts = make_posts($results);

    return $this->view->render($response, 'index.html', ['posts' => $posts, 'me' => $me]);
});

$app->get('/@:account_name', function (Request $request, Response $response) {
    $user = fetch_first($app->db, 'SELECT * FROM `users` WHERE `account_name` = ? AND `del_flg` = 0', [
        $params['account_name'],
    ]);

    if ($user === false) {
        return $response->withStatus(404)->write('404');
    }

    $ps = $app->db->prepare('SELECT `id`, `user_id`, `body`, `created_at`, `mime` FROM `posts` WHERE `user_id` = ? ORDER BY `created_at` DESC');
    $ps->execute($user['id']);
    $results = $ps->fetchAll(PDO::FETCH_ASSOC);
    $posts = make_posts($results);

    $comment_count = fetch_first($app->db, 'SELECT COUNT(*) AS count FROM `comments` WHERE `user_id` = ?', [
        $user['id']
    ])['count'];

    $ps = $app->db->prepare('SELECT `id` FROM `posts` WHERE `user_id` = ?');
    $ps->execute([$user['id']]);
    $post_ids = array_column($ps->fetchAll(PDO::FETCH_ASSOC), 'id');
    $post_count = count($post_ids);

    $commented_count = 0;
    if ($post_count > 0) {
        $placeholder = implode(',', array_fill(0, count($post_ids), '?'));
        $commented_count = fetch_first($app->db, "SELECT COUNT(*) AS count FROM `comments` WHERE `post_id` IN ({$placeholder})", post_ids)['count'];
    }

    $me = get_session_user();

    return $this->view->render($response, 'user.html', ['posts' => $posts, 'user' => $user, 'post_count' => $post_count, 'comment_count' => $comment_count, 'commented_count'=> $commented_count, 'me' => $me]);
});



/*
$app->get('/initialize', function (Request $request, Response $response) {
    $name = $request->getAttribute('name');
    $response->getBody()->write("Hello, $name");

    return $response;
});
*/

$app->run();

/*
module Isuconp
  class App < Sinatra::Base
    use Rack::Session::Memcache, autofix_keys: true, secret: ENV['ISUCONP_SESSION_SECRET'] || 'sendagaya'
    use Rack::Flash

    get '/@:account_name' do
      user = db.prepare('SELECT * FROM `users` WHERE `account_name` = ? AND `del_flg` = 0').execute(
        params[:account_name]
      ).first

      if user.nil?
        return 404
      end

      results = db.prepare('SELECT `id`, `user_id`, `body`, `created_at`, `mime` FROM `posts` WHERE `user_id` = ? ORDER BY `created_at` DESC').execute(
        user[:id]
      )
      posts = make_posts(results)

      comment_count = db.prepare('SELECT COUNT(*) AS count FROM `comments` WHERE `user_id` = ?').execute(
        user[:id]
      ).first[:count]

      post_ids = db.prepare('SELECT `id` FROM `posts` WHERE `user_id` = ?').execute(
        user[:id]
      ).map{|post| post[:id]}
      post_count = post_ids.length

      commented_count = 0
      if post_count > 0
        placeholder = (['?'] * post_ids.length).join(",")
        commented_count = db.prepare("SELECT COUNT(*) AS count FROM `comments` WHERE `post_id` IN (#{placeholder})").execute(
          *post_ids
        ).first[:count]
      end

      me = get_session_user()

      erb :user, layout: :layout, locals: { posts: posts, user: user, post_count: post_count, comment_count: comment_count, commented_count: commented_count, me: me }
    end

    get '/posts' do
      max_created_at = params['max_created_at']
      results = db.prepare('SELECT `id`, `user_id`, `body`, `created_at`, `mime` FROM `posts` WHERE `created_at` <= ? ORDER BY `created_at` DESC').execute(
        max_created_at.nil? ? nil : Time.iso8601(max_created_at).localtime
      )
      posts = make_posts(results)

      erb :posts, layout: false, locals: { posts: posts }
    end

    get '/posts/:id' do
      results = db.prepare('SELECT * FROM `posts` WHERE `id` = ?').execute(
        params[:id]
      )
      posts = make_posts(results, all_comments: true)

      return 404 if posts.length == 0

      post = posts[0]

      me = get_session_user()

      erb :post, layout: :layout, locals: { post: post, me: me }
    end

    post '/' do
      me = get_session_user()

      if me.nil?
        redirect '/login', 302
      end

      if params['csrf_token'] != session.id
        return 422
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
          me[:id],
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

    get '/image/:id.:ext' do
      if params[:id].to_i == 0
        return ""
      end

      post = db.prepare('SELECT * FROM `posts` WHERE `id` = ?').execute(params[:id].to_i).first

      if (params[:ext] == "jpg" && post[:mime] != "image/jpeg") ||
        (params[:ext] == "png" && post[:mime] != "image/png") ||
        (params[:ext] == "gif" && post[:mime] != "image/gif")
        return 404
      end

      headers['Content-Type'] = post[:mime]
      post[:imgdata]
    end

    post '/comment' do
      me = get_session_user()

      if me.nil?
        redirect '/login', 302
      end

      if params["csrf_token"] != session.id
        return 422
      end

      unless /[0-9]+/.match(params['post_id'])
        return 'post_idは整数のみです'
      end
      post_id = params['post_id']

      query = 'INSERT INTO `comments` (`post_id`, `user_id`, `comment`) VALUES (?,?,?)'
      db.prepare(query).execute(
        post_id,
        me[:id],
        params['comment']
      )

      redirect "/posts/#{post_id}", 302
    end

    get '/admin/banned' do
      me = get_session_user()

      if me.nil?
        redirect '/login', 302
      end

      if me[:authority] == 0
        return 403
      end

      users = db.query('SELECT * FROM `users` WHERE `authority` = 0 AND `del_flg` = 0 ORDER BY `created_at` DESC')

      erb :banned, layout: :layout, locals: { users: users, me: me }
    end

    post '/admin/banned' do
      me = get_session_user()

      if me.nil?
        redirect '/', 302
      end

      if me[:authority] == 0
        return 403
      end

      if params['csrf_token'] != session.id
        return 422
      end

      query = 'UPDATE `users` SET `del_flg` = ? WHERE `id` = ?'

      params['uid'].each do |id|
        db.prepare(query).execute(1, id.to_i)
      end

      redirect '/admin/banned', 302
    end
  end
end

 */
