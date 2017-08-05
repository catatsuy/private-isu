name := "private-isu"
organization := "jp.ne.opt"
version := "1.0-SNAPSHOT"

lazy val root = (project in file(".")).enablePlugins(PlayScala)

scalaVersion := "2.12.2"

libraryDependencies ++= Seq(
  guice,
  "org.scalikejdbc" %% "scalikejdbc" % "3.0.0",
  "org.scalikejdbc" %% "scalikejdbc-config" % "3.0.0",
  "org.scalikejdbc" %% "scalikejdbc-syntax-support-macro" % "3.0.0",
  "mysql" % "mysql-connector-java" % "5.1.40",
  "org.scalatestplus.play" %% "scalatestplus-play" % "3.0.0" % Test
)

TwirlKeys.templateImports += "helpers._"
