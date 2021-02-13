CREATE EXTENSION IF NOT EXISTS postgis;

CREATE TABLE visit (
  visit_id int4 not null,
  geo      public.geography
);
