package main

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

import (
	"context"
	"fmt"
	"io"
	"time"

	"golang.org/x/sync/errgroup"
)

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

const (
	chunkSize      = 1024
	workerAmount   = 15
	amountOfChunks = 5
)

func Connect(ctx context.Context, systemType string) (LogSystem, error)

func sendLogs(ctx context.Context, from LogSystem, to LogSystem, date time.Time) error {
	files, err := from.ListFiles(ctx, date)
	if err != nil {
		return fmt.Errorf("error %v: error while list files from log sys: %s", time.Now(), err)
	}
	chunkLog := make([]LogChunk, 0, amountOfChunks)
	for _, file := range files {
		chunk, err := from.ReadFile(ctx, date, file)
		if err != nil {
			return fmt.Errorf("error %v: error while reading file %s from log sys: %s", time.Now(), file, err)
		}
		if len(chunk) <= chunkSize {
			chunkLog = append(chunkLog, chunk)
		} else {
			fileSize := len(chunk) / chunkSize
			for i := 0; i < fileSize; i++ {
				chunkToSave := chunk[:chunkSize]
				chunkLog = append(chunkLog, chunkToSave)
				chunk = chunk[chunkSize:]
			}
			chunkLog = append(chunkLog, chunk)
		}

		if len(chunkLog) < amountOfChunks {
			continue
		}
		chunkSendSize := len(chunkLog) / amountOfChunks
		for i := 0; i < chunkSendSize; i++ {
			chunkToSend := chunkLog[:amountOfChunks]
			err = to.WriteLogs(ctx, date, chunkToSend)
			if err != nil {
				return fmt.Errorf("error %v: error while sendibng file %s to target log sys: %s", time.Now(), file, err)
			}
			chunkLog = chunkLog[amountOfChunks:]
		}
		err = to.WriteLogs(ctx, date, chunkLog)
		if err != nil {
			return fmt.Errorf("error %v: error while sendibng file %s to target log sys: %s", time.Now(), file, err)
		}
		chunkLog = chunkLog[:0]
	}
	return nil
}

// Если full=false - продолжить обработку с места последней ошибки
// Если full=true - обработать все логи заново
func Send(from string, to string, full bool) error {
	ctx := context.Background()

	fromLog, err := Connect(ctx, from)
	if err != nil {
		return fmt.Errorf("error %v: error while connecting to log system: %s", time.Now(), err)
	}
	defer fromLog.Close()

	toLog, err := Connect(ctx, to)
	if err != nil {
		return fmt.Errorf("error %v: error while connecting to log system: %s", time.Now(), err)
	}
	defer toLog.Close()

	datesFrom, err := fromLog.ListDates(ctx)
	if err != nil {
		return fmt.Errorf("error %v: error while getting dates from log sys: %s", time.Now(), err)
	}

	if !full {
		datesTo, err := toLog.ListDates(ctx)
		if err != nil {
			return fmt.Errorf("error %v: error while getting dates from log sys: %s", time.Now(), err)
		}
		if len(datesTo) != 0 {
			datesFrom = datesFrom[len(datesTo)-1:]
		}
	}

	g := new(errgroup.Group)
	g.SetLimit(workerAmount)
	for _, date := range datesFrom {
		g.Go(func() error {
			err := sendLogs(ctx, fromLog, toLog, date)
			return err
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}

	return nil
}
