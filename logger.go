package loggerN

/*
Author: chenwenjiang
Email: senchen2013@gmail.com
Create date: 2015-04-29
*/

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	debugLevel = iota
	infoLevel
	noticeLevel
	warningLevel
	errorLevel
)

const (
	nocolor = 0
	red     = 30 + iota
	green
	yellow
	blue
	purple
	cyan
)

var (
	logPrefix = map[int]string{
		debugLevel:   "DEBUG",
		infoLevel:    "INFO",
		noticeLevel:  "NOTICE",
		warningLevel: "WARNING",
		errorLevel:   "ERROR",
	}
	logColor = map[int]int{
		debugLevel:   cyan,
		infoLevel:    nocolor,
		noticeLevel:  green,
		warningLevel: yellow,
		errorLevel:   red,
	}

	LogStr2Int = map[string]int{
		"DEBUG":   debugLevel,
		"INFO":    infoLevel,
		"NOTICE":  noticeLevel,
		"WARNING": warningLevel,
		"ERROR":   errorLevel,
	}
	Levels = map[int]bool{}
	mtime  string
)

type FileStat struct {
	name   string
	fmtime int64
}
type FSTS []FileStat

func (fs FSTS) Len() int           { return len(fs) }
func (fs FSTS) Swap(i, j int)      { fs[i], fs[j] = fs[j], fs[i] }
func (fs FSTS) Less(i, j int) bool { return fs[i].fmtime > fs[j].fmtime }

type Logger struct {
	sync.Mutex
	isConsole      bool
	isColorfull    bool
	reserveCounter int
	timeFormat     string
	fileName       string
	fileWriter     io.WriteCloser
}

