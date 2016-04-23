'use strict';
var express = require('express');
var session = require('express-session');
var ejs = require('ejs');
var mysql = require('promise-mysql');
var Promise = require('bluebird');

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
app.use(session({
  'resave': true,
  'saveUninitialized': true,
  'secret': process.env.ISUCONP_SESSION_SECRET || 'sendagaya'
}));

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

function getUser(userId) {
  return new Promise((resolve, reject) => {
    db.query('SELECT * FROM `users` WHERE `id` = ?', [userId]).then(function(users) {
      resolve(users[0]);
    });
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
});

app.get('/login', function(req, res) {
});

app.post('/login', function(req, res) {
});

app.get('/register', function(req, res) {
});

app.post('/register', function(req, res) {
});

app.get('/logout', function(req, res) {
});

app.get('/', function(req, res) {
  getSessionUser(req).then(function(me) {
    db.query('SELECT `id`, `user_id`, `body`, `created_at`, `mime` FROM `posts` ORDER BY `created_at` DESC').then(function(posts) {
      makePosts(posts.slice(0, POSTS_PER_PAGE * 2)).then(function(posts) {
        res.render('index.ejs', { posts: posts.filter((post) => post.user.del_flg === 0).slice(0, POSTS_PER_PAGE), me: me, imageUrl: imageUrl });
      });
    });
  });
});

app.get('/@(.+)/', function(req, res) {
});

app.get('/posts', function(req, res) {
});

app.get('/posts/(.+)', function(req, res) {
});

app.post('/', function(req, res) {
});

app.get('/image/(.+)\.(.+)', function(req, res) {
});

app.post('/comment', function(req, res) {
});

app.get('/admin/banned', function(req, res) {
});

app.post('/admin/banned', function(req, res) {
});

app.use(express.static('../public', {}));

app.listen(8080);

