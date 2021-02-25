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