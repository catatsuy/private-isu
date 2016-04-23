'use strict';
var bodyParser = require('body-parser');
var express = require('express');
var session = require('express-session');
var flash = require('express-flash');
var ejs = require('ejs');
var mysql = require('promise-mysql');
var Promise = require('bluebird');
var exec = require('child_process').exec;

var app = express();

const POSTS_PER_PAGE = 20;

var db = mysql.createPool({
  host: process.env.ISUCONP_DB_HOST || 'localhost',
  port: process.env.ISUCONP_DB_PORT || 3306,
  user: process.env.ISUCONP_DB_USER || 'root',
  password: process.env.ISUCONP_DB_PASSWORD,
  database: process.env.ISUCONP_DB_NAME || 'isuconp',
  charset: 'utf8mb4'
});

app.engine('ejs', ejs.renderFile);
app.use(bodyParser.urlencoded({extended: true}));
app.set('etag', false);

app.use(session({
  'resave': true,
  'saveUninitialized': true,
  'secret': process.env.ISUCONP_SESSION_SECRET || 'sendagaya'
}));

app.use(flash());

function getSessionUser(req) {
  return new Promise(function(done, reject) {
    if (!req.session.userId) {
      done();
      return;
    }
    db.query('SELECT * FROM `users` WHERE `id` = ?', [req.session.userId]).then(function(users) {
      done(users[0]);
    }).catch(reject);
  });
}

function digest(src) {
  return new Promise((resolve, reject) => {
    // TODO: shellescape対策
    exec("printf \"%s\" " + src + " | openssl dgst -sha512 | sed 's/^.*= //'", (err, stdout, stderr) => {
      if (err) {
        reject(err);
        return;
      }
      resolve(stdout.replace(/^\s*(.+)\s*$/, "$1"));
    });
  });
}

function calculatePasshash(accountName, password) {
  return new Promise((resolve, reject) => {
    digest(accountName).then((salt) => {
      digest(`${password}:${salt}`).then(resolve, reject);
    }).catch(reject);
  });
}

function tryLogin(accountName, password) {
  return new Promise((resolve, reject) => {
    db.query('SELECT * FROM users WHERE account_name = ? AND del_flg = 0', accountName).then((users) => {
      let user = users[0];
      if (!user) {
        resolve();
        return;
      }
      calculatePasshash(accountName, password).then((passhash) => {
        if (passhash === user.passhash) {
          resolve(user);
        } else {
          resolve();
        }
      });
    }).catch(reject);
  });
}

function getUser(userId) {
  return new Promise((resolve, reject) => {
    db.query('SELECT * FROM `users` WHERE `id` = ?', [userId]).then(function(users) {
      resolve(users[0]);
    });
  });
}

function dbInitialize() {
  return new Promise((resolve, reject) => {
    let sqls = [];
    sqls.push('DELETE FROM users WHERE id > 1000');
    sqls.push('DELETE FROM posts WHERE id > 10000');
    sqls.push('DELETE FROM comments WHERE id > 100000');
    sqls.push('UPDATE users SET del_flg = 0');

    Promise.all(sqls.map((sql) => db.query(sql))).then(() => {
      db.query('UPDATE users SET del_flg = 1 WHERE id % 50 = 0');
    }).then(resolve, reject);
  });
}

function imageUrl(post) {
  let ext = ""

  switch(post.mime) {
  case "image/jpeg":
    ext = ".jpg";
    break;
  case "image/png":
    ext = ".png";
  case "image/gif":
    ext = ".gif";
  }

  return `/image/${post.id}${ext}`;
}

function makeComment(comment) {
  return new Promise((resolve, reject) => {
    getUser(comment.user_id).then((user) => {
      comment.user = user;
      resolve(comment);
    }).catch(reject);
  });
}

function makePost(post, options) {
  return new Promise((resolve, reject) => {
    db.query('SELECT COUNT(*) AS `count` FROM `comments` WHERE `post_id` = ?', [post.id]).then(function(commentCount) {
      post.comment_count = commentCount.count || 0;
      var query = 'SELECT * FROM `comments` WHERE `post_id` = ? ORDER BY `created_at` DESC';
      if (options.allComments) {
        query += ' LIMIT 3';
      }
      db.query(query, [post.id]).then((comments) => {
        Promise.all(comments.map((comment) => {
          return makeComment(comment);
        })).then((comments) => {
          post.comments = comments;
          getUser(post.user_id).then((user) => {
            post.user = user;
            resolve(post);
          }).catch(reject);
        }).catch(reject);
      }).catch(reject);
    }).catch(reject);
  });
}

function filterPosts(posts) {
  return posts.filter((post) => post.user.del_flg === 0).slice(0, POSTS_PER_PAGE);
}

