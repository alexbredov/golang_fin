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