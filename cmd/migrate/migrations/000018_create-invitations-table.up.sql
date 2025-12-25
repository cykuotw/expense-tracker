CREATE TABLE invitations(
    id serial PRIMARY KEY,
    token varchar(255) NOT NULL UNIQUE,
    email varchar(255) NOT NULL,
    inviter_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at timestamp NOT NULL,
    used_at timestamp,
    created_at timestamp DEFAULT NOW()
);