function makePosts(posts, options) {
  if (typeof options === 'undefined') {
    options = {};
  }
  if (typeof options.allComments === 'undefined') {
    options.allComments = false;
  }
  return new Promise((resolve, reject) => {
    if (posts.length === 0) {
      resolve([]);
      return;
    }
    Promise.all(posts.map((post) => {
      return makePost(post, options);
    })).then(resolve, reject);
  });
}

app.get('/initialize', function(req, res) {
  dbInitialize().then(() => {
    res.send('OK');
  }).catch((error) => {
    console.log(error);
    res.status(500).send(error);
  });
});

app.get('/login', function(req, res) {
  getSessionUser(req).then((me) => {
    if (me) {
      res.redirect('/');
      return;
    }
    res.render('login.ejs', {me});
  });
});

app.post('/login', function(req, res) {
  getSessionUser(req).then((me) => {
    if (me) {
      res.redirect('/');
      return;
    }
    tryLogin(req.body.account_name || '', req.body.password || '').then((user) => {
      if (user) {
        req.session.userId = user.id;
        res.redirect('/');
      } else {
        req.flash('notice', 'アカウント名かパスワードが間違っています');
        res.redirect('/login');
      }
    }).catch((error) => {
      console.log(error);
      res.status(500).send(error);
    });
  });
});

app.get('/register', function(req, res) {
});

app.post('/register', function(req, res) {
});

app.get('/logout', function(req, res) {
  req.session.destroy();
  res.redirect('/');
});

app.get('/', function(req, res) {
  getSessionUser(req).then(function(me) {
    db.query('SELECT `id`, `user_id`, `body`, `created_at`, `mime` FROM `posts` ORDER BY `created_at` DESC').then(function(posts) {
      makePosts(posts.slice(0, POSTS_PER_PAGE * 2)).then(function(posts) {
        res.render('index.ejs', { posts: filterPosts(posts), me: me, imageUrl: imageUrl });
      }).catch((error) => {
        console.log(error);
        res.status(500).send(error);
      });
    }).catch((error) => {
      console.log(error);
      res.status(500).send(error);
    });
  }).catch((error) => {
    console.log(error);
    res.status(500).send(error);
  });
});

app.get('/@:accountName/', function(req, res) {
  db.query('SELECT * FROM `users` WHERE `account_name` = ? AND `del_flg` = 0', req.params.accountName).then((users) => {
    let user = users[0];
    if (!user) {
      res.status(404).send('not_found');
      return;
    }

    db.query('SELECT `id`, `user_id`, `body`, `mime`, `created_at` FROM `posts` WHERE `user_id` = ? ORDER BY `created_at` DESC', user.id).then((posts) => {
      makePosts(posts).then((posts) => {
        getSessionUser(req).then((me) => {
          res.render('user.ejs', {user: user, posts: filterPosts(posts), post_count: 0, comment_count: 0, commented_count: 0, me: me, imageUrl: imageUrl});
        });
      });
    });
  });
});

app.get('/posts', function(req, res) {
  let max_created_at = Date.parse(req.params.max_created_at) || new Date();
  db.query('SELECT `id`, `user_id`, `body`, `mime`, `created_at` FROM `posts` WHERE `created_at` <= ? ORDER BY `created_at` DESC', max_created_at).then((posts) => {
    makePosts(posts.slice(0, POSTS_PER_PAGE * 2)).then((posts) => {
      res.render('posts.ejs', {imageUrl, posts: filterPosts(posts)});
    });
  });
});

app.get('/posts/:id', function(req, res) {
  db.query('SELECT * FROM `posts` WHERE `id` = ?', req.params.id || '').then((posts) => {
    makePosts(posts, {allComments: true}).then((posts) => {
      let post = posts[0];
      if (!post) {
        res.status(404).send('not found');
        return;
      }
      getSessionUser(req).then((me) => {
        res.render('post.ejs', {imageUrl, post: post, me: me});
      });
    });
  });
});

app.post('/', function(req, res) {
});

app.get('/image/:id.:ext', function(req, res) {
  db.query('SELECT * FROM `posts` WHERE `id` = ?', req.params.id).then((posts) => {
    let post = posts[0];
    if (!post) {
      res.status(404).send('image not found');
      return;
    }
    if ((req.params.ext === 'jpg' && post.mime === 'image/jpeg') ||
        (req.params.ext === 'jpg' && post.mime === 'image/jpeg') ||
        (req.params.ext === 'jpg' && post.mime === 'image/jpeg')) {
      res.append('Content-Type', post.mime);
      res.send(post.imgdata);
    }
  }).catch((error) => {
    console.log(error);
    res.status(500).send(error);
  }) ;
});

app.post('/comment', function(req, res) {
});

app.get('/admin/banned', function(req, res) {
});

app.post('/admin/banned', function(req, res) {
});

app.use(express.static('../public', {}));

app.listen(8080);

