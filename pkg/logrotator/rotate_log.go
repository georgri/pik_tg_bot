package logrotator

import (
	"bytes"
	"context"
	"fmt"
	"github.com/georgri/pik_tg_bot/pkg/flatstorage"
	"github.com/georgri/pik_tg_bot/pkg/util"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"sync"
	"time"
)

const (
	LogFolder   = "./logs"
	LogFile     = "bot.log"
	MaxLogFiles = 48

	TryRotateEvery = 5 * time.Minute
)

var LogFileRegexp = regexp.MustCompile(`^bot-.*-.*\.log\.gz$`)

var logMutex sync.Mutex
var logFile *os.File

func ArchiveLogFile(targetFileName, archiveName string) error {

	origFile := fmt.Sprintf("%v/%v", LogFolder, LogFile)

	// first, rename the log file
	err := os.Rename(origFile, targetFileName)
	if err != nil {
		return fmt.Errorf("unable to rename log file: %v", err)
	}

	// start writing to the new log file
	err = SetupLogging()
	if err != nil {
		return fmt.Errorf("unable to setup new log file: %v", err)
	}

	// tar + gzip new file
	var buf bytes.Buffer
	err = util.Compress(targetFileName, &buf)
	if err != nil {
		return err
	}

	// write the .gz
	archiveFilename := archiveName
	err = os.MkdirAll(filepath.Dir(archiveFilename), os.FileMode(0777))
	if err != nil {
		return err
	}

	fileToWrite, err := os.OpenFile(archiveFilename, os.O_CREATE|os.O_RDWR, os.FileMode(0666))
	if err != nil {
		return err
	}
	if _, err = io.Copy(fileToWrite, &buf); err != nil {
		return err
	}

	err = os.Remove(targetFileName)
	if err != nil {
		return fmt.Errorf("unable to remove old log file: %v", err)
	}

	return nil
}

func GetLogFileList() ([]string, error) {
	entries, err := os.ReadDir(LogFolder)
	if err != nil {
		return nil, err
	}

	var filenames []string

	for _, e := range entries {
		filename := e.Name()
		if !LogFileRegexp.MatchString(filename) {
			continue
		}
		filenames = append(filenames, fmt.Sprintf("%v/%v", LogFolder, filename))
	}

	sort.Slice(filenames, func(i, j int) bool {
		return filenames[i] > filenames[j]
	})

	return filenames, err
}

func DeleteExtraLogFiles() error {
	filenames, err := GetLogFileList()
	if err != nil {
		return err
	}

	if len(filenames) <= MaxLogFiles {
		return nil
	}

	for _, filename := range filenames[MaxLogFiles:] {
		err = os.Remove(filename)
		if err != nil {
			return err
		}
	}

	return nil
}

func RotateLogsForever(ctx context.Context, wg *sync.WaitGroup) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		err := RotateLogsOnce(wg)
		if err != nil {
			log.Printf("error while rotating logs: %v", err)
		}
		time.Sleep(TryRotateEvery)
	}
}

func GetCurrentLogFile() string {
	yesterday := time.Now().AddDate(0, 0, -1)
	hostname, _ := os.Hostname()
	return fmt.Sprintf("%v/bot-%v-%v.log", LogFolder, hostname, yesterday.Format(time.DateOnly))
}

func RotateLogsOnce(wg *sync.WaitGroup) error {
	wg.Add(1)
	defer wg.Done()

	// check if log file with yesterday's name exists
	logFileName := GetCurrentLogFile()
	archiveName := logFileName + ".gz"
	if flatstorage.FileExistsNonBlocking(archiveName) {
		return nil
	}

	err := ArchiveLogFile(logFileName, archiveName)
	if err != nil {
		return err
	}

	err = DeleteExtraLogFiles()
	if err != nil {
		return err
	}

	err = SendLastLogFile(archiveName)
	if err != nil {
		return err
	}

	log.Printf("sucessfully rotated log into %v", archiveName)

	return nil
}

func SetupLogging() error {

	// open or create default log file
	logFilename := fmt.Sprintf("%v/%v", LogFolder, LogFile)
	f, err := os.OpenFile(logFilename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("error opening log file: %v", err)
	}

	log.SetOutput(f)

	logMutex.Lock()
	defer logMutex.Unlock()

	oldfile := logFile
	logFile = f

	// close old log file if existed
	if oldfile != nil {
		err := oldfile.Close()
		if err != nil {
			log.Printf("error while trying to close old log file: %v", err)
		}
	}

	return nil
}

func CloseLog() {
	log.SetOutput(os.Stderr)
	if logFile != nil {
		err := logFile.Close()
		if err != nil {
			log.Fatalf("failed to close log file")
		}
	}
	return
}
