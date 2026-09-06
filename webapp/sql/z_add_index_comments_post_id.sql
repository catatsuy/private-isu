-- comments.post_id にインデックスを追加する
--
-- 背景:
--   posts 詳細・タイムラインで `WHERE post_id = ?` / `WHERE post_id IN (...)`
--   が多用され、フルスキャンになっていた。pt-query-digest で
--   `WHERE post_id = ?` 系が平均 35-41us / Rows examine 3前後まで改善した。
--
-- 流し方:
--   - Docker Compose 初回起動時は /docker-entrypoint-initdb.d 経由で
--     `dump.sql.bz2` の直後に自動適用される
--     (ファイル名の頭を `z_` にして dump より後にソートされるようにしている)。
--   - 稼働中DBへ手動適用する場合 (mysql: root/root, DB名 isuconp):
--     mysql -h 127.0.0.1 -P 3306 -u root -proot isuconp < webapp/sql/z_add_index_comments_post_id.sql
--
--   # 確認
--   mysql -h 127.0.0.1 -P 3306 -u root -proot isuconp -e "SHOW INDEX FROM comments;"
--   mysql -h 127.0.0.1 -P 3306 -u root -proot isuconp -e "EXPLAIN SELECT * FROM comments WHERE post_id = 1 ORDER BY created_at DESC;"
--
-- 注意:
--   - `webapp/sql/dump.sql.bz2` は外部リリース由来の初期データのため本PRに含めない。
--   - 新規に dump を作り直す場合は `benchmarker/sql/schema.sql` 側に同定義済みのため不要。
--   - ロールバック: `DROP INDEX idx_comments_post_id ON comments;`

-- docker-entrypoint-initdb.d 経由では選択中DBがないため明示する。
USE `isuconp`;

CREATE INDEX `idx_comments_post_id` ON `comments` (`post_id`);
