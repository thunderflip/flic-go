package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"time"
	
	"github.com/thunderflip/flic-go/integrity"
)

var EXIT_CODE_OK = 0
var EXIT_CODE_ERR_OPTION = -1
var EXIT_CODE_ERR_VALIDATION = -2

var PERCENTAGE_THRESHOLD_MIN string = "MIN"
var PERCENTAGE_THRESHOLD_MAX string = "MAX"

var MODTIME_TOLERANCE time.Duration = time.Millisecond
var MINUTES_BETWEEN_AUTO_SAVE time.Duration = 3

func intersect(ier_list []integrity.IntegrityEntry, ied_list []integrity.IntegrityEntry) []integrity.IntegrityEntry {

	results := make([]integrity.IntegrityEntry, 0, max(len(ier_list), len(ied_list)))

	ier_dict := make(map[string]integrity.IntegrityEntry, len(ier_list))
	for _, ie := range ier_list {
		ier_dict[ie.FilePath] = ie
	}

	for _, ied := range ied_list {
		ie_new := ied
		ier, exists := ier_dict[ied.FilePath]
		if exists {
			if ier.FileSize == ied.FileSize {
				diff_time := ier.FileModTime.Sub(ied.FileModTime)
				if diff_time < 0 {
					diff_time = ied.FileModTime.Sub(ier.FileModTime)
				}
				if diff_time <= MODTIME_TOLERANCE {
					ie_new = ier
					if diff_time != 0 {
						// Force time read from disk
						ie_new.FileModTime = ied.FileModTime
					}
				}
			}
		}
		results = append(results, ie_new)
	}

	return results
}

func main() {

	var flac_path string
	var folder string
	var report_file string
	var p_age int
	var p_percentage_min int
	var p_percentage_max int

	var age *int
	var percentage *int
	var percentage_threshold *string

	flag.StringVar(&flac_path, "flac", "", "Path to the flac executable")
	flag.StringVar(&folder, "folder", "", "Root folder path to FLAC collection for recursive files search")
	flag.StringVar(&report_file, "report", "", "Path to the report file")
	flag.IntVar(&p_age, "age", math.MinInt32, "Age in minutes to identify files to check")
	flag.IntVar(&p_percentage_min, "min-percentage", math.MinInt32, "Minimum percentage of collection to check")
	flag.IntVar(&p_percentage_max, "max-percentage", math.MinInt32, "Maximum percentage of collection to check")
	flag.Parse()

	if flac_path == "" {
		log.Println("'flac' argument is mandatory")
		os.Exit(EXIT_CODE_ERR_OPTION)
	}

	if folder == "" {
		log.Println("'folder' argument is mandatory")
		os.Exit(EXIT_CODE_ERR_OPTION)
	}

	if report_file == "" {
		log.Println("'report' argument is mandatory")
		os.Exit(EXIT_CODE_ERR_OPTION)
	}

	if p_percentage_min != math.MinInt32 && p_percentage_max != math.MinInt32 {
		log.Println("'xxx-percentage' argument has been provided more than once")
		os.Exit(EXIT_CODE_ERR_OPTION)
	}

	if p_percentage_min != math.MinInt32 {
		percentage = &p_percentage_min
		percentage_threshold = &PERCENTAGE_THRESHOLD_MIN
	}

	if p_percentage_max != math.MinInt32 {
		percentage = &p_percentage_max
		percentage_threshold = &PERCENTAGE_THRESHOLD_MAX
	}

	if p_age != math.MinInt32 {
		age = &p_age
	}

	check(flac_path, folder, report_file, age, percentage, percentage_threshold)
	os.Exit(EXIT_CODE_OK)
}

