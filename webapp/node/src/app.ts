import express from 'express';
import session from 'express-session';
import flash from 'express-flash';
import ejs from 'ejs';
import multer from 'multer';
import {createPool, Pool} from 'mysql2/promise';
import crypto from 'crypto';
import connectMemcached from 'connect-memcached';

const MemcachedStore = connectMemcached(session);
const app = express();
const upload = multer({});

const POSTS_PER_PAGE = 20;
const UPLOAD_LIMIT = 10 * 1024 * 1024; // 10mb

const db: Pool = createPool({
  host: process.env.ISUCONP_DB_HOST || 'localhost',
  port: Number(process.env.ISUCONP_DB_PORT) || 3306,
  user: process.env.ISUCONP_DB_USER || 'root',
  password: process.env.ISUCONP_DB_PASSWORD,
  database: process.env.ISUCONP_DB_NAME || 'isuconp',
  connectionLimit: 1,
  charset: 'utf8mb4'
});

app.engine('ejs', ejs.renderFile as any);
app.use(express.urlencoded({ extended: true }));
app.set('etag', false);

app.use(
  session({
    resave: true,
    saveUninitialized: true,
    secret: process.env.ISUCONP_SESSION_SECRET || 'sendagaya',
    store: new MemcachedStore({ hosts: [process.env.ISUCONP_MEMCACHED_ADDRESS || 'localhost:11211'] })
  })
);

app.use(flash());

async function getSessionUser(req: express.Request) {
  if (!req.session.userId) return undefined;
  const [rows] = await db.query<any[]>(
    'SELECT * FROM `users` WHERE `id` = ?',
    [req.session.userId]
  );
  const user = rows[0];
  if (user) user.csrfToken = req.session.csrfToken;
  return user;
}

function digest(src: string) {
  return crypto.createHash('sha512').update(src).digest('hex');
}

function validateUser(accountName: string, password: string) {
  return (
    /^[0-9a-zA-Z_]{3,}$/.test(accountName) &&
    /^[0-9a-zA-Z_]{6,}$/.test(password)
  );
}

function calculatePasshash(accountName: string, password: string) {
  const salt = digest(accountName);
  return digest(`${password}:${salt}`);
}

async function tryLogin(accountName: string, password: string) {
  const [rows] = await db.query<any[]>(
    'SELECT * FROM users WHERE account_name = ? AND del_flg = 0',
    [accountName]
  );
  const user = rows[0];
  if (!user) return undefined;
  const passhash = calculatePasshash(accountName, password);
  if (passhash === user.passhash) return user;
  return undefined;
}

async function getUser(userId: number) {
  const [rows] = await db.query<any[]>(
    'SELECT * FROM `users` WHERE `id` = ?',
    [userId]
  );
  return rows[0];
}

async function dbInitialize() {
  const sqls = [
    'DELETE FROM users WHERE id > 1000',
    'DELETE FROM posts WHERE id > 10000',
    'DELETE FROM comments WHERE id > 100000',
    'UPDATE users SET del_flg = 0'
  ];
  await Promise.all(sqls.map((sql) => db.query(sql)));
  await db.query('UPDATE users SET del_flg = 1 WHERE id % 50 = 0');
}

function imageUrl(post: any) {
  let ext = '';
  switch (post.mime) {
    case 'image/jpeg':
      ext = '.jpg';
      break;
    case 'image/png':
      ext = '.png';
      break;
    case 'image/gif':
      ext = '.gif';
      break;
  }
  return `/image/${post.id}${ext}`;
}

async function makeComment(comment: any) {
  comment.user = await getUser(comment.user_id);
  return comment;
}

async function makePost(post: any, options: { allComments?: boolean } = {}) {
  const [[countRow]] = await db.query<any[]>(
    'SELECT COUNT(*) AS `count` FROM `comments` WHERE `post_id` = ?',
    [post.id]
  );
  post.comment_count = countRow.count || 0;
  let query =
    'SELECT * FROM `comments` WHERE `post_id` = ? ORDER BY `created_at` DESC';
  if (!options.allComments) {
    query += ' LIMIT 3';
  }
  const [comments] = await db.query<any[]>(query, [post.id]);
  post.comments = await Promise.all(comments.map(makeComment));
  post.user = await getUser(post.user_id);
  return post;
}

function filterPosts(posts: any[]) {
  return posts.filter((p) => p.user.del_flg === 0).slice(0, POSTS_PER_PAGE);
}

async function makePosts(posts: any[], options: { allComments?: boolean } = {}) {
  if (posts.length === 0) return [];
  return Promise.all(posts.map((p) => makePost(p, options)));
}

app.get('/initialize', async (_req, res) => {
  try {
    await dbInitialize();
    res.send('OK');
  } catch (e) {
    console.error(e);
    res.status(500).send(String(e));
  }
});

app.get('/login', async (req, res) => {
  const me = await getSessionUser(req);
  if (me) {
    res.redirect('/');
    return;
  }
  res.render('login.ejs', { me });
});

app.post('/login', async (req, res) => {
  const me = await getSessionUser(req);
  if (me) {
    res.redirect('/');
    return;
  }
  try {
    const user = await tryLogin(
      req.body.account_name || '',
      req.body.password || ''
    );
    if (user) {
      req.session.userId = user.id;
      req.session.csrfToken = crypto.randomBytes(16).toString('hex');
      res.redirect('/');
    } else {
      req.flash('notice', 'アカウント名かパスワードが間違っています');
      res.redirect('/login');
    }
  } catch (e) {
    console.error(e);
    res.status(500).send(String(e));
  }
});

app.get('/register', async (req, res) => {
  const me = await getSessionUser(req);
  if (me) {
    res.redirect('/');
    return;
  }
  res.render('register.ejs', { me });
});

app.post('/register', async (req, res) => {
  const me = await getSessionUser(req);
  if (me) {
    res.redirect('/');
    return;
  }
  const accountName = req.body.account_name || '';
  const password = req.body.password || '';
  if (!validateUser(accountName, password)) {
    req.flash('notice', 'アカウント名は3文字以上、パスワードは6文字以上である必要があります');
    res.redirect('/register');
    return;
  }
  const [rows] = await db.query<any[]>(
    'SELECT 1 FROM users WHERE `account_name` = ?',
    [accountName]
  );
  if (rows[0]) {
    req.flash('notice', 'アカウント名がすでに使われています');
    res.redirect('/register');
    return;
  }
  const passhash = calculatePasshash(accountName, password);
  await db.query(
    'INSERT INTO `users` (`account_name`, `passhash`) VALUES (?, ?)',
    [accountName, passhash]
  );
  const [meRow] = await db.query<any[]>(
    'SELECT * FROM `users` WHERE `account_name` = ?',
    [accountName]
  );
  const newUser = meRow[0];
  req.session.userId = newUser.id;
  req.session.csrfToken = crypto.randomBytes(16).toString('hex');
  res.redirect('/');
});

app.get('/logout', (req, res) => {
  req.session.destroy(() => {
    res.redirect('/');
  });
});

app.get('/', async (req, res) => {
  try {
    const me = await getSessionUser(req);
    const [posts] = await db.query<any[]>(
      'SELECT `id`, `user_id`, `body`, `created_at`, `mime` FROM `posts` ORDER BY `created_at` DESC'
    );
    const enriched = await makePosts(posts.slice(0, POSTS_PER_PAGE * 2));
    res.render('index.ejs', { posts: filterPosts(enriched), me, imageUrl });
  } catch (e) {
    console.error(e);
    res.status(500).send(String(e));
  }
});

app.get('/@:accountName/', async (req, res) => {
  try {
    const [urows] = await db.query<any[]>(
      'SELECT * FROM `users` WHERE `account_name` = ? AND `del_flg` = 0',
      [req.params.accountName]
    );
    const user = urows[0];
    if (!user) {
      res.status(404).send('not_found');
      return;
    }
    const [postRows] = await db.query<any[]>(
      'SELECT `id`, `user_id`, `body`, `mime`, `created_at` FROM `posts` WHERE `user_id` = ? ORDER BY `created_at` DESC',
      [user.id]
    );
    const posts = await makePosts(postRows);
    const [commentCountRows] = await db.query<any[]>(
      'SELECT COUNT(*) AS count FROM `comments` WHERE `user_id` = ?',
      [user.id]
    );
    const commentCount = commentCountRows[0] ? commentCountRows[0].count : 0;
    const [postIdRows] = await db.query<any[]>(
      'SELECT `id` FROM `posts` WHERE `user_id` = ?',
      [user.id]
    );
    const postIds = postIdRows.map((r: any) => r.id);
    const postCount = postIds.length;
    let commentedCount = 0;
    if (postCount > 0) {
      const [countRows] = await db.query<any[]>(
        'SELECT COUNT(*) AS count FROM `comments` WHERE `post_id` IN (?)',
        [postIds]
      );
      commentedCount = countRows[0] ? countRows[0].count : 0;
    }
    const me = await getSessionUser(req);
    res.render('user.ejs', {
      me,
      user,
      posts: filterPosts(posts),
      post_count: postCount,
      comment_count: commentCount,
      commented_count: commentedCount,
      imageUrl
    });
  } catch (e) {
    console.error(e);
    res.status(500).send('ERROR');
  }
});

app.get('/posts', async (req, res) => {
  let maxCreatedAt = new Date(req.query.max_created_at as string);
  if (maxCreatedAt.toString() === 'Invalid Date') {
    maxCreatedAt = new Date();
  }
  const [posts] = await db.query<any[]>(
    'SELECT `id`, `user_id`, `body`, `mime`, `created_at` FROM `posts` WHERE `created_at` <= ? ORDER BY `created_at` DESC',
    [maxCreatedAt]
  );
  const enriched = await makePosts(posts.slice(0, POSTS_PER_PAGE * 2));
  const me = await getSessionUser(req);
  res.render('posts.ejs', { me, imageUrl, posts: filterPosts(enriched) });
});

