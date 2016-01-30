-- DROP TABLE IF EXISTS users;
CREATE TABLE IF NOT EXISTS users (
  `id` int NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `account_name` varchar(64) NOT NULL UNIQUE,
  `email` varchar(255) CHARACTER SET utf8 NOT NULL UNIQUE,
  `passhash` varchar(128) NOT NULL, -- SHA2 512 non-binary (hex)
  `salt` varchar(255) NOT NULL,
  `del_flg` tinyint(1) NOT NULL DEFAULT 0
) DEFAULT CHARSET=utf8mb4;

