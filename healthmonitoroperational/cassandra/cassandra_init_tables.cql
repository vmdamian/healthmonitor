CREATE KEYSPACE IF NOT EXISTS healthmonitor
  WITH REPLICATION = {
   'class' : 'SimpleStrategy',
   'replication_factor' : 1
};

CREATE TABLE IF NOT EXISTS healthmonitor.users (
  id text PRIMARY KEY,
  password text,
  user_devices set<text>
);

CREATE TABLE IF NOT EXISTS healthmonitor.tokens (
    id text PRIMARY KEY,
    user_token text
);