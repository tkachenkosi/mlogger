// Package mlogger - минималистичный логгер для Go проектов
package mlogger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// ==================== НАСТРОЙКИ (измените под свои нужды) ====================

const (
	// Режим вывода: "file", "console", "both"
	LOG_OUTPUT = "console"

	// Путь к файлу логов (используется при LOG_OUTPUT = "file" или "both")
	LOG_FILE_PATH = "logs/app.log"

	// Уровень логирования: "debug", "info", "warn", "error"
	LOG_LEVEL = "debug"

	// Формат времени: "datetime" (2024-01-15 10:30:45) или "time" (10:30:45)
	LOG_TIME_FORMAT = "datetime"

	// Показывать имя файла и номер строки
	LOG_SHOW_CALLER = true

	// Цветной вывод в консоль
	LOG_COLOR_CONSOLE = true

	// Размер буфера для асинхронной записи (0 - синхронная запись)
	LOG_BUFFER_SIZE = 100 // количество сообщений в буфере

	// Максимальный размер файла логов в МБ (0 - без ограничения)
	LOG_MAX_SIZE_MB = 10

	// Максимальное количество старых файлов логов (0 - не удалять)
	LOG_MAX_BACKUPS = 3
)

// ==================== ВНУТРЕННЯЯ РЕАЛИЗАЦИЯ ====================

type LogLevel int

const (
	LEVEL_DEBUG LogLevel = iota
	LEVEL_INFO
	LEVEL_WARN
	LEVEL_ERROR
)

var levelNames = map[LogLevel]string{
	LEVEL_DEBUG: "DEBUG",
	LEVEL_INFO:  "INFO",
	LEVEL_WARN:  "WARN",
	LEVEL_ERROR: "ERROR",
}

var levelColors = map[LogLevel]string{
	LEVEL_DEBUG: "\033[36m", // циан
	LEVEL_INFO:  "\033[32m", // зеленый
	LEVEL_WARN:  "\033[33m", // желтый
	LEVEL_ERROR: "\033[31m", // красный
}

var levelPriority = map[string]LogLevel{
	"debug": LEVEL_DEBUG,
	"info":  LEVEL_INFO,
	"warn":  LEVEL_WARN,
	"error": LEVEL_ERROR,
}

type Logger struct {
	mu         sync.Mutex
	file       *os.File
	consoleOut io.Writer
	logLevel   LogLevel
	buffer     chan string
	wg         sync.WaitGroup
	stopCh     chan struct{}
}

var (
	instance *Logger
	once     sync.Once
)

// getLevelPriority возвращает числовой уровень логирования
func getLevelPriority(level string) LogLevel {
	if lvl, ok := levelPriority[strings.ToLower(level)]; ok {
		return lvl
	}
	return LEVEL_INFO
}

// getTimeString возвращает отформатированное время
func getTimeString() string {
	now := time.Now()
	if LOG_TIME_FORMAT == "time" {
		return now.Format("15:04:05")
	}
	return now.Format("2006-01-02 15:04:05")
}

// getCallerInfo возвращает имя файла и номер строки
func getCallerInfo() string {
	if !LOG_SHOW_CALLER {
		return ""
	}

	_, file, line, ok := runtime.Caller(3) // 3 уровень вверх по стеку
	if !ok {
		return ""
	}

	// Берем только имя файла без пути
	file = filepath.Base(file)
	return fmt.Sprintf(" %s:%d", file, line)
}

// getColor возвращает цвет для уровня
func getColor(level LogLevel) string {
	if LOG_COLOR_CONSOLE {
		return levelColors[level]
	}
	return ""
}

// resetColor возвращает сброс цвета
func resetColor() string {
	if LOG_COLOR_CONSOLE {
		return "\033[0m"
	}
	return ""
}

// formatMessage форматирует сообщение
func formatMessage(level LogLevel, format string, args ...any) string {
	timeStr := getTimeString()
	caller := getCallerInfo()

	msg := fmt.Sprintf(format, args...)

	return fmt.Sprintf("[%s] [%s]%s %s",
		timeStr,
		levelNames[level],
		caller,
		msg,
	)
}

// writeToFile записывает сообщение в файл с ротацией
func (l *Logger) writeToFile(msg string) error {
	if l.file == nil {
		return nil
	}

	// Проверка размера файла
	if LOG_MAX_SIZE_MB > 0 {
		fileInfo, err := l.file.Stat()
		if err == nil && fileInfo.Size() > int64(LOG_MAX_SIZE_MB*1024*1024) {
			l.rotateFile()
		}
	}

	_, err := l.file.WriteString(msg + "\n")
	return err
}

// rotateFile выполняет ротацию файла логов
func (l *Logger) rotateFile() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		l.file.Close()
	}

	// Переименовываем текущий файл
	if LOG_MAX_BACKUPS > 0 {
		timestamp := time.Now().Format("20060102_150405")
		backupName := fmt.Sprintf("%s.%s", LOG_FILE_PATH, timestamp)
		os.Rename(LOG_FILE_PATH, backupName)

		// Удаляем старые файлы
		l.cleanOldLogs()
	}

	// Создаем новый файл
	os.MkdirAll(filepath.Dir(LOG_FILE_PATH), 0o755)
	l.file, _ = os.OpenFile(LOG_FILE_PATH, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
}

