-- +goose Up

CREATE TABLE whitelist
    {
    id INT PRIMARY KEY AUTO_INCREMENT,
    mask INT NOT NULL,
    IP varchar(20) NOT NULL
    };

CREATE TABLE blacklist
    {
    id INT PRIMARY KEY AUTO_INCREMENT,
    mask INT NOT NULL,
    IP varchar(20) NOT NULL
    };

-- +goose Down
DROP TABLE whitelist;
DROP TABLE blacklist;