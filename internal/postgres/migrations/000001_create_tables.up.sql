CREATE TABLE gauge (
                       id VARCHAR(256) PRIMARY KEY,
                       value DOUBLE PRECISION NOT NULL
);

CREATE TABLE counter (
                         id VARCHAR(256) PRIMARY KEY,
                         value BIGINT NOT NULL
);