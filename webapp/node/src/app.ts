import express, { Request, Response } from 'express';
import session from 'express-session';
import flash from 'express-flash';
import ejs, { Data as EjsData } from 'ejs';
import multer from 'multer';
import {
  createPool,
  Pool,
  RowDataPacket,
  ResultSetHeader,
} from 'mysql2/promise';
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

interface User {
  id: number;
  account_name: string;
  passhash: string;
  authority: number;
  del_flg: number;
  created_at: Date;
  csrfToken?: string;
}

interface Post {
  id: number;
  user_id: number;
  imgdata: Buffer;
  body: string;
  mime: string;
  created_at: Date;
  comment_count?: number;
  comments?: Comment[];
  user?: User;
}

interface Comment {
  id: number;
  post_id: number;
  user_id: number;
  comment: string;
  created_at: Date;
  user?: User;
}

interface CountRow {
  count: number;
}

interface IdRow {
  id: number;
}

app.engine('ejs', (path, options, callback) => {
  ejs.renderFile(path, options as EjsData, {}, callback);
});
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

async function getSessionUser(req: Request): Promise<User | undefined> {
  if (!req.session.userId) return undefined;
  const [rows] = await db.query<RowDataPacket[]>(
    'SELECT * FROM `users` WHERE `id` = ?',
    [req.session.userId]
  );
  const user = rows[0] as User;
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

async function tryLogin(accountName: string, password: string): Promise<User | undefined> {
  const [rows] = await db.query<RowDataPacket[]>(
    'SELECT * FROM users WHERE account_name = ? AND del_flg = 0',
    [accountName]
  );
  const user = rows[0] as User;
  if (!user) return undefined;
  const passhash = calculatePasshash(accountName, password);
  if (passhash === user.passhash) return user;
  return undefined;
}

async function getUser(userId: number) {
  const [rows] = await db.query<RowDataPacket[]>(
    'SELECT * FROM `users` WHERE `id` = ?',
    [userId]
  );
  return rows[0] as User;
}

async function dbInitialize() {
  const sqls = [
    'DELETE FROM users WHERE id > 1000',
    'DELETE FROM posts WHERE id > 10000',
    'DELETE FROM comments WHERE id > 100000',
    'UPDATE users SET del_flg = 0'
  ];
  await Promise.all(sqls.map((sql) => db.query<ResultSetHeader>(sql)));
  await db.query<ResultSetHeader>('UPDATE users SET del_flg = 1 WHERE id % 50 = 0');
}

function imageUrl(post: Pick<Post, 'id' | 'mime'>) {
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

async function makeComment(comment: Comment): Promise<Comment> {
  comment.user = await getUser(comment.user_id);
  return comment;
}

async function makePost(post: Post, options: { allComments?: boolean } = {}): Promise<Post> {
  const [[countRow]] = await db.query<RowDataPacket[]>(
    'SELECT COUNT(*) AS `count` FROM `comments` WHERE `post_id` = ?',
    [post.id]
  );
  post.comment_count = (countRow as CountRow | undefined)?.count || 0;
  let query =
    'SELECT * FROM `comments` WHERE `post_id` = ? ORDER BY `created_at` DESC';
  if (!options.allComments) {
    query += ' LIMIT 3';
  }
  const [commentRows] = await db.query<RowDataPacket[]>(query, [post.id]);
  post.comments = await Promise.all((commentRows as Comment[]).map(makeComment));
  post.user = await getUser(post.user_id);
  return post;
}

function filterPosts(posts: Post[]): Post[] {
  return posts.filter((p) => p.user && p.user.del_flg === 0).slice(0, POSTS_PER_PAGE);
}

async function makePosts(posts: Post[], options: { allComments?: boolean } = {}): Promise<Post[]> {
  if (posts.length === 0) return [];
  return Promise.all(posts.map((p) => makePost(p, options)));
}

app.get('/initialize', async (_req: Request, res: Response) => {
  try {
    await dbInitialize();
    res.send('OK');
  } catch (e) {
    console.error(e);
    res.status(500).send(String(e));
  }
});

app.get('/login', async (req: Request, res: Response) => {
  const me = await getSessionUser(req);
  if (me) {
    res.redirect('/');
    return;
  }
  res.render('login.ejs', { me });
});

app.post('/login', async (req: Request, res: Response) => {
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

app.get('/register', async (req: Request, res: Response) => {
  const me = await getSessionUser(req);
  if (me) {
    res.redirect('/');
    return;
  }
  res.render('register.ejs', { me });
});

app.post('/register', async (req: Request, res: Response) => {
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
  const [rows] = await db.query<RowDataPacket[]>(
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
  const [meRow] = await db.query<RowDataPacket[]>(
    'SELECT * FROM `users` WHERE `account_name` = ?',
    [accountName]
  );
  const newUser = meRow[0] as User;
  req.session.userId = newUser.id;
  req.session.csrfToken = crypto.randomBytes(16).toString('hex');
  res.redirect('/');
});

app.get('/logout', (req, res) => {
  req.session.destroy(() => {
    res.redirect('/');
  });
});

app.get('/', async (req: Request, res: Response) => {
  try {
    const me = await getSessionUser(req);
  const [postRows] = await db.query<RowDataPacket[]>(
    'SELECT `id`, `user_id`, `body`, `created_at`, `mime` FROM `posts` ORDER BY `created_at` DESC'
  );
  const posts = postRows as Post[];
  const enriched = await makePosts(posts.slice(0, POSTS_PER_PAGE * 2));
    res.render('index.ejs', { posts: filterPosts(enriched), me, imageUrl });
  } catch (e) {
    console.error(e);
    res.status(500).send(String(e));
  }
});

app.get('/@:accountName/', async (req: Request, res: Response) => {
  try {
    const [urows] = await db.query<RowDataPacket[]>(
      'SELECT * FROM `users` WHERE `account_name` = ? AND `del_flg` = 0',
      [req.params.accountName]
    );
    const user = urows[0] as User;
    if (!user) {
      res.status(404).send('not_found');
      return;
    }
    const [postRowData] = await db.query<RowDataPacket[]>(
      'SELECT `id`, `user_id`, `body`, `mime`, `created_at` FROM `posts` WHERE `user_id` = ? ORDER BY `created_at` DESC',
      [user.id]
    );
    const posts = await makePosts(postRowData as Post[]);
    const [commentCountRows] = await db.query<RowDataPacket[]>(
      'SELECT COUNT(*) AS count FROM `comments` WHERE `user_id` = ?',
      [user.id]
    );
    const commentCount = (commentCountRows[0] as CountRow | undefined)?.count ?? 0;
    const [postIdRows] = await db.query<RowDataPacket[]>(
      'SELECT `id` FROM `posts` WHERE `user_id` = ?',
      [user.id]
    );
    const postIds = (postIdRows as IdRow[]).map((r) => r.id);
    const postCount = postIds.length;
    let commentedCount = 0;
    if (postCount > 0) {
      const [countRows] = await db.query<RowDataPacket[]>(
        'SELECT COUNT(*) AS count FROM `comments` WHERE `post_id` IN (?)',
        [postIds]
      );
      commentedCount = (countRows[0] as CountRow | undefined)?.count ?? 0;
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

app.get('/posts', async (req: Request, res: Response) => {
  let maxCreatedAt = new Date(req.query.max_created_at as string);
  if (maxCreatedAt.toString() === 'Invalid Date') {
    maxCreatedAt = new Date();
  }
  const [postRows] = await db.query<RowDataPacket[]>(
    'SELECT `id`, `user_id`, `body`, `mime`, `created_at` FROM `posts` WHERE `created_at` <= ? ORDER BY `created_at` DESC',
    [maxCreatedAt]
  );
  const posts = postRows as Post[];
  const enriched = await makePosts(posts.slice(0, POSTS_PER_PAGE * 2));
  const me = await getSessionUser(req);
  res.render('posts.ejs', { me, imageUrl, posts: filterPosts(enriched) });
});

app.get('/posts/:id', async (req: Request, res: Response) => {
  const [postRows] = await db.query<RowDataPacket[]>(
    'SELECT * FROM `posts` WHERE `id` = ?',
    [req.params.id]
  );
  const posts = postRows as Post[];
  const enriched = await makePosts(posts, { allComments: true });
  const post = enriched[0];
  if (!post) {
    res.status(404).send('not found');
    return;
  }
  const me = await getSessionUser(req);
  res.render('post.ejs', { imageUrl, post, me });
});

app.post('/', upload.single('file'), async (req: Request, res: Response) => {
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
  const [result] = await db.query<ResultSetHeader>(
    'INSERT INTO `posts` (`user_id`, `mime`, `imgdata`, `body`) VALUES (?,?,?,?)',
    [me.id, mime, req.file.buffer, req.body.body]
  );
  const insertId = result.insertId;
  res.redirect(`/posts/${encodeURIComponent(insertId)}`);
});

app.get('/image/:id.:ext', async (req: Request, res: Response) => {
  try {
    const [posts] = await db.query<RowDataPacket[]>(
      'SELECT * FROM `posts` WHERE `id` = ?',
      [req.params.id]
    );
    const post = (posts as Post[])[0];
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

app.post('/comment', async (req: Request, res: Response) => {
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
  await db.query<ResultSetHeader>(
    'INSERT INTO `comments` (`post_id`, `user_id`, `comment`) VALUES (?,?,?)',
    [req.body.post_id, me.id, req.body.comment || '']
  );
  res.redirect(`/posts/${encodeURIComponent(req.body.post_id)}`);
});

app.get('/admin/banned', async (req: Request, res: Response) => {
  const me = await getSessionUser(req);
  if (!me) {
    res.redirect('/login');
    return;
  }
  if (me.authority === 0) {
    res.status(403).send('authority is required');
    return;
  }
  const [userRows] = await db.query<RowDataPacket[]>(
    'SELECT * FROM `users` WHERE `authority` = 0 AND `del_flg` = 0 ORDER BY `created_at` DESC'
  );
  const users = userRows as User[];
  res.render('banned.ejs', { me, users });
});

app.post('/admin/banned', async (req: Request, res: Response) => {
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
  const ids: string[] = Array.isArray(req.body.uid) ? req.body.uid : [req.body.uid];
  await Promise.all(ids.map((id: string) => db.query<ResultSetHeader>(query, [1, id])));
  res.redirect('/admin/banned');
});

app.use(express.static('../public', {}));

const port = 8080;
app.listen(port, () => {
  console.log(`server started on ${port}`);
});
