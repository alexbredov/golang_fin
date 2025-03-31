-- +goose Up

CREATE TABLE whitelist (
                           id SERIAL PRIMARY KEY,
                           mask INT NOT NULL,
                           IP inet NOT NULL
);

CREATE TABLE blacklist (
                           id SERIAL PRIMARY KEY,
                           mask INT NOT NULL,
                           IP inet NOT NULL
);
-- +goose Down
DROP TABLE whitelist;
DROP TABLE blacklist;