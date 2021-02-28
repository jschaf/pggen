-- name: FindTopScienceChildren :many
SELECT path
FROM test
WHERE path <@ 'Top.Science';

-- name: FindTopScienceChildrenAgg :one
SELECT array_agg(path)
FROM test
WHERE path <@ 'Top.Science';

-- name: InsertSampleData :exec
INSERT INTO test
VALUES ('Top'),
       ('Top.Science'),
       ('Top.Science.Astronomy'),
       ('Top.Science.Astronomy.Astrophysics'),
       ('Top.Science.Astronomy.Cosmology'),
       ('Top.Hobbies'),
       ('Top.Hobbies.Amateurs_Astronomy'),
       ('Top.Collections'),
       ('Top.Collections.Pictures'),
       ('Top.Collections.Pictures.Astronomy'),
       ('Top.Collections.Pictures.Astronomy.Stars'),
       ('Top.Collections.Pictures.Astronomy.Galaxies'),
       ('Top.Collections.Pictures.Astronomy.Astronauts');

-- name: FindLtreeInput :one
SELECT
  pggen.arg('in_ltree')::ltree                   AS ltree,
  -- This won't work, but I'm not quite sure why.
  -- Postgres errors with "wrong element type (SQLSTATE 42804)"
  -- All caps because we use regex to find pggen.arg and it confuses pggen.
  -- PGGEN.arg('in_ltree_array_direct')::ltree[]    AS direct_arr,

  -- The parenthesis around the text[] cast are important. They signal to pggen
  -- that we need a text array that Postgres then converts to ltree[].
  (pggen.arg('in_ltree_array')::text[])::ltree[] AS text_arr;