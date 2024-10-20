package main

import (
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strings"
)

// https://xiph.org/flac/documentation_tools_metaflac.html
// https://xiph.org/flac/documentation_tools_flac.html
// https://github.com/xiph/flac/blob/9b3826006a3fc27b34d9297a9a8194accacc2c44/src/flac/main.c


type FlacOperation struct {
	flac_path string
	metaflac_path string
	file_path string
}

func (f *FlacOperation) GetHash() (string, error) {
	arg := []string{"--show-md5sum", f.file_path}
	output, err := exec.Command(f.metaflac_path, arg...).Output()
	if err != nil {
		log.Printf("Error running metaflac: %v", err)
		return "", err
	}

	match := regexp.MustCompile(`(^\S*)\S+.*`).FindStringSubmatch(string(output))
	if match == nil || len(match) != 2 {
		log.Printf("Failed to parse metaflac output: %s", string(output))
		return "", fmt.Errorf("failed to parse metaflac output")
	}

	return match[1], nil
}

func (f *FlacOperation) Reencode() (bool, error) {
	args := []string{"--force", "--no-error-on-compression-fail", "--verify", f.file_path}
	out, err := exec.Command(f.flac_path, args...).CombinedOutput()
	if err != nil {
		log.Printf("Error running flac: %v", err)
		return false, err
	}

	output := string(out)
	if strings.TrimSpace(output) == "" {
		log.Println("FLAC output expected")
		return false, fmt.Errorf("FLAC output is empty")
	}

	// Search for 'Verify OK' pattern
	matched := regexp.MustCompile(`.*Verify OK,.*`).MatchString(output)
	if !matched {
		log.Println("FLAC verification failed:\n%s", output)
		return false, fmt.Errorf("FLAC verification failed")
	}

	log.Println("FLAC reencode successful")
	return true, nil
}

func (f *FlacOperation) Test() (bool, error) {
	args := []string{"--test", f.file_path}
	out, err := exec.Command(f.flac_path, args...).CombinedOutput()
	if err != nil {
		log.Printf("Error running flac: %v", err)
		return false, err
	}

	output := string(out)
	if strings.TrimSpace(output) == "" {
		log.Println("FLAC output expected")
		return false, fmt.Errorf("FLAC output is empty")
	}

	// Search for '*ok' at the end of each line
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if matched := regexp.MustCompile(`.*ok`).MatchString(line); matched {
			//log.Println("FLAC verification successful")
			return true, nil
		}
	}

	log.Println("FLAC verification failed:\n%s", output)
	return false, fmt.Errorf("FLAC verification failed")
}