-- name: CreateUser :exec
INSERT INTO "user" (email, pass)
VALUES (pggen.arg('email'), crypt(pggen.arg('password'), gen_salt('bf')));

-- name: FindUser :one
SELECT email, pass from "user"
where email = pggen.arg('email');
