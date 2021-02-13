--- name: CreateVisit :exec
INSERT INTO visit(visit_id, geo)
VALUES (pggen.arg('visit_id'), pggen.arg('geo'));

--- name: FindVisit :one
SELECT visit_id, geo
FROM visit
WHERE visit_id = pggen.arg('visit_id')
LIMIT 1;
