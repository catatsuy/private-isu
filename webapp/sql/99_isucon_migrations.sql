ALTER TABLE comments ADD INDEX idx_post_id (post_id);
ALTER TABLE posts ADD INDEX idx_created_at (created_at);
ALTER TABLE comments ADD INDEX idx_user_id (user_id);
ALTER TABLE posts ADD INDEX idx_user_created (user_id, created_at);
