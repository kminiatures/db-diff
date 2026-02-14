-- Migration script to create differences for testing
-- Run this script manually to modify the database, then create a second snapshot

-- Add a new column to users table
ALTER TABLE users ADD COLUMN phone VARCHAR(20);

-- Modify existing data
UPDATE users SET email = 'alice.new@example.com' WHERE username = 'alice';
UPDATE users SET age = 26 WHERE username = 'alice';

-- Add new user
INSERT INTO users (username, email, age, phone) VALUES
    ('dave', 'dave@example.com', 28, '+1234567890');

-- Delete a user (this will cascade delete related posts and comments)
DELETE FROM users WHERE username = 'charlie';

-- Add new post
INSERT INTO posts (user_id, title, content, published) VALUES
    (1, 'Updated Post', 'Alice''s new post after migration', TRUE);

-- Update post
UPDATE posts SET published = TRUE WHERE id = 2;

-- Create a new table
CREATE TABLE tags (
    id INT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(50) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO tags (name) VALUES ('tech'), ('lifestyle'), ('news');

-- Add index to posts
CREATE INDEX idx_published ON posts(published);
