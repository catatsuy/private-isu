import { Hono, type Context } from 'hono'
import { serve } from '@hono/node-server'
import { getCookie, setCookie } from 'hono/cookie'
import { serveStatic } from '@hono/node-server/serve-static'
import ejs from 'ejs'
import { createPool, Pool, RowDataPacket, ResultSetHeader } from 'mysql2/promise'
import crypto from 'crypto'
import path from 'path'
import { fileURLToPath } from 'url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))

type Variables = {
  session: SessionData
}

const app = new Hono<{ Variables: Variables }>()
type AppContext = Context<{ Variables: Variables }>

const POSTS_PER_PAGE = 20
const UPLOAD_LIMIT = 10 * 1024 * 1024

const db: Pool = createPool({
  host: process.env.ISUCONP_DB_HOST || 'localhost',
  port: Number(process.env.ISUCONP_DB_PORT) || 3306,
  user: process.env.ISUCONP_DB_USER || 'root',
  password: process.env.ISUCONP_DB_PASSWORD,
  database: process.env.ISUCONP_DB_NAME || 'isuconp',
  connectionLimit: 1,
  charset: 'utf8mb4'
})

interface User {
  id: number
  account_name: string
  passhash: string
  authority: number
  del_flg: number
  created_at: Date
  csrfToken?: string
}

interface Post {
  id: number
  user_id: number
  imgdata: Buffer
  body: string
  mime: string
  created_at: Date
  comment_count?: number
  comments?: Comment[]
  user?: User
}

interface Comment {
  id: number
  post_id: number
  user_id: number
  comment: string
  created_at: Date
  user?: User
}

interface CountRow {
  count: number
}

interface IdRow {
  id: number
}

interface SessionData {
  userId?: number
  csrfToken?: string
  flashNotice?: string
}

const sessions = new Map<string, SessionData>()

function generateSessionId() {
  return crypto.randomBytes(16).toString('hex')
}

app.use('*', async (c, next) => {
  let sid = getCookie(c, 'sid')
  let session = sid ? sessions.get(sid) : undefined
  if (!sid || !session) {
    sid = generateSessionId()
    session = {}
    sessions.set(sid, session)
    setCookie(c, 'sid', sid, { httpOnly: true, path: '/' })
  }
  c.set('session', session)
  await next()
})

function digest(src: string) {
  return crypto.createHash('sha512').update(src).digest('hex')
}

function validateUser(accountName: string, password: string) {
  return /^[0-9a-zA-Z_]{3,}$/.test(accountName) && /^[0-9a-zA-Z_]{6,}$/.test(password)
}

function calculatePasshash(accountName: string, password: string) {
  const salt = digest(accountName)
  return digest(`${password}:${salt}`)
}

async function tryLogin(accountName: string, password: string): Promise<User | undefined> {
  const [rows] = await db.query<RowDataPacket[]>('SELECT * FROM users WHERE account_name = ? AND del_flg = 0', [accountName])
  const user = rows[0] as User
  if (!user) return undefined
  const passhash = calculatePasshash(accountName, password)
  if (passhash === user.passhash) return user
  return undefined
}

async function getUser(userId: number) {
  const [rows] = await db.query<RowDataPacket[]>('SELECT * FROM `users` WHERE `id` = ?', [userId])
  return rows[0] as User
}

async function dbInitialize() {
  const sqls = [
    'DELETE FROM users WHERE id > 1000',
    'DELETE FROM posts WHERE id > 10000',
    'DELETE FROM comments WHERE id > 100000',
    'UPDATE users SET del_flg = 0'
  ]
  await Promise.all(sqls.map((sql) => db.query<ResultSetHeader>(sql)))
  await db.query<ResultSetHeader>('UPDATE users SET del_flg = 1 WHERE id % 50 = 0')
}

function imageUrl(post: Pick<Post, 'id' | 'mime'>) {
  let ext = ''
  switch (post.mime) {
    case 'image/jpeg':
      ext = '.jpg'
      break
    case 'image/png':
      ext = '.png'
      break
    case 'image/gif':
      ext = '.gif'
      break
  }
  return `/image/${post.id}${ext}`
}

