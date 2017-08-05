package models

import org.joda.time.DateTime
import scalikejdbc._

case class Post(
  id: Int,
  userId: Int,
  mime: String,
  imgdata: Array[Byte],
  body: String,
  createdAt: DateTime
)

case class PostResult(
  post: Post,
  commentCount: Int,
  comments: Seq[CommentResult],
  user: User
)

object Post extends SQLSyntaxSupport[Post] {
  override val tableName: String = "posts"
  val p: SyntaxProvider[Post] = this.syntax("p")

  def apply(rs: WrappedResultSet): Post = autoConstruct[Post](rs, p.resultName)

  def withoutImage(rs: WrappedResultSet): Post = Post(
    id = rs.get(p.resultName.id),
    userId = rs.get(p.resultName.userId),
    body = rs.get(p.resultName.body),
    imgdata = Array.empty,
    mime = rs.get(p.resultName.mime),
    createdAt = rs.get(p.resultName.createdAt)
  )
}
