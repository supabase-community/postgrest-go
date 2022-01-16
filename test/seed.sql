CREATE TABLE
    IF NOT EXISTS users (id serial PRIMARY KEY, name text, email text UNIQUE);

INSERT INTO
    users (name, email)
VALUES ('sean', 'sean@test.com'), ('patti', 'patti@test.com')
    ON CONFLICT DO NOTHING;