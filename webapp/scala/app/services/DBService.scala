package services

import javax.inject._

import scalikejdbc._
import scalikejdbc.config.DBs

@Singleton
class DBService {

  if (!ConnectionPool.isInitialized()) {
    DBs.setup()
  }

  def readOnly[A](f: DBSession => A): A = {
    DB.readOnly(f)
  }

  def localTx[A](f: DBSession => A): A = {
    DB.localTx(f)
  }

  def autoCommit[A](f: DBSession => A): A = {
    DB.autoCommit(f)
  }
}
