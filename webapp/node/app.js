var express = require('express');
var session = require('express-session');
var ejs = require('ejs');
var mysql = require('promise-mysql');
var Promise = require('bluebird');

var app = express();

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
      res.render('index.ejs', { posts: posts, me: me });
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

