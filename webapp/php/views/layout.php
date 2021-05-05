<!DOCTYPE html>
<html>
  <head>
    <meta charset="utf-8">
    <title>Iscogram</title>
    <link href="/css/style.css" media="screen" rel="stylesheet" type="text/css">
  </head>
  <body>
    <div class="container">
      <?php require __DIR__ . '/header.php' ?>
      <?php require __DIR__ . '/' . $view ?>
    </div>
    <script src="/js/timeago.min.js"></script>
    <script src="/js/main.js"></script>
  </body>
</html>
