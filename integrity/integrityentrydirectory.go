package integrity

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type IntegrityEntryDirectory struct {
	DirPath string
}

func (ied IntegrityEntryDirectory) GetIntegrityEntries() []IntegrityEntry {

	var results []IntegrityEntry

	err := filepath.WalkDir(ied.DirPath, func(filePath string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}

		if strings.ToLower(filepath.Ext(d.Name())) == ".flac" {
			fileInfo, err := os.Stat(filePath)
			if err != nil {
				fmt.Println("Error getting file info:", err)
			}

			ie := IntegrityEntry{
				FilePath:    filePath,
				FileSize:    fileInfo.Size(),
				FileModTime: fileInfo.ModTime(),
				DateChecked: &DATE_UNDEFINED,
			}

			results = append(results, ie)
		}
		return nil
	})

	if err != nil {
		log.Fatalf("impossible to walk directories: %s", err)
	}

	return results
}