// 滚动切割文件
func rollingLogFile(toFileName string, logger *Logger) {
	logger.Lock()
	defer logger.Unlock()

	logger.fileWriter.Close()
	logger.fileWriter = nil
	err := os.Rename(logger.fileName, toFileName)
	if err != nil {
		panic(err)
	}
	fileWriter, err := os.OpenFile(logger.fileName, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	logger.fileWriter = fileWriter
}

// log split checker
func logSplitChecker(logger *Logger) {
	for {
		if mtime == "" {
			mtime = time.Now().Format(logger.timeFormat)
			continue
		}
		time.Sleep(time.Second * 1)
		currentTime := time.Now().Format(logger.timeFormat)
		if currentTime != mtime {
			toFileName := fmt.Sprintf("%s.%s", logger.fileName, mtime)
			rollingLogFile(toFileName, logger)
			mtime = currentTime
		}
	}
}

func logCounterChecker(logger *Logger) {

	for {
		dirname := filepath.Dir(logger.fileName)
		basename := filepath.Base(logger.fileName)
		logLists := make(FSTS, 0)
		filepath.Walk(fmt.Sprintf("%s/", dirname), func(path string, f os.FileInfo, err error) error {
			if strings.HasPrefix(f.Name(), fmt.Sprintf("%s.", basename)) {
				fs := FileStat{
					name:   f.Name(),
					fmtime: f.ModTime().Unix(),
				}
				logLists = append(logLists, fs)
			}
			return nil
		})
		sort.Sort(logLists)

		if len(logLists) > logger.reserveCounter {
			removes := logLists[logger.reserveCounter:]
			for _, fname := range removes {
				rmname := fmt.Sprintf("%s/%s", dirname, fname.name)
				os.Remove(rmname)
			}
		}
		time.Sleep(time.Second * 10)
	}
}

// 设置日志级别
func setLevels(levels []string) {
	Levels = make(map[int]bool)

	if len(levels) > 0 {
		flag := false
		for _, v := range levels {
			if v == "NOSET" { // 表示所有的log都大于
				flag = true
			}
		}
		if !flag {
			for _, v := range levels {
				if sv, ok := LogStr2Int[v]; ok {
					Levels[sv] = true
				}
			}
		}
	}
}

func NewLogger(loggerConfigFile string) *Logger {
	config, err := NewConfig(loggerConfigFile)
	if err != nil {
		panic(err)
	}
	logLevel := config.MustValue("golog", "log.level", "INFO")
	logDir := config.MustValue("golog", "log.dir", "./")
	logFile := config.MustValue("golog", "log.file", "file.log")
	logReserveCounter := config.MustInt("golog", "log.reserve.counter", 5)
	logSuffix := config.MustValue("golog", "log.suffix", "2006010215")
	logConsole := config.MustInt("golog", "log.console", 0)
	logColorfull := config.MustInt("golog", "log.colorfull", 1)

	logFilePath := fmt.Sprintf("%s/%s", logDir, logFile)
	fileWriter := getFileWriter(logFilePath)
	setLevels(strings.Split(logLevel, ","))

	boolConsole := false
	if logConsole == 1 {
		boolConsole = true
	}
	boolColorfull := false
	if logColorfull == 1 {
		boolColorfull = true
	}

	loggerHandle := &Logger{
		isConsole:      boolConsole,
		isColorfull:    boolColorfull,
		reserveCounter: logReserveCounter,
		timeFormat:     logSuffix,
		fileName:       logFilePath,
		fileWriter:     fileWriter,
	}
	go logSplitChecker(loggerHandle)
	go logCounterChecker(loggerHandle)
	return loggerHandle
}

// 创建默认的logger log的配置在pwd
func NewDefaultLogger() *Logger {
	file, _ := os.Getwd()
	logFile := fmt.Sprintf("%s/%s", file, "golog.cfg")
	return NewLogger(logFile)
}

// 新建console的日志handle
func NewConsoleLogger() *Logger {
	return nil
}

// 创建文件
func getFileWriter(fileName string) io.WriteCloser {
	fileWriter, err := os.OpenFile(fileName, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	return fileWriter
}

func getDetail() (string, string, int) {
	// 获取调用函数的数据
	pc, file, line, _ := runtime.Caller(3)
	fc := runtime.FuncForPC(pc).Name()
	parts := strings.Split(fc, ".")
	filename := path.Base(file)
	return fmt.Sprintf("%s/%s", parts[0], filename), parts[1], line
}

func (l *Logger) write(level int, format string, content ...interface{}) {
	filename, fc, line := getDetail() // 获取文件的信息
	if len(Levels) > 0 {
		if _, ok := Levels[level]; !ok {
			return
		}
	}

	now := time.Now()

	var s string
	if format == "" {
		s = renderColor(fmt.Sprintf("[%s]\t[%s]\t[%s]\t[%s:%d]\t%s\n", now.Format("2006/01/02 15:04:05"), logPrefix[level], filename, fc, line, fmt.Sprint(content...)), logColor[level], l.isColorfull)
	} else {
		s = renderColor(fmt.Sprintf("[%s]\t[%s]\t[%s]\t[%s:%d]\t%s\n", now.Format("2006/01/02 15:04:05"), logPrefix[level], filename, fc, line, fmt.Sprintf(format, content...)), logColor[level], l.isColorfull)
	}

	l.Lock()
	defer l.Unlock()
	l.fileWriter.Write([]byte(s))
	if l.isConsole {
		fmt.Print(s)
	}
}

func (l *Logger) Info(content ...interface{}) {
	l.write(infoLevel, "", content...)
}

func (l *Logger) Infof(format string, content ...interface{}) {
	l.write(infoLevel, format, content...)
}

func (l *Logger) Warning(content ...interface{}) {
	l.write(warningLevel, "", content...)
}

func (l *Logger) Warningf(format string, content ...interface{}) {
	l.write(warningLevel, format, content...)
}

func (l *Logger) Notice(content ...interface{}) {
	l.write(noticeLevel, "", content...)
}

func (l *Logger) Noticef(format string, content ...interface{}) {
	l.write(noticeLevel, format, content...)
}

func (l *Logger) Debug(content ...interface{}) {
	l.write(debugLevel, "", content...)
}

func (l *Logger) Debugf(format string, content ...interface{}) {
	l.write(debugLevel, format, content...)
}

func (l *Logger) Error(content ...interface{}) {
	l.write(errorLevel, "", content...)
}

func (l *Logger) Errorf(format string, content ...interface{}) {
	l.write(errorLevel, format, content...)
}

func renderColor(s string, color int, isColorfull bool) string {
	if isColorfull {
		return fmt.Sprintf("\033[%dm%s\033[0m", color, s)
	} else {
		return s
	}
}