async function makeComment(comment: Comment): Promise<Comment> {
  comment.user = await getUser(comment.user_id)
  return comment
}

async function makePost(post: Post, options: { allComments?: boolean } = {}): Promise<Post> {
  const [[countRow]] = await db.query<RowDataPacket[]>('SELECT COUNT(*) AS `count` FROM `comments` WHERE `post_id` = ?', [post.id])
  post.comment_count = (countRow as CountRow | undefined)?.count || 0
  let query = 'SELECT * FROM `comments` WHERE `post_id` = ? ORDER BY `created_at` DESC'
  if (!options.allComments) {
    query += ' LIMIT 3'
  }
  const [commentRows] = await db.query<RowDataPacket[]>(query, [post.id])
  post.comments = await Promise.all((commentRows as Comment[]).map(makeComment))
  post.user = await getUser(post.user_id)
  return post
}

function filterPosts(posts: Post[]): Post[] {
  return posts.filter((p) => p.user && p.user.del_flg === 0).slice(0, POSTS_PER_PAGE)
}

async function makePosts(posts: Post[], options: { allComments?: boolean } = {}): Promise<Post[]> {
  if (posts.length === 0) return []
  return Promise.all(posts.map((p) => makePost(p, options)))
}

async function render(c: AppContext, view: string, params: Record<string, unknown>) {
  const session = c.get('session') as SessionData
  const messages = { notice: session.flashNotice }
  session.flashNotice = undefined
  const html = await ejs.renderFile(path.join(__dirname, '../views', view), { ...params, messages })
  return c.html(html)
}

async function getSessionUser(c: AppContext): Promise<User | undefined> {
  const session = c.get('session') as SessionData
  if (!session.userId) return undefined
  const [rows] = await db.query<RowDataPacket[]>('SELECT * FROM `users` WHERE `id` = ?', [session.userId])
  const user = rows[0] as User
  if (user) user.csrfToken = session.csrfToken
  return user
}

app.get('/initialize', async (c) => {
  try {
    await dbInitialize()
    return c.text('OK')
  } catch (e) {
    console.error(e)
    return c.text(String(e), 500)
  }
})

app.get('/login', async (c) => {
  const me = await getSessionUser(c)
  if (me) return c.redirect('/')
  return render(c, 'login.ejs', { me })
})

app.post('/login', async (c) => {
  const me = await getSessionUser(c)
  if (me) return c.redirect('/')
  const body = await c.req.parseBody()
  try {
    const user = await tryLogin(String(body.account_name || ''), String(body.password || ''))
    const session = c.get('session') as SessionData
    if (user) {
      session.userId = user.id
      session.csrfToken = crypto.randomBytes(16).toString('hex')
      return c.redirect('/')
    } else {
      session.flashNotice = 'アカウント名かパスワードが間違っています'
      return c.redirect('/login')
    }
  } catch (e) {
    console.error(e)
    return c.text(String(e), 500)
  }
})

app.get('/register', async (c) => {
  const me = await getSessionUser(c)
  if (me) return c.redirect('/')
  return render(c, 'register.ejs', { me })
})

app.post('/register', async (c) => {
  const me = await getSessionUser(c)
  if (me) return c.redirect('/')
  const body = await c.req.parseBody()
  const accountName = String(body.account_name || '')
  const password = String(body.password || '')
  const session = c.get('session') as SessionData
  if (!validateUser(accountName, password)) {
    session.flashNotice = 'アカウント名は3文字以上、パスワードは6文字以上である必要があります'
    return c.redirect('/register')
  }
  const [rows] = await db.query<RowDataPacket[]>('SELECT 1 FROM users WHERE `account_name` = ?', [accountName])
  if (rows[0]) {
    session.flashNotice = 'アカウント名がすでに使われています'
    return c.redirect('/register')
  }
  const passhash = calculatePasshash(accountName, password)
  await db.query('INSERT INTO `users` (`account_name`, `passhash`) VALUES (?, ?)', [accountName, passhash])
  const [meRow] = await db.query<RowDataPacket[]>('SELECT * FROM `users` WHERE `account_name` = ?', [accountName])
  const newUser = meRow[0] as User
  session.userId = newUser.id
  session.csrfToken = crypto.randomBytes(16).toString('hex')
  return c.redirect('/')
})

app.get('/logout', (c) => {
  const sid = getCookie(c, 'sid')
  if (sid) sessions.delete(sid)
  setCookie(c, 'sid', '', { maxAge: 0, path: '/' })
  return c.redirect('/')
})

app.get('/', async (c) => {
  try {
    const me = await getSessionUser(c)
    const [postRows] = await db.query<RowDataPacket[]>('SELECT `id`, `user_id`, `body`, `created_at`, `mime` FROM `posts` ORDER BY `created_at` DESC')
    const posts = postRows as Post[]
    const enriched = await makePosts(posts.slice(0, POSTS_PER_PAGE * 2))
    return render(c, 'index.ejs', { posts: filterPosts(enriched), me, imageUrl })
  } catch (e) {
    console.error(e)
    return c.text(String(e), 500)
  }
})

app.get('/:accountName{@[A-Za-z0-9_]+}', async (c) => {
  try {
    const accountName = c.req.param('accountName').slice(1)
    const [urows] = await db.query<RowDataPacket[]>('SELECT * FROM `users` WHERE `account_name` = ? AND `del_flg` = 0', [accountName])
    const user = urows[0] as User
    if (!user) return c.text('not_found', 404)
    const [postRowData] = await db.query<RowDataPacket[]>('SELECT `id`, `user_id`, `body`, `mime`, `created_at` FROM `posts` WHERE `user_id` = ? ORDER BY `created_at` DESC', [user.id])
    const posts = await makePosts(postRowData as Post[])
    const [commentCountRows] = await db.query<RowDataPacket[]>('SELECT COUNT(*) AS count FROM `comments` WHERE `user_id` = ?', [user.id])
    const commentCount = (commentCountRows[0] as CountRow | undefined)?.count ?? 0
    const [postIdRows] = await db.query<RowDataPacket[]>('SELECT `id` FROM `posts` WHERE `user_id` = ?', [user.id])
    const postIds = (postIdRows as IdRow[]).map((r) => r.id)
    const postCount = postIds.length
    let commentedCount = 0
    if (postCount > 0) {
      const [countRows] = await db.query<RowDataPacket[]>('SELECT COUNT(*) AS count FROM `comments` WHERE `post_id` IN (?)', [postIds])
      commentedCount = (countRows[0] as CountRow | undefined)?.count ?? 0
    }
    const me = await getSessionUser(c)
    return render(c, 'user.ejs', { me, user, posts: filterPosts(posts), post_count: postCount, comment_count: commentCount, commented_count: commentedCount, imageUrl })
  } catch (e) {
    console.error(e)
    return c.text('ERROR', 500)
  }
})

app.get('/posts', async (c) => {
  let maxCreatedAt = new Date(String(c.req.query('max_created_at') || ''))
  if (maxCreatedAt.toString() === 'Invalid Date') maxCreatedAt = new Date()
  const [postRows] = await db.query<RowDataPacket[]>('SELECT `id`, `user_id`, `body`, `mime`, `created_at` FROM `posts` WHERE `created_at` <= ? ORDER BY `created_at` DESC', [maxCreatedAt])
  const posts = postRows as Post[]
  const enriched = await makePosts(posts.slice(0, POSTS_PER_PAGE * 2))
  const me = await getSessionUser(c)
  return render(c, 'posts.ejs', { me, imageUrl, posts: filterPosts(enriched) })
})

app.get('/posts/:id', async (c) => {
  const id = c.req.param('id')
  const [postRows] = await db.query<RowDataPacket[]>('SELECT * FROM `posts` WHERE `id` = ?', [id])
  const posts = postRows as Post[]
  const enriched = await makePosts(posts, { allComments: true })
  const post = enriched[0]
  if (!post) return c.text('not found', 404)
  const me = await getSessionUser(c)
  return render(c, 'post.ejs', { imageUrl, post, me })
})

