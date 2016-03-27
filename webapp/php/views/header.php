<div class="header">
  <div class="isu-title">
    <h1><a href="/">Iscogram</a></h1>
  </div>
  <div class="isu-header-menu">
    <?php if (!isset($me)): ?>
    <div><a href="/login">ログイン</a></div>
    <?php else: ?>
    <div><a href="/@<?= escape_html(rawurlencode($me['account_name'])) ?>"><span class="isu-account-name"><?= escape_html($me['account_name']) ?></span>さん</a></div>
    <?php if ($me['authority'] == 1): ?>
    <div><a href="/admin/banned">管理者用ページ</a></div>
    <?php endif ?>
    <div><a href="/logout">ログアウト</a></div>
    <?php endif ?>
  </div>
</div>
