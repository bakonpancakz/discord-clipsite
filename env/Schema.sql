-- All changes to this schema file should be incremental
-- as the program will execute this query on startup!

-- Version 1.0 - Initial Release
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS users (
    id                  TEXT        NOT NULL UNIQUE,                    -- Discord User ID
    created             TEXT        NOT NULL DEFAULT CURRENT_TIMESTAMP, -- Created At
    avatar              TEXT,                                           -- Cached Avatar Hash
    name                TEXT,                                           -- Cached Username/Displayname
    token               TEXT        UNIQUE                              -- Current Session Token
);

CREATE TABLE IF NOT EXISTS videos (
    id                  TEXT        NOT NULL UNIQUE,                    -- Upload ID
    created             TEXT        NOT NULL DEFAULT CURRENT_TIMESTAMP, -- Created At
    user_id             TEXT        NOT NULL,                           -- Relevant User ID
    status              TEXT        NOT NULL CHECK(status IN ('QUEUE', 'PROCESS', 'ERROR', 'FINISH')),
    FOREIGN KEY (user_id) REFERENCES users(id)
);