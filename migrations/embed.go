package migrations

import "embed"

// FS содержит все SQL-файлы миграций, встроенные в бинарник.
//
//go:embed *.sql
var FS embed.FS