func check(flac_path string, folder string, report_file string, age *int, percentage *int, percentage_threshold *string) {

	log.Println("BEG - Check")
	date_begin := time.Now()

	ier := integrity.IntegrityEntryReport{
		FilePath:  report_file,
		Separator: ';',
	}
	ier_list := ier.GetIntegrityEntries()

	ied := integrity.IntegrityEntryDirectory{
		DirPath: folder,
	}
	ied_list := ied.GetIntegrityEntries()

	integrity_entries := intersect(ier_list, ied_list)
	sort.Slice(integrity_entries, func(i, j int) bool {
		return integrity_entries[i].DateChecked.Before(*integrity_entries[j].DateChecked)
	})

	if len(integrity_entries) <= 0 {
		log.Println("No item, nothing will be done")
	} else {
		log.Println("Total item(s): " + strconv.Itoa(len(integrity_entries)))

		checked_date_oldest := integrity_entries[0].DateChecked
		checked_date_newest := integrity_entries[len(integrity_entries)-1].DateChecked
		log.Println(checked_date_oldest)
		log.Println(checked_date_newest)

		var limit_by_age *int
		if age != nil {
			limit_age_date := time.Now()
			if *age >= 0 {
				now := time.Now()
				minutes, _ := time.ParseDuration(strconv.Itoa(int(*age)) + "m")
				limit_age_date = now.Add(-1 * minutes)
			} else if *age == -1 {
				limit_age_date = integrity.DATE_UNDEFINED
			} else if *age == -2 {
				limit_age_date = time.Now().Truncate(24 * time.Hour)
			}

			index_first_newer := sort.Search(len(integrity_entries), func(i int) bool {
				return limit_age_date.Before(*integrity_entries[i].DateChecked)
			})

			limit_by_age = &index_first_newer
			log.Println("Limit item(s) by age: " + strconv.Itoa(*limit_by_age))
		} else {
			log.Println("Limit item(s) by age: not defined")
		}

		var limit_by_percentage *int
		if percentage != nil && *percentage > 0 {
			n := (float64(len(integrity_entries)) / float64(100) * float64(*percentage))
			nb_entries := int(math.Round(n))
			nb_entries = min(nb_entries, len(integrity_entries))
			limit_by_percentage = &nb_entries
			log.Println("Limit item(s) by percentage: " + strconv.Itoa(*limit_by_percentage) + " " + *percentage_threshold)
		}

		limit := 0
		if limit_by_age != nil {
			limit = *limit_by_age
			if limit_by_percentage != nil {
				if *percentage_threshold == PERCENTAGE_THRESHOLD_MIN {
					if *limit_by_age < *limit_by_percentage {
						limit = *limit_by_percentage
						log.Println("Limit item 'by age' changed by limit 'by percentage' from " + strconv.Itoa(*limit_by_age) + " to " + strconv.Itoa(*limit_by_percentage))
					}
				} else if *percentage_threshold == PERCENTAGE_THRESHOLD_MAX {
					if *limit_by_age > *limit_by_percentage {
						limit = *limit_by_percentage
						log.Println("Limit item 'by age' changed by limit 'by percentage' from " + strconv.Itoa(*limit_by_age) + " to " + strconv.Itoa(*limit_by_percentage))
					}
				}
			}
		} else if limit_by_percentage != nil {
			if *percentage_threshold == PERCENTAGE_THRESHOLD_MIN {
				limit = *limit_by_percentage
			}
		}
		log.Println("Effective item(s) limit: " + strconv.Itoa(limit))

		if limit >= len(integrity_entries) {
			// Everything will be checked: change check order to filePath
			sort.Slice(integrity_entries, func(i, j int) bool {
				return integrity_entries[i].FilePath < integrity_entries[j].FilePath
			})
		}

		last_save := time.Now()
		number_format := "% " + strconv.Itoa(len(strconv.Itoa(limit))) + "d"		

		for i := range integrity_entries {
			if i < limit {
				_, err := os.Stat(integrity_entries[i].FilePath)
				if !errors.Is(err, os.ErrNotExist) {
					flac_op := FlacOperation{
						flac_path:     flac_path,
						file_path:     integrity_entries[i].FilePath,
					}

					cur_index := fmt.Sprintf(number_format, i+1)
					cur_percentage := fmt.Sprintf("%6.2f", float64(i+1)/float64(limit)*float64(100))
					cur_file := integrity_entries[i].FilePath[:10] + "..." + integrity_entries[i].FilePath[len(integrity_entries[i].FilePath)-50:]
					if true || len(integrity_entries[i].FilePath) < len(cur_file) {
						cur_file = integrity_entries[i].FilePath
					}
					log.Println("Verifying (" + cur_index + "/" + strconv.Itoa(limit) + " - " + cur_percentage + "%) " + cur_file)

					successful, err := flac_op.Test()
					if successful && err == nil {
						now := time.Now()
						integrity_entries[i].DateChecked = &now
					} else {
						log.Fatalf("KO")
						os.Exit(EXIT_CODE_ERR_VALIDATION)
					}
				}

				now := time.Now()
				if now.Sub(last_save) > (time.Minute * MINUTES_BETWEEN_AUTO_SAVE) {
					ier.SetIntegrityEntries(integrity_entries)
					last_save = now
				}
			} else {
				log.Println("There are no more items satisfying 'age' or 'percentage' conditions")
				break
			}
		}

		sort.Slice(integrity_entries, func(i, j int) bool {
			return integrity_entries[i].DateChecked.Before(*integrity_entries[j].DateChecked)
		})

		ier.SetIntegrityEntries(integrity_entries)

		date_end := time.Now()
		log.Println("Elapsed time:", date_end.Sub(date_begin), "for "+strconv.Itoa(limit)+" item(s)")
		log.Println("END - Check")
	}
}