app.get('/posts/:id', async (req, res) => {
  const [posts] = await db.query<any[]>(
    'SELECT * FROM `posts` WHERE `id` = ?',
    [req.params.id]
  );
  const enriched = await makePosts(posts as any[], { allComments: true });
  const post = enriched[0];
  if (!post) {
    res.status(404).send('not found');
    return;
  }
  const me = await getSessionUser(req);
  res.render('post.ejs', { imageUrl, post, me });
});

app.post('/', upload.single('file'), async (req, res) => {
  const me = await getSessionUser(req);
  if (!me) {
    res.redirect('/login');
    return;
  }
  if (req.body.csrf_token !== req.session.csrfToken) {
    res.status(422).send('invalid CSRF Token');
    return;
  }
  if (!req.file) {
    req.flash('notice', '画像が必須です');
    res.redirect('/');
    return;
  }
  let mime = '';
  if (req.file.mimetype.includes('jpeg')) {
    mime = 'image/jpeg';
  } else if (req.file.mimetype.includes('png')) {
    mime = 'image/png';
  } else if (req.file.mimetype.includes('gif')) {
    mime = 'image/gif';
  } else {
    req.flash('notice', '投稿できる画像形式はjpgとpngとgifだけです');
    res.redirect('/');
    return;
  }
  if (req.file.size > UPLOAD_LIMIT) {
    req.flash('notice', 'ファイルサイズが大きすぎます');
    res.redirect('/');
    return;
  }
  const [result] = await db.query<any>(
    'INSERT INTO `posts` (`user_id`, `mime`, `imgdata`, `body`) VALUES (?,?,?,?)',
    [me.id, mime, req.file.buffer, req.body.body]
  );
  const insertId = (result as any).insertId;
  res.redirect(`/posts/${encodeURIComponent(insertId)}`);
});

app.get('/image/:id.:ext', async (req, res) => {
  try {
    const [posts] = await db.query<any[]>(
      'SELECT * FROM `posts` WHERE `id` = ?',
      [req.params.id]
    );
    const post = posts[0];
    if (!post) {
      res.status(404).send('image not found');
      return;
    }
    if (
      (req.params.ext === 'jpg' && post.mime === 'image/jpeg') ||
      (req.params.ext === 'png' && post.mime === 'image/png') ||
      (req.params.ext === 'gif' && post.mime === 'image/gif')
    ) {
      res.contentType(post.mime);
      res.send(post.imgdata);
    }
  } catch (e) {
    console.error(e);
    res.status(500).send(String(e));
  }
});

app.post('/comment', async (req, res) => {
  const me = await getSessionUser(req);
  if (!me) {
    res.redirect('/login');
    return;
  }
  if (req.body.csrf_token !== req.session.csrfToken) {
    res.status(422).send('invalid CSRF Token');
    return;
  }
  if (!req.body.post_id || !/^[0-9]+$/.test(req.body.post_id)) {
    res.send('post_idは整数のみです');
    return;
  }
  await db.query(
    'INSERT INTO `comments` (`post_id`, `user_id`, `comment`) VALUES (?,?,?)',
    [req.body.post_id, me.id, req.body.comment || '']
  );
  res.redirect(`/posts/${encodeURIComponent(req.body.post_id)}`);
});

app.get('/admin/banned', async (req, res) => {
  const me = await getSessionUser(req);
  if (!me) {
    res.redirect('/login');
    return;
  }
  if (me.authority === 0) {
    res.status(403).send('authority is required');
    return;
  }
  const [users] = await db.query<any[]>(
    'SELECT * FROM `users` WHERE `authority` = 0 AND `del_flg` = 0 ORDER BY `created_at` DESC'
  );
  res.render('banned.ejs', { me, users });
});

app.post('/admin/banned', async (req, res) => {
  const me = await getSessionUser(req);
  if (!me) {
    res.redirect('/');
    return;
  }
  if (me.authority === 0) {
    res.status(403).send('authority is required');
    return;
  }
  if (req.body.csrf_token !== req.session.csrfToken) {
    res.status(422).send('invalid CSRF Token');
    return;
  }
  const query = 'UPDATE `users` SET `del_flg` = ? WHERE `id` = ?';
  const ids = Array.isArray(req.body.uid) ? req.body.uid : [req.body.uid];
  await Promise.all(ids.map((id: any) => db.query(query, [1, id])));
  res.redirect('/admin/banned');
});

app.use(express.static('../public', {}));

const port = 8080;
app.listen(port, () => {
  console.log(`server started on ${port}`);
});
