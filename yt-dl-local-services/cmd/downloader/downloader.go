package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/gcottom/yt-dl-3-hybrid/yd-dl-local-service/config"
	"github.com/gcottom/yt-dl-3-hybrid/yd-dl-local-service/pkg/http_client"
	"github.com/gcottom/yt-dl-3-hybrid/yd-dl-local-service/pkg/youtube_v2"
)

const tempDir = "ytdl-temp"

func main() {
	id := flag.String("id", "", "ID of the video to download")
	flag.Parse()
	config, err := config.LoadConfigFromFile("")
	if err != nil {
		panic(err)
	}
	httpClient := http_client.NewHTTPClient()
	ytClient := youtube_v2.NewYoutubeClient(config, httpClient)
	data, err := ytClient.Download(context.Background(), *id, false)
	if err != nil {
		panic(err)
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	if err = os.Mkdir(fmt.Sprintf("%s/%s", homeDir, tempDir), 0755); err != nil && !os.IsExist(err) {
		panic(err)
	}
	savePath := fmt.Sprintf("%s/%s/%s", homeDir, tempDir, *id)
	if err = os.WriteFile(savePath, data, 0644); err != nil {
		panic(err)
	}
}
