-- Initial test data
INSERT INTO users (username, email, age) VALUES
    ('alice', 'alice@example.com', 25),
    ('bob', 'bob@example.com', 30),
    ('charlie', 'charlie@example.com', 35);

INSERT INTO posts (user_id, title, content, published) VALUES
    (1, 'First Post', 'This is Alice''s first post', TRUE),
    (1, 'Second Post', 'Another post by Alice', FALSE),
    (2, 'Bob''s Post', 'Bob''s thoughts', TRUE),
    (3, 'Charlie''s Post', 'Hello world', TRUE);

INSERT INTO comments (post_id, user_id, comment) VALUES
    (1, 2, 'Great post, Alice!'),
    (1, 3, 'I agree with Bob'),
    (3, 1, 'Nice post, Bob!'),
    (4, 2, 'Welcome, Charlie!');
