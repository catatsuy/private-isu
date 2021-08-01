<div class="isu-submit">
  <form method="post" action="/" enctype="multipart/form-data">
    <div class="isu-form">
      <input type="file" name="file" value="file">
    </div>
    <div class="isu-form">
      <textarea name="body"></textarea>
    </div>
    <div class="form-submit">
      <input type="hidden" name="csrf_token" value="<?= escape_html(session_id()); ?>">
      <input type="submit" name="submit" value="submit">
    </div>
    <?php if ($flash): ?>
    <div id="notice-message" class="alert alert-danger">
      <?= escape_html($flash); ?>
    </div>
    <?php endif ?>
  </form>
</div>

<?php require __DIR__ . '/posts.php' ?>

<div id="isu-post-more">
  <button id="isu-post-more-btn">もっと見る</button>
  <img class="isu-loading-icon" src="/img/ajax-loader.gif">
</div>
