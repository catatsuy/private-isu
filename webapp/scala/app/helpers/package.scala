import models._
import org.joda.time.DateTime
import org.joda.time.format.ISODateTimeFormat

package object helpers {

  def imageUrl(post: Post): String = {
    val ext = post.mime match {
      case "image/jpeg" => ".jpg"
      case "image/png" => ".png"
      case "image/gif" => ".gif"
      case _ => ""
    }

    s"/image/${post.id}${ext}"
  }

  implicit class RichDateTime(val self: DateTime) extends AnyVal {

    def iso8601: String = {
      val parser = ISODateTimeFormat.dateTimeNoMillis()
      parser.print(self)
    }
  }

  implicit class RichString(val self: String) extends AnyVal {

    def urlEncode: String = {
      java.net.URLEncoder.encode(self, "UTF-8")
    }
  }
}
