package discovery

import "Postwoman/internal/importexport"

// importexportParse isolates the dependency on the import parser so the
// generated collection is validated by the exact code path used when a user
// imports a file.
func importexportParse(content string) (*importexport.Parsed, error) {
	return importexport.Parse([]byte(content), "postwoman-collection.json")
}
