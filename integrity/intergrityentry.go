package integrity

import "time"

var DATE_UNDEFINED time.Time = time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)

type IntegrityEntry struct {
	FilePath    string
	FileSize    int64
	FileModTime time.Time
	DateChecked *time.Time
}
