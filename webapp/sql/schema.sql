-- DROP TABLE IF EXISTS users;
CREATE TABLE IF NOT EXISTS users (
  `id` int NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `account_name` varchar(64) NOT NULL UNIQUE,
  `passhash` varchar(128) NOT NULL, -- SHA2 512 non-binary (hex)
  `authority` tinyint(1) NOT NULL DEFAULT 0,
  `del_flg` tinyint(1) NOT NULL DEFAULT 0,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
) DEFAULT CHARSET=utf8mb4;

-- DROP TABLE IF EXISTS posts;
CREATE TABLE IF NOT EXISTS posts (
  `id` int NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `user_id` int NOT NULL,
  `mime` varchar(64) NOT NULL,
  `imgdata` mediumblob NOT NULL,
  `body` text NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
) DEFAULT CHARSET=utf8mb4;

-- DROP TABLE IF EXISTS comments;
CREATE TABLE IF NOT EXISTS comments (
  `id` int NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `post_id` int NOT NULL,
  `user_id` int NOT NULL,
  `comment` text NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
) DEFAULT CHARSET=utf8mb4;
