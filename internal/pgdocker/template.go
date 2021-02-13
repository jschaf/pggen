package pgdocker

type pgTemplate struct {
	PGPass      string
	InitScripts []string
}

const dockerfileTemplate = `
{{- /*gotype: github.com/jschaf/pggen/internal/pgdocker.pgTemplate*/ -}}
{{- define "dockerfile" -}}
FROM postgis/postgis
{{ range .InitScripts }}
COPY {{.}} /docker-entrypoint-initdb.d/
{{ end }}
{{- end }}
`
