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

$app->get('/posts', function (Request $request, Response $response) {
    $max_created_at = $params['max_created_at'];
    $ps = $app->db->prepare('SELECT `id`, `user_id`, `body`, `created_at`, `mime` FROM `posts` WHERE `created_at` <= ? ORDER BY `created_at` DESC');
    $ps->execute([$max_created_at === null ? null : date(DATE_ATOM, $max_created_at)]);
    $results = $ps->fetchAll(PDO::FETCH_ASSOC);
    $posts = make_posts($results);

    return $this->view->render($response, 'posts.html', ['posts' => $posts]);
});

$app->get('/posts/:id', function (Request $request, Response $response) {
    $ps = $app->db->prepare('SELECT * FROM `posts` WHERE `id` = ?');
    $ps->execute([$params['id']]);
    $results = $ps->fetchAll(PDO::FETCH_ASSOC);
    $posts = make_posts($results, ['all_comments' => true]);

    if (count(posts) == 0) {
        return $response->withStatus(404)->write('404');
    }

    $post = $posts[0];

    $me = get_session_user();

    return $this->view->render($response, 'post.html', ['post' => $post, 'me' => $me]);
});

$app->post '/', function (Request $request, Response $response) {
    $me = get_session_user();

    if ($me === null) {
        return redirect($response, '/login', 302);
    }

    if ($params['csrf_token'] != session_id()) {
        return $response->withStatus(422)->write('422');
    }

    if ($_FILES['file']) {
        $mime = '';
        // 投稿のContent-Typeからファイルのタイプを決定する
        if (strpos($_FILES['file']['type'], 'jpeg') !== false) {
            $mime = 'image/jpeg';
        } elseif (strpos($_FILES['file']['type'], 'png') !== false) {
            $mime = 'image/png';
        } elseif (strpos($_FILES['file']['type'], 'gif') !== false) {
            $mime = 'image/gif';
        } else {
            $this->flash->addMessage('notice', '投稿できる画像形式はjpgとpngとgifだけです');
            return redirect($response, '/', 302);
        }

        if (strlen(file_get_contents($_FILES['file']['tmp_name'])) > UPLOAD_LIMIT) {
            $this->flash->addMessage('notice', 'ファイルサイズが大きすぎます');
            return redirect($response, '/', 302);
        }

        $query = 'INSERT INTO `posts` (`user_id`, `mime`, `imgdata`, `body`) VALUES (?,?,?,?)';
        $ps = $app->db->prepare($query);
        $ps->execute(
          $me['id'],
          $mime,
          file_get_contents($_FILES['file']['tmp_name']),
          $params['body'],
        )
        $pid = $app->db->lastInsertId();
        return redirect($response, "/posts/{$pid}", 302);
    } else {
        $this->flash->addMessage('notice', '画像が必須です');
        return redirect($response, '/', 302);
    }
});

$app->get('/image/:id.:ext', function (Request $request, Response $response) {
    if ($params['id'] == 0) {
        return '';
    }

    $post = fetch_first($app->db, 'SELECT * FROM `posts` WHERE `id` = ?', [$params['id']]);

    if ($params['ext'] == 'jpg' && $post':mime'] != 'image/jpeg') ||
        ($params['ext'] == 'png' && $post['mime'] != 'image/png') ||
        ($params['ext'] == 'gif' && $post['mime'] != 'image/gif') {
        return $response->withStatus(404)->write('404');
    }

    return $response->withHeader('Content-Type', $post['mime'])
                    ->write($post['imgdata']);
});

$app->post('/comment', function (Request $request, Response $response) {
    $me = get_session_user();

    if ($me === null) {
        return redirect($response, '/login', 302);
    }

    if ($params['csrf_token'] !== session_id()) {
        return $response->withStatus(422)->write('422');
    }

    // TODO: /\A[0-9]\Z/ か確認
    if (preg_match('/[0-9]+/', $params['post_id']) === 0) {
        return $response->write('post_idは整数のみです');
    }
    $post_id = $params['post_id'];

    $query = 'INSERT INTO `comments` (`post_id`, `user_id`, `comment`) VALUES (?,?,?)';
    $ps = $app->db->prepare(query);
    $ps->execute([
        $post_id,
        $me['id'],
        $params['comment']
    ]);

    return redirect($response, "/posts/{$post_id}", 302);
});

$app->get('/admin/banned', function (Request $request, Response $response) {
    $me = get_session_user();

    if ($me === null) {
        return redirect($response, '/login', 302);
    }

    if ($me['authority'] === 0) {
        return $response->withStatus(403)->write('403');
    }

    $ps = $app->db->prepare('SELECT * FROM `users` WHERE `authority` = 0 AND `del_flg` = 0 ORDER BY `created_at` DESC');
    $ps->execute();
    $users = $ps->fetchAll(PDO::FETCH_ASSOC);

    return $this->view->render($response, 'banned.html', ['users' => $users, 'me' => $me]);
});

$app->post('/admin/banned', function (Request $request, Response $response) {
    $me = get_session_user();

    if ($me === null) {
        return redirect($response, '/login', 302);
    }

    if ($me['authority'] === 0) {
        return $response->withStatus(403)->write('403');
    }

    if ($params['csrf_token'] !== session_id()) {
        return $response->withStatus(422)->write('422');
    }

    $query = 'UPDATE `users` SET `del_flg` = ? WHERE `id` = ?';
    foreach ($params['uid'] as $id) {
        $ps = $app->db->prepare($query);
        $ps->execute([1, $id]);
    }

    return redirect($response, '/admin/banned', 302);
});

$app->run();
