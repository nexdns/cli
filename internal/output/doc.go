// Package output renders API values in the format the user selected.
//
// Formatter is the common interface; New returns the implementation for a
// format string: TableFormatter (human-readable, optionally colorised),
// JSONFormatter, YAMLFormatter, or CSVFormatter. Commands obtain a Formatter
// from the cmd package and hand it typed api values (zones, records, DNSSEC
// status, account, API keys); the formatter alone decides layout. Output is
// written to the supplied io.Writer (normally stdout). The quiet flag suppresses
// summary lines and compacts JSON; the color mode (auto/always/never, plus
// NO_COLOR) controls table colourisation.
package output
