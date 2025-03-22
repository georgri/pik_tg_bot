package telegrambot

import (
	"context"
	"fmt"
	"github.com/georgri/pik_tg_bot/pkg/backup_data"
	"github.com/georgri/pik_tg_bot/pkg/downloader"
	"github.com/georgri/pik_tg_bot/pkg/logrotator"
	"github.com/georgri/pik_tg_bot/pkg/util"
	"log"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const (
	invokeEvery = 1 * time.Minute

	maxHttpThreads = 10

	shutdownTimeout = 10 * time.Second
)

var (
	wg = &sync.WaitGroup{}
)

func RunForever() {

	err := logrotator.SetupLogging()
	if err != nil {
		panic(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGSTOP)
	defer stop()

	go logrotator.RotateLogsForever(ctx, wg)

	go backup_data.BackupDataForever(ctx, wg)

	go UpdateBlocksForever(ctx, wg)

	go GetUpdatesForever(ctx, wg)

	go RunUpdateFlatsForever(ctx, wg)

	<-ctx.Done()

	// wait for max 10 seconds
	shutdownCtx, cancelFunc := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancelFunc()

	var c chan struct{}
	go func(c chan struct{}) {
		wg.Wait()
		logrotator.CloseLog()
		c <- struct{}{}
	}(c)

	select {
	case <-shutdownCtx.Done():
		log.Fatalf("failed to exit gracefully - shutdown timeout reached")
	case <-c:
		log.Printf("shutting down gracefully")
	}
}

func RunUpdateFlatsForever(ctx context.Context, wg *sync.WaitGroup) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		RunUpdateFlatsOnce(ctx, wg)
		time.Sleep(invokeEvery)
	}
}

func RunUpdateFlatsOnce(ctx context.Context, wg *sync.WaitGroup) {
	log.Printf("begin to check for updates")

	envType := util.GetEnvType()

	// 1. Get map of block slug => subscribed channels
	// 2. Update block slug
	// 3. Send info to all subscribed channels

	slugs := make(map[string][]int64, 10)

	for _, channelInfo := range ChannelIDs[envType] {
		slugs[channelInfo.BlockSlug] = append(slugs[channelInfo.BlockSlug], channelInfo.ChatID)
	}

	var count int
	threadsWg := &sync.WaitGroup{}
	for slug, chatIDs := range slugs {

		select {
		case <-ctx.Done():
			break
		default:
		}

		threadsWg.Add(1)
		wg.Add(1)
		count += 1
		go func(slug string, chatIDs []int64, threadsWg, wg *sync.WaitGroup) {
			ProcessWithSlugAndChatIDs(slug, chatIDs)
			threadsWg.Done()
			wg.Done()
		}(slug, chatIDs, threadsWg, wg)

		if count%maxHttpThreads == 0 {
			threadsWg.Wait()
		}
	}

	// download info about all the unsubscribed blocks
	for slug := range BlockSlugs {
		if _, ok := slugs[slug]; ok {
			continue // already processed
		}

		select {
		case <-ctx.Done():
			break
		default:
		}

		threadsWg.Add(1)
		wg.Add(1)
		count += 1
		go func(slug string, chatIDs []int64, threadsWg, wg *sync.WaitGroup) {
			ProcessWithSlugAndChatIDs(slug, chatIDs)
			threadsWg.Done()
			wg.Done()
		}(slug, nil, threadsWg, wg)

		if count%maxHttpThreads == 0 {
			threadsWg.Wait()
		}
	}
	threadsWg.Wait() // wait synchronously before triggering the next job
	log.Printf("checked updates for %v projects", count)
}

func ProcessWithSlugAndChatIDs(blockSlug string, chatIDs []int64) {
	msgs, err := DownloadAndUpdateFile(blockSlug)
	if err != nil {
		//if err == errorNoNewFlats {
		//	return
		//}
		log.Printf("error while updating flats: %v", err)
		return
	}

	for i := range msgs {
		if len(msgs[i]) > 0 && msgs[i][0] == '!' { // let ! be the magic symbol to send to all known chats
			err = SendToAllKnownChats(msgs[i])
			if err != nil {
				log.Printf("error while sending message to all known chats about %v: %v", blockSlug, err)
				return
			}
		} else {
			for _, chatID := range chatIDs {
				err = SendMessage(chatID, msgs[i])
				if err != nil {
					log.Printf("error while sending message in %v (chatID %v): %v", blockSlug, chatID, err)
					return
				}
			}
		}
	}
}

func DownloadAndUpdateFile(blockSlug string) ([]string, error) {
	blockID := GetBlockIDBySlug(blockSlug)

	envtype := util.GetEnvType().String()

	flatMsgs, updateCallback, err := downloader.GetFlats(blockID)
	if err != nil {
		//if err == downloader.ErrorZeroFlats {
		//	return nil, errorNoNewFlats
		//}
		return nil, fmt.Errorf("error getting response from pik.ru: %v", err)
	}

	err = updateCallback()
	if err != nil {
		return nil, fmt.Errorf("update callback failed in %v (envtype %v): %v", blockSlug, envtype, err)
	}

	if len(flatMsgs) == 0 {
		return nil, fmt.Errorf("got 0 new flats for zhk %v", blockSlug)
	}

	log.Printf("Got flats in %v (envtype %v): %v", blockSlug, envtype, flatMsgs)

	return flatMsgs, nil
}
