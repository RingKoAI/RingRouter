// Package handler: dbutil.go holds small query-shaping helpers shared by the
// CRUD handlers.
package handler

import "strings"

// escapeLike neutralizes SQL LIKE wildcards inside user-supplied search
// terms. Values are already bound as parameters (no injection risk); this
// keeps a literal "%" or "_" in the query from silently widening the filter.
// The escape character itself is escaped first.
func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

// likePattern wraps a search term as a bounded LIKE pattern with escaped
// wildcards.
func likePattern(q string) string {
	return "%" + escapeLike(q) + "%"
}
