package controllers

import java.nio.file.Files
import java.security.SecureRandom
import javax.inject._

import models._
import org.joda.time.format.ISODateTimeFormat
import play.api.data.Form
import play.api.data.Forms._
import play.api.mvc._
import services.DBService

import sys.process._
import scalikejdbc._

@Singleton
class IsuController @Inject()(db: DBService, cc: ControllerComponents) extends AbstractController(cc) {

  private[this] val uploadLimit = 10 * 1024 * 1024 // 10mb
  private[this] val postsPerPage = 20
  private[this] val rand = SecureRandom.getInstance("NativePRNGNonBlocking")

  private[this] def getSessionUser(implicit request: Request[AnyContent]): Option[User] = {
    import User.u

    db.readOnly { implicit session =>
      request.session.get("user").flatMap { id =>
        sql"SELECT ${u.resultAll} FROM ${User as u} WHERE ${u.id} = ${id.toLong}".map(User(_)).first().apply()
      }
    }
  }

  private[this] def digest(src: String): String = {
    (Process("printf", Seq("%s", src)) #|
      Process("openssl", Seq("dgst", "-sha512"))).!!.trim.replaceAll("^.*= ", "")
  }

  private[this] def calculateSalt(accountName: String): String = {
    digest(accountName)
  }

  private[this] def calculatePasshash(accountName: String, password: String): String = {
    digest(s"${password}:${calculateSalt(accountName)}")
  }

  private[this] def makePostResults(posts: Seq[Post], allComments: Boolean = false): Seq[PostResult] = {
    import Comment.c, User.u

    def loop(current: Seq[Post], acc: Vector[PostResult], count: Int)(implicit session: DBSession): Vector[PostResult] = current match {

      case head +: tail =>
        if (count >= postsPerPage) {
          acc
        } else {
          val commentCount = sql"SELECT COUNT(*) AS `count` FROM `comments` WHERE `post_id` = ${head.id}".map(_.int(1)).first().apply().getOrElse(0)

          val comments = sql"SELECT ${c.resultAll} FROM ${Comment as c} WHERE ${c.postId} = ${head.id} ORDER BY ${c.createdAt} DESC ${if (allComments) sqls.empty else sqls"LIMIT 3"}".map(Comment(_)).list().apply()
          val commentResults = comments.map { comment =>
            val Some(user) = sql"SELECT ${u.resultAll} FROM ${User as u} WHERE ${u.id} = ${comment.userId}".map(User(_)).first().apply()
            CommentResult(comment, user)
          }.reverse

          val Some(user) = sql"SELECT ${u.resultAll} FROM ${User as u} WHERE ${u.id} = ${head.userId}".map(User(_)).first().apply()
          if (!user.delFlg) {
            loop(tail, acc :+ PostResult(head, commentCount, commentResults, user), count + 1)
          } else {
            loop(tail, acc, count)
          }
        }
      case _ =>
        acc
    }

    db.readOnly { implicit session =>
      loop(posts, Vector.empty, 0)
    }
  }

  def index() = Action { implicit request: Request[AnyContent] =>
    import Post.p

    db.readOnly { implicit session =>
      val posts = sql"SELECT ${p.result.id}, ${p.result.userId}, ${p.result.body}, ${p.result.createdAt}, ${p.result.mime} FROM ${Post as p} ORDER BY ${p.createdAt} DESC"
        .map(Post.withoutImage).list().apply()
      Ok(views.html.index(getSessionUser, makePostResults(posts)))
    }
  }

  def initialize() = Action { implicit request: Request[AnyContent] =>
    db.autoCommit { implicit session =>
      sql"DELETE FROM users WHERE id > 1000".execute().apply()
      sql"DELETE FROM posts WHERE id > 10000".execute().apply()
      sql"DELETE FROM comments WHERE id > 100000".execute().apply()
      sql"UPDATE users SET del_flg = 0".execute().apply()
      sql"UPDATE users SET del_flg = 1 WHERE id % 50 = 0".execute().apply()
    }
    Ok
  }

  def showLogin() = Action { implicit request: Request[AnyContent] =>
    getSessionUser match {
      case Some(_) => Redirect(routes.IsuController.index())
      case _ => Ok(views.html.login(None))
    }
  }

  case class LoginData(accountName: String, password: String)
  val loginForm = Form(mapping("account_name" -> text, "password" -> text)(LoginData.apply)(LoginData.unapply))
  def newLogin() = Action { implicit request: Request[AnyContent] =>
    getSessionUser match {
      case Some(_) => Redirect(routes.IsuController.index())
      case _ =>
        val LoginData(accountName, password) = loginForm.bindFromRequest().get
        import User.u

        db.readOnly { implicit session =>
          sql"SELECT ${u.resultAll} FROM ${User as u} WHERE ${u.accountName} = ${accountName} AND ${u.delFlg} = 0".map(User(_)).first().apply() match {
            case Some(user) if calculatePasshash(user.accountName, password) == user.passhash =>
              Redirect(routes.IsuController.index())
                .withSession("user" -> user.id.toString, "csrf_token" -> BigInt(128, rand).toString(16))
            case _ =>
              Redirect(routes.IsuController.showLogin())
                .flashing("notice" -> "アカウント名かパスワードが間違っています")
          }
        }
    }
  }

