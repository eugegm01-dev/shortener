ALTER TABLE urls
ADD COLUMN user_id VARCHAR(50);

CREATE INDEX idx_user_id ON urls(user_id);