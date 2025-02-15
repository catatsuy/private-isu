-- Create "comments" table
CREATE TABLE `comments` (`id` int NOT NULL AUTO_INCREMENT, `post_id` int NOT NULL, `user_id` int NOT NULL, `comment` text NOT NULL, `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP, PRIMARY KEY (`id`)) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
-- Create "posts" table
CREATE TABLE `posts` (`id` int NOT NULL AUTO_INCREMENT, `user_id` int NOT NULL, `mime` varchar(64) NOT NULL, `imgdata` mediumblob NOT NULL, `body` text NOT NULL, `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP, PRIMARY KEY (`id`)) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
-- Create "users" table
CREATE TABLE `users` (`id` int NOT NULL AUTO_INCREMENT, `account_name` varchar(64) NOT NULL, `passhash` varchar(128) NOT NULL, `authority` bool NOT NULL DEFAULT 0, `del_flg` bool NOT NULL DEFAULT 0, `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP, PRIMARY KEY (`id`), UNIQUE INDEX `account_name` (`account_name`)) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