  def showRegister() = Action { implicit request: Request[AnyContent] =>
    getSessionUser match {
      case Some(_) => Redirect(routes.IsuController.index())
      case _ => Ok(views.html.register(None))
    }
  }

  case class RegistrationData(accountName: String, password: String)
  val registrationForm = Form(mapping(
    "account_name" -> text.verifying(_.matches("""\A[0-9a-zA-Z_]{3,}\z""")),
    "password" -> text.verifying(_.matches("""\A[0-9a-zA-Z_]{6,}\z"""))
  )(RegistrationData.apply)(RegistrationData.unapply))
  def register() = Action { implicit request: Request[AnyContent] =>
    db.autoCommit { implicit session =>
      getSessionUser match {
        case Some(_) => Redirect(routes.IsuController.index())
        case _ =>
          val data = registrationForm.bindFromRequest()

          if (data.hasErrors) {
            Redirect(routes.IsuController.showRegister())
              .flashing("notice" -> "アカウント名は3文字以上、パスワードは6文字以上である必要があります")
          } else if (sql"SELECT 1 FROM users WHERE `account_name` = ${data.get.accountName}"
            .map(_.int(1)).first().apply().nonEmpty) {
            Redirect(routes.IsuController.showRegister())
              .flashing("notice" -> "アカウント名がすでに使われています")
          } else {
            val passhash = calculatePasshash(data.get.accountName, data.get.password)
            val id =
              sql"INSERT INTO `users` (`account_name`, `passhash`) VALUES (${data.get.accountName}, ${passhash})".updateAndReturnGeneratedKey().apply()

            Redirect(routes.IsuController.index())
              .withSession("user" -> id.toString, "csrf_token" -> BigInt(16, rand).toString(16))
          }
      }
    }
  }

  def logout() = Action { implicit request: Request[AnyContent] =>
    Redirect(routes.IsuController.index())
      .removingFromSession("user")
  }

  def showAccount(accountName: String) = Action { implicit request: Request[AnyContent] =>
    db.readOnly { implicit session =>
      import User.u, Post.p

      sql"SELECT ${u.resultAll} FROM ${User as u} WHERE ${u.accountName} = ${accountName} AND ${u.delFlg} = 0".map(User(_)).first().apply() match {
        case None =>
          NotFound
        case Some(user) =>
          val userId = user.id
          val posts = sql"SELECT ${p.result.id}, ${p.result.userId}, ${p.result.body}, ${p.result.createdAt}, ${p.result.mime} FROM ${Post as p} WHERE ${p.userId} = ${userId} ORDER BY ${p.createdAt} DESC"
            .map(Post.withoutImage).list().apply()
          val postResults = makePostResults(posts)

          val commentCount = sql"SELECT COUNT(*) AS count FROM `comments` WHERE `user_id` = ${userId}".map(_.int(1)).single().apply().getOrElse(0)

          val postIds = sql"SELECT `id` FROM `posts` WHERE `user_id` = ${userId}".map(_.int(1)).list().apply()

          val postCount = postIds.size

          val commentedCount = if (postCount > 0) {
            sql"SELECT COUNT(*) AS count FROM `comments` WHERE `post_id` IN (${postIds})".map(_.int(1)).first().apply().getOrElse(0)
          } else {
            0
          }

          Ok(views.html.user(getSessionUser, user, postResults, postCount, commentCount, commentedCount))
      }
    }
  }

  def posts() = Action { implicit request: Request[AnyContent] =>
    import Post.p

    val maxCreatedAt = request.getQueryString("max_created_at") match {
      case Some(param) =>
        ISODateTimeFormat.dateTimeNoMillis().parseDateTime(param).toLocalDateTime
      case _ =>
        null
    }

    val posts = db.readOnly { implicit session =>
      sql"SELECT ${p.result.id}, ${p.result.userId}, ${p.result.body}, ${p.result.createdAt}, ${p.result.mime} FROM ${Post as p} WHERE ${p.createdAt} <= ${maxCreatedAt} ORDER BY ${p.createdAt} DESC"
        .map(Post.withoutImage).list().apply()
    }

    Ok(views.html.posts(makePostResults(posts)))
  }

  def showPost(id: Int) = Action { implicit request: Request[AnyContent] =>
    import Post.p

    db.readOnly { implicit session =>
      val posts = sql"SELECT ${p.resultAll} FROM ${Post as p} WHERE ${p.id} = ${id}".map(Post(_)).list().apply()
      val postResults = makePostResults(posts, allComments = true)

      postResults match {
        case head +: _ => Ok(views.html.main(getSessionUser)(views.html.post(head)))
        case _ => NotFound
      }
    }
  }