// cleanOldLogs удаляет старые файлы логов
func (l *Logger) cleanOldLogs() {
	files, err := filepath.Glob(LOG_FILE_PATH + ".*")
	if err != nil || len(files) <= LOG_MAX_BACKUPS {
		return
	}

	// Сортируем по времени создания
	type fileInfo struct {
		name string
		time time.Time
	}

	var infos []fileInfo
	for _, file := range files {
		info, err := os.Stat(file)
		if err == nil {
			infos = append(infos, fileInfo{file, info.ModTime()})
		}
	}

	// Удаляем самые старые
	if len(infos) > LOG_MAX_BACKUPS {
		for i := 0; i < len(infos)-LOG_MAX_BACKUPS; i++ {
			os.Remove(infos[i].name)
		}
	}
}

// write сообщение во все выходные потоки
func (l *Logger) write(level LogLevel, color string, msg string) {
	// Проверка уровня
	if level < l.logLevel {
		return
	}

	finalMsg := msg

	// Добавляем цвет для консоли
	consoleMsg := fmt.Sprintf("%s%s%s", color, msg, resetColor())

	// Запись в зависимости от режима
	switch LOG_OUTPUT {
	case "file":
		if l.buffer != nil {
			select {
			case l.buffer <- finalMsg:
			default:
				// Буфер полон, пишем синхронно
				l.writeToFile(finalMsg)
			}
		} else {
			l.writeToFile(finalMsg)
		}

	case "console":
		fmt.Fprintln(l.consoleOut, consoleMsg)

	case "both":
		if l.buffer != nil {
			select {
			case l.buffer <- finalMsg:
			default:
				l.writeToFile(finalMsg)
			}
		} else {
			l.writeToFile(finalMsg)
		}
		fmt.Fprintln(l.consoleOut, consoleMsg)
	}
}

// инициализация логгера
func initLogger() {
	instance = &Logger{
		consoleOut: os.Stdout,
		logLevel:   getLevelPriority(LOG_LEVEL),
		stopCh:     make(chan struct{}),
	}

	// Настройка файлового вывода
	if LOG_OUTPUT == "file" || LOG_OUTPUT == "both" {
		os.MkdirAll(filepath.Dir(LOG_FILE_PATH), 0o755)
		file, err := os.OpenFile(LOG_FILE_PATH, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
		if err == nil {
			instance.file = file
		}

		// Создаем буфер для асинхронной записи
		if LOG_BUFFER_SIZE > 0 {
			instance.buffer = make(chan string, LOG_BUFFER_SIZE)
			instance.wg.Add(1)
			go instance.asyncWriter()
		}
	}
}

// asyncWriter асинхронная запись в файл
func (l *Logger) asyncWriter() {
	defer l.wg.Done()
	for {
		select {
		case msg := <-l.buffer:
			l.writeToFile(msg)
		case <-l.stopCh:
			// Опустошаем буфер перед выходом
			for len(l.buffer) > 0 {
				l.writeToFile(<-l.buffer)
			}
			return
		}
	}
}

// ==================== ПУБЛИЧНЫЕ ФУНКЦИИ ====================

// GetLogger возвращает экземпляр логгера
func GetLogger() *Logger {
	once.Do(initLogger)
	return instance
}

// Debug логирование уровня DEBUG
func Debug(format string, args ...any) {
	l := GetLogger()
	if l.logLevel <= LEVEL_DEBUG {
		msg := formatMessage(LEVEL_DEBUG, format, args...)
		l.write(LEVEL_DEBUG, getColor(LEVEL_DEBUG), msg)
	}
}

// Info логирование уровня INFO
func Info(format string, args ...any) {
	l := GetLogger()
	if l.logLevel <= LEVEL_INFO {
		msg := formatMessage(LEVEL_INFO, format, args...)
		l.write(LEVEL_INFO, getColor(LEVEL_INFO), msg)
	}
}

// Warn логирование уровня WARN
func Warn(format string, args ...any) {
	l := GetLogger()
	if l.logLevel <= LEVEL_WARN {
		msg := formatMessage(LEVEL_WARN, format, args...)
		l.write(LEVEL_WARN, getColor(LEVEL_WARN), msg)
	}
}

// Error логирование уровня ERROR
func Error(format string, args ...any) {
	l := GetLogger()
	if l.logLevel <= LEVEL_ERROR {
		msg := formatMessage(LEVEL_ERROR, format, args...)
		l.write(LEVEL_ERROR, getColor(LEVEL_ERROR), msg)
	}
}

// Fatal логирование фатальной ошибки с завершением программы
func Fatal(format string, args ...any) {
	Error(format, args...)
	Close()
	os.Exit(1)
}

// Close закрывает логгер (вызывать при завершении программы)
func Close() {
	if instance != nil {
		if instance.buffer != nil {
			close(instance.stopCh)
			instance.wg.Wait()
		}
		if instance.file != nil {
			instance.file.Close()
		}
	}
}

// SetLevel динамически изменяет уровень логирования
func SetLevel(level string) {
	if lvl, ok := levelPriority[strings.ToLower(level)]; ok {
		GetLogger().logLevel = lvl
		Info("Уровень логирования изменен на %s", strings.ToUpper(level))
	}
}

// Sync принудительно сбрасывает буфер
func Sync() {
	if instance != nil && instance.file != nil {
		instance.file.Sync()
	}
}
