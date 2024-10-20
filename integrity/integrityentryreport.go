package integrity

import (
	"encoding/csv"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

const FILE_PATH string = "File"
const FILE_SIZE string = "Size"
const FILE_MODTIME string = "Mod-Time"
const DATE_CHECKED string = "Last-Check"

const DATE_FORMAT string = "2006-01-02 15:04:05.999999"

type IntegrityEntryReport struct {
	FilePath  string
	Separator rune
	headers   []string
}

func (ier *IntegrityEntryReport) GetIntegrityEntries() []IntegrityEntry {

	index_FilePath, index_FileSize, index_FileModTime, index_DateChecked := -1, -1, -1, -1

	datas := ier.readRows()
	results := make([]IntegrityEntry, 0, len(*datas))

	for i := range *datas {
		row := (*datas)[i]
		if i == 0 {
			ier.headers = row
			for j := range row {
				switch row[j] {
				case FILE_PATH:
					index_FilePath = j
				case FILE_SIZE:
					index_FileSize = j
				case FILE_MODTIME:
					index_FileModTime = j
				case DATE_CHECKED:
					index_DateChecked = j
				}
			}
		} else {
			filePath := row[index_FilePath]
			fileSize, _ := strconv.ParseInt(row[index_FileSize], 10, 64)
			fileModTime, _ := parseUnixUTC(row[index_FileModTime])
			dateChecked, _ := time.Parse(DATE_FORMAT, row[index_DateChecked])

			ie := IntegrityEntry{
				FilePath:    filePath,
				FileSize:    fileSize,
				FileModTime: *fileModTime,
				DateChecked: &dateChecked,
			}
			results = append(results, ie)
		}
	}
	return results
}

func (ier *IntegrityEntryReport) SetIntegrityEntries(datas []IntegrityEntry) {

	records := make([][]string, 0, len(datas)+1)

	records = append(records, ier.headers)

	for i := range datas {
		var record []string
		record = append(record, datas[i].FilePath)
		record = append(record, strconv.FormatInt(datas[i].FileSize, 10))
		record = append(record, strUnixUTC(datas[i].FileModTime))

		date_checked_string := (*datas[i].DateChecked).Format(DATE_FORMAT)
		parts := strings.SplitN(date_checked_string, ".", 2)
		if len(parts) == 1 {
			date_checked_string = date_checked_string + ".000000"
		}
		record = append(record, date_checked_string)

		records = append(records, record)
	}

	ier.writeRows(&records)
}

func (ier *IntegrityEntryReport) readRows() *[][]string {

	var results *[][]string

	_, err := os.Stat(ier.FilePath)
	if !errors.Is(err, os.ErrExist) {
		fd, err := os.Open(ier.FilePath)
		if err != nil {
			fmt.Println(err)
		}

		reader := csv.NewReader(fd)
		reader.Comma = ier.Separator

		records, err := reader.ReadAll()
		if err != nil {
			fmt.Println(err)
		}
		results = &records

		fd.Close()
	}

	return results
}

func (ier *IntegrityEntryReport) writeRows(records *[][]string) {

	fd, err := os.Create(ier.FilePath)
	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}
	writer := csv.NewWriter(fd)
	writer.Comma = ier.Separator

	for _, record := range *records {
		err := writer.Write(record)
		if err != nil {
			log.Fatalln("error writing record to file", err)
		}
	}

	writer.Flush()
	fd.Close()
}

func parseUnixUTC(p_param string) (*time.Time, error) {

	if len(p_param) < 20 {
		p_param = p_param + strings.Repeat("0", 20-len(p_param))
	}
	parts := strings.SplitN(p_param, ".", 2)

	sec, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		fmt.Println("Error converting whole part:", err)
	}

	nsec, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		fmt.Println("Error converting fractional part:", err)
	}

	result := time.Unix(sec, nsec)

	return &result, nil
}

func strUnixUTC(p_param time.Time) string {
	epochref := time.Unix(0, 0)
	difftime := p_param.Sub(epochref).Nanoseconds()

	s2 := fmt.Sprintf("%d", difftime)
	s2 = s2[:10] + "." + s2[10:]

	return s2
}
