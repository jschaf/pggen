package pgdocker

type pgTemplate struct {
	PGPass    string
	InitFiles []string
}

const dockerfileTemplate = `
{{- /*gotype: github.com/jschaf/pggen/internal/pgdocker.pgTemplate*/ -}}
{{- define "dockerfile" -}}
FROM postgres:13

{{ range .InitFiles }}
COPY {{.}} /docker-entrypoint-initdb.d/
{{ end }}
{{ end }}
`
