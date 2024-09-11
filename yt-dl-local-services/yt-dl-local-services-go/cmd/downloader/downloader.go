package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/gcottom/yt-dl-3-hybrid/yd-dl-local-services/yt-dl-local-services-go/config"
	"github.com/gcottom/yt-dl-3-hybrid/yd-dl-local-services/yt-dl-local-services-go/pkg/http_client"
	"github.com/gcottom/yt-dl-3-hybrid/yd-dl-local-services/yt-dl-local-services-go/pkg/youtube_v2"
)

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
	if err = os.Mkdir(config.TempDir, 0755); err != nil && !os.IsExist(err) {
		panic(err)
	}
	savePath := fmt.Sprintf("%s/%s", config.TempDir, *id)
	if err = os.WriteFile(savePath, data, 0644); err != nil {
		panic(err)
	}
}
