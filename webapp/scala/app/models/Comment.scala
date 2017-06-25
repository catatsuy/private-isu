package models

import org.joda.time.DateTime
import scalikejdbc._

case class Comment(
  id: Int,
  postId: Int,
  userId: Int,
  comment: String,
  createdAt: DateTime
)

case class CommentResult(
  comment: Comment,
  user: User
)

object Comment extends SQLSyntaxSupport[Comment] {
  override val tableName: String = "comments"
  val c: SyntaxProvider[Comment] = this.syntax("c")

  def apply(rs: WrappedResultSet): Comment = autoConstruct[Comment](rs, c.resultName)
}