app.post('/', async (c) => {
  const me = await getSessionUser(c)
  if (!me) return c.redirect('/login')
  const body = await c.req.parseBody()
  const session = c.get('session') as SessionData
  if (body.csrf_token !== session.csrfToken) return c.text('invalid CSRF Token', 422)
  const file = body.file as File | undefined
  if (!file) {
    session.flashNotice = '画像が必須です'
    return c.redirect('/')
  }
  let mime = file.type
  if (!(mime.includes('jpeg') || mime.includes('png') || mime.includes('gif'))) {
    session.flashNotice = '投稿できる画像形式はjpgとpngとgifだけです'
    return c.redirect('/')
  }
  if (file.size > UPLOAD_LIMIT) {
    session.flashNotice = 'ファイルサイズが大きすぎます'
    return c.redirect('/')
  }
  const buffer = Buffer.from(await file.arrayBuffer())
  const [result] = await db.query<ResultSetHeader>('INSERT INTO `posts` (`user_id`, `mime`, `imgdata`, `body`) VALUES (?,?,?,?)', [me.id, mime, buffer, String(body.body)])
  const insertId = result.insertId
  return c.redirect(`/posts/${encodeURIComponent(String(insertId))}`)
})

app.get('/image/:filename{[0-9]+\\.(png|jpg|gif)}', async (c) => {
  try {
    const [id, ext] = c.req.param('filename').split('.')
    const [posts] = await db.query<RowDataPacket[]>('SELECT * FROM `posts` WHERE `id` = ?', [id])
    const post = (posts as Post[])[0]
    if (!post) return c.text('image not found', 404)
    if ((ext === 'jpg' && post.mime === 'image/jpeg') || (ext === 'png' && post.mime === 'image/png') || (ext === 'gif' && post.mime === 'image/gif')) {
      return new Response(new Uint8Array(post.imgdata), { headers: { 'Content-Type': post.mime } })
    }
  } catch (e) {
    console.error(e)
    return c.text(String(e), 500)
  }
})

app.post('/comment', async (c) => {
  const me = await getSessionUser(c)
  if (!me) return c.redirect('/login')
  const body = await c.req.parseBody()
  const session = c.get('session') as SessionData
  if (body.csrf_token !== session.csrfToken) return c.text('invalid CSRF Token', 422)
  if (!body.post_id || !/^[0-9]+$/.test(String(body.post_id))) return c.text('post_idは整数のみです')
  await db.query<ResultSetHeader>('INSERT INTO `comments` (`post_id`, `user_id`, `comment`) VALUES (?,?,?)', [Number(body.post_id), me.id, String(body.comment || '')])
  return c.redirect(`/posts/${encodeURIComponent(String(body.post_id))}`)
})

app.get('/admin/banned', async (c) => {
  const me = await getSessionUser(c)
  if (!me) return c.redirect('/login')
  if (me.authority === 0) return c.text('authority is required', 403)
  const [userRows] = await db.query<RowDataPacket[]>('SELECT * FROM `users` WHERE `authority` = 0 AND `del_flg` = 0 ORDER BY `created_at` DESC')
  const users = userRows as User[]
  return render(c, 'banned.ejs', { me, users })
})

app.post('/admin/banned', async (c) => {
  const me = await getSessionUser(c)
  if (!me) return c.redirect('/')
  if (me.authority === 0) return c.text('authority is required', 403)
  const body = await c.req.parseBody()
  const session = c.get('session') as SessionData
  if (body.csrf_token !== session.csrfToken) return c.text('invalid CSRF Token', 422)
  const query = 'UPDATE `users` SET `del_flg` = ? WHERE `id` = ?'
  const ids: string[] = Array.isArray(body.uid) ? body.uid as string[] : [String(body.uid)]
  await Promise.all(ids.map((id) => db.query<ResultSetHeader>(query, [1, id])))
  return c.redirect('/admin/banned')
})

app.use('/*', serveStatic({ root: '../public' }))

serve(
  { fetch: app.fetch, port: 8080 },
  info => {
    console.log(`server started on ${info.port}`)
  }
)
