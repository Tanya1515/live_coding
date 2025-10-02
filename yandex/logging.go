package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"golang.org/x/sync/errgroup"
)

// Есть две системы:

// INGEST - сервис приёма логов в реальном времени (например, Fluentd или Vector)
// WAREHOUSE - хранилище логов для аналитики (например, ClickHouse или Elasticsearch)
// На сервере INGEST в директории /var/log/archive хранятся сжатые gzip-логи в формате:

// c
// /var/log/archive/
//   2023-01-01/
//     service1.log.gz
//     service2.log.gz
//   2023-01-02/
//     ...
// Общий объём - около 100TB данных.

// Для работы с системами предполагается использовать интерфейс:
type LogChunk []byte

type LogSystem interface {
	io.Closer
	// Получить список всех доступных дат
	ListDates(ctx context.Context) ([]time.Time, error)
	// Получить список файлов для указанной даты
	ListFiles(ctx context.Context, date time.Time) ([]string, error)
	// Чтение файла (автоматически распаковывает gzip)
	ReadFile(ctx context.Context, date time.Time, filename string) (LogChunk, error)
	// Запись логов (идемпотентная операция)
	WriteLogs(ctx context.Context, date time.Time, logs []LogChunk) error
}

func Connect(ctx context.Context, systemType string) (LogSystem, error)

const (
	logChuckSize = 1024
	workerAmount = 10
)

func sendLogs(ctx context.Context, dateToSend time.Time, logTo LogSystem, logFrom LogSystem) error {
	files, err := logFrom.ListFiles(ctx, dateToSend)
	if err != nil {
		return fmt.Errorf("error while getting log files from system: %w", err)
	}

	buf := make([]LogChunk, 0, logChuckSize)

	j := 0

	for j < len(files) {
		select {
		case <-ctx.Done():
			return ctx.Err() // 1
		default:
			if len(buf) >= logChuckSize {
				err = logTo.WriteLogs(ctx, dateToSend, buf)
				if err != nil {
					return fmt.Errorf("error while writing logs to end log sys: %w", err) // 3
				}
				buf = buf[:0]
			}
			chunck, err := logFrom.ReadFile(ctx, dateToSend, files[j])
			if err != nil && !errors.Is(err, io.EOF) {
				return fmt.Errorf("error while getting file %s: %w", files[j], err)
			}
			buf = append(buf, chunck)
			if errors.Is(err, io.EOF) {
				j++
			}
		}
	}

	if len(buf) >= logChuckSize {
		err = logTo.WriteLogs(ctx, dateToSend, buf)
		if err != nil {
			return fmt.Errorf("error while writing logs to end log sys: %w", err) // 3
		}
	}

	return nil

}

func Send(from string, to string, full bool) error {
	ctxBase := context.Background()
	logTo, err := Connect(ctxBase, to)

	if err != nil {
		return fmt.Errorf("Can not connect to end log sys: %w", err)
	}

	logFrom, err := Connect(ctxBase, from)

	if err != nil {
		return fmt.Errorf("Can not connect to initial log sys: %w", err)
	}

	defer logFrom.Close()
	defer logTo.Close()

	datesFrom, err := logFrom.ListDates(ctxBase) // 7
	if err != nil {
		return fmt.Errorf("error while getting dates from source system")
	}

	datesTo, err := logTo.ListDates(ctxBase) // 8
	if err != nil {
		return fmt.Errorf("error while getting dates from target system")
	}

	if !full && len(datesTo) > 0 {
		datesFrom = datesFrom[len(datesTo)-1:]
	}

	if len(datesFrom) == 0 {
		return nil // 4
	}

	g, groupCtx := errgroup.WithContext(ctxBase)
	g.SetLimit(workerAmount)

	for i := 0; i < len(datesFrom); i++ {
		g.Go(func() error {
			return sendLogs(groupCtx, datesFrom[i], logTo, logFrom)
		})
	}

	return g.Wait()
}
