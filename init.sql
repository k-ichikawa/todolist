CREATE DATABASE IF NOT EXISTS todo_app
    CHARACTER SET utf8mb4
    COLLATE utf8mb4_unicode_ci;

USE todo_app;

CREATE TABLE IF NOT EXISTS todos (
    id INT AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    completed BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

INSERT INTO todos (title, completed) VALUES ('Goを学ぶ', FALSE);
INSERT INTO todos (title, completed) VALUES ('Reactを学ぶ', FALSE);
INSERT INTO todos (title, completed) VALUES ('Dockerアプリを構築する', FALSE);