CREATE INDEX idx_users_account_name ON users (account_name);
CREATE INDEX idx_posts_created_at ON posts (created_at);
CREATE INDEX idx_posts_user_id ON posts (user_id);
CREATE INDEX idx_comments_user_id ON comments (user_id);
CREATE INDEX idx_comments_post_id ON comments (post_id);