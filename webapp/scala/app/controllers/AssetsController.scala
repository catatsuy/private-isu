package controllers

import java.io._
import javax.inject.Inject

import play.api._
import play.api.http.FileMimeTypes
import play.api.mvc._

import scala.concurrent.{ ExecutionContext, Future }

/**
 * Stolen from controllers.ExternalAssets
 */
class AssetsController @Inject() (environment: Environment)(implicit ec: ExecutionContext, fileMimeTypes: FileMimeTypes)
  extends ControllerHelpers {

  private val rootPath = sys.props.getOrElse("isu.public.dir", "/home/isucon/private_isu/webapp/public")

  private val Action = new ActionBuilder.IgnoringBody()(_root_.controllers.Execution.trampoline)

  def at(file: String): Action[AnyContent] = Action.async { request =>
    Future {

      val fileToServe = new File(rootPath, file)

      if (fileToServe.exists) {
        Ok.sendFile(fileToServe, inline = true).withHeaders(CACHE_CONTROL -> "max-age=3600")
      } else {
        NotFound
      }

    }
  }

}
