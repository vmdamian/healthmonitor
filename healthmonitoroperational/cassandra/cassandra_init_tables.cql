CREATE KEYSPACE IF NOT EXISTS healthmonitor
  WITH REPLICATION = {
   'class' : 'SimpleStrategy',
   'replication_factor' : 1
  };

CREATE TABLE IF NOT EXISTS healthmonitor.users (
  id text PRIMARY KEY,
  password text
  );

INSERT INTO healthmonitor.users (id, password)
VALUES ('damian', '123456')
IF NOT EXISTS;