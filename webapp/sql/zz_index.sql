-- 計測 #01（BOTTLENECK_RESULT_01.md）で判明したボトルネックへの対処。
-- docker-entrypoint-initdb.d は名前順に実行されるため、dump.sql.bz2 の後に流れるよう zz_ で始めている。
USE `isuconp`;

-- DB 実行時間の 97.3% が comments への `WHERE post_id = ?` のフルスキャン（1回あたり 97,660 行走査）だった。
--   SELECT * FROM comments WHERE post_id = ? ORDER BY created_at DESC LIMIT 3  … 65.3%
--   SELECT COUNT(*) FROM comments WHERE post_id = ?                            … 25.9%
--   SELECT * FROM comments WHERE post_id = ? ORDER BY created_at DESC          …  6.1%
-- created_at を第2キーに含めることで ORDER BY のソートも省ける。
ALTER TABLE comments ADD INDEX idx_post_id_created_at (post_id, created_at);