  def createPost() = Action { implicit request: Request[AnyContent] =>
    getSessionUser match {
      case None =>
        Redirect(routes.IsuController.showLogin())
      case Some(me) =>
        val Some(body) = request.body.asMultipartFormData

        body.dataParts("csrf_token") match {
          case token +: _ if token == request.session("csrf_token") =>
            body.file("file") match {
              case Some(file) if !file.contentType.exists(t => t.contains("jpeg") || t.contains("png") || t.contains("gif")) =>
                Redirect(routes.IsuController.index())
                  .flashing("notice" -> "投稿できる画像形式はjpgとpngとgifだけです")
              case Some(file) if file.ref.length() > uploadLimit =>
                Redirect(routes.IsuController.index())
                  .flashing("notice" -> "ファイルサイズが大きすぎます")
              case Some(file) =>
                val Some(contentType) = file.contentType
                val mime = if (contentType.contains("jpeg")) {
                  "image/jpeg"
                } else if (contentType.contains("png")) {
                  "image/png"
                } else if (contentType.contains("gif")) {
                  "image/gif"
                } else { "" }
                val postBody +: _ = body.dataParts("body")

                val tmpFile = java.io.File.createTempFile("isu", ".tmp")
                try {
                  file.ref.moveTo(tmpFile, replace = true)
                  val bytes = Files.readAllBytes(tmpFile.toPath)
                  val id = db.autoCommit { implicit session =>
                    sql"INSERT INTO `posts` (`user_id`, `mime`, `imgdata`, `body`) VALUES (${me.id}, ${mime}, ${bytes}, ${postBody})".updateAndReturnGeneratedKey().apply()
                  }
                  Redirect(routes.IsuController.showPost(id.toInt))
                } finally {
                  tmpFile.delete()
                }
              case _ =>
                Redirect(routes.IsuController.index())
                  .flashing("notice" -> "画像が必須です")
            }
          case _ =>
            UnprocessableEntity
        }
    }
  }

  def showImage(id: Int, ext: String) = Action { implicit request: Request[AnyContent] =>
    import Post.p

    if (id == 0) {
      Ok("")
    } else {
      db.readOnly { implicit session =>
        sql"SELECT ${p.resultAll} FROM ${Post as p} WHERE ${p.id} = ${id}".map(Post(_)).first().apply() match {
          case None =>
            NotFound
          case Some(post) =>
            if ((ext, post.mime) match {
              case ("jpg", "image/jpeg") => true
              case ("png", "image/png") => true
              case ("gif", "image/gif") => true
              case _ => false
            }) {
              Ok(post.imgdata).as(post.mime)
            } else {
              NotFound
            }
        }
      }

    }
  }

  def createComment = Action { implicit request: Request[AnyContent] =>
    getSessionUser match {
      case None =>
        Redirect(routes.IsuController.showLogin())
      case Some(me) =>
        val Some(body) = request.body.asFormUrlEncoded

      body("csrf_token") match {
        case token +: _ if token == request.session("csrf_token") =>
          body("post_id") match {
            case head +: _ if head.matches("""\A[0-9]+\z""") =>
              val comment = body("comment").head
              val postId = head.toInt
              db.autoCommit { implicit session =>
                sql"INSERT INTO `comments` (`post_id`, `user_id`, `comment`) VALUES (${postId}, ${me.id}, ${comment})".execute().apply()
              }

              Redirect(routes.IsuController.showPost(postId))
            case _ =>
              // Rubyに合わせてるが、これ200でいいなの？
              Ok("post_idは整数のみです")
          }
        case _ =>
          UnprocessableEntity
      }
    }
  }

  def banned() = Action { implicit request: Request[AnyContent] =>
    import User.u

    getSessionUser match {
      case None =>
        Redirect(routes.IsuController.index())
      case Some(me) if !me.authority =>
        Forbidden
      case Some(me) =>
        db.readOnly { implicit session =>
          val users = sql"SELECT ${u.resultAll} FROM ${User as u} WHERE ${u.authority} = 0 AND ${u.delFlg} = 0 ORDER BY ${u.createdAt} DESC".map(User(_)).list().apply()
          Ok(views.html.banned(Some(me), users))
        }
    }
  }

  def ban() = Action { implicit request: Request[AnyContent] =>
    getSessionUser match {
      case None =>
        Redirect(routes.IsuController.index())
      case Some(me) if !me.authority =>
        Forbidden
      case Some(me) =>
        val Some(body) = request.body.asFormUrlEncoded

        body("csrf_token") match {
          case token +: _ if token == request.session("csrf_token") =>
            db.autoCommit { implicit session =>
              body("uid").foreach { id =>
                sql"UPDATE `users` SET `del_flg` = 1 WHERE `id` = ${id.toInt}".executeUpdate().apply()
              }
            }
            Redirect(routes.IsuController.banned())
          case _ =>
            UnprocessableEntity
        }
    }
  }
}
