CREATE TABLE author (
  author_id  serial PRIMARY KEY,
  first_name text NOT NULL,
  last_name  text NOT NULL,
  suffix text NULL
);
