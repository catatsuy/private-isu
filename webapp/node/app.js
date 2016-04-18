var express = require('express');
var ejs = require('ejs');
var app = express();

app.engine('ejs', ejs.renderFile);

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
  res.render('index.ejs', { posts: [] });
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

