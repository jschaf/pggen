-- Query to test escaping in generated Go.
-- name: Backtick :one
SELECT '`';

-- Query to test escaping in generated Go.
-- name: BacktickQuoteBacktick :one
SELECT '`"`';

-- Query to test escaping in generated Go.
-- name: BacktickNewline :one
SELECT '`
';

-- Query to test escaping in generated Go.
-- name: BacktickDoubleQuote :one
SELECT '`"';

-- Query to test escaping in generated Go.
-- name: BacktickBackslashN :one
SELECT '`\n';

-- Illegal names.
-- name: IllegalNameSymbols :one
SELECT '`\n' as "$", pggen.arg('@hello world!') as "foo.bar!@#$%&*()""--+";
