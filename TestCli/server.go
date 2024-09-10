package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/CESSProject/cess-dcdn-components/cd2n-client/client"
	cdnlib "github.com/CESSProject/cess-dcdn-components/cd2n-lib"
	"github.com/CESSProject/cess-dcdn-components/cd2n-lib/types"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/mr-tron/base58/base58"
)

var cli *client.Client

type Request struct {
	Url      string `json:"url"`
	Filename string `json:"filename"`
}

func QueryCacheHandle(w http.ResponseWriter, r *http.Request) {
	var req Request
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	if err = json.Unmarshal(body, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	bPeerId, err := base58.Decode("12D3KooWQgrQLZ3x9WC43z9wFoJ6oZvTUVEqbZfkdF1kywfVinVp") //12D3KooWLK8DUHynHLxiPpoDtmEKh85Hs95V7mGNfUAZk4muFwJV
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Println("request 12D3KooWQgrQLZ3x9WC43z9wFoJ6oZvTUVEqbZfkdF1kywfVinVp")
	resp, err := cdnlib.QueryFileInfoFromCache(
		cli.CacheCli.Host, peer.ID(bPeerId),
		types.CacheRequest{
			AccountId: cli.GetPublickey(),
			WantUrl:   req.Url,
		})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if resp.Status != "hit" {
		http.Error(w, "wait CDepin node to cache resource", http.StatusOK)
		return
	}

	http.Error(w, "CDepin node cached resource success.", http.StatusOK)

}

func DownloadFromCacheHandle(w http.ResponseWriter, r *http.Request) {
	var req Request

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	if err = json.Unmarshal(body, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	bPeerId, err := base58.Decode("12D3KooWQgrQLZ3x9WC43z9wFoJ6oZvTUVEqbZfkdF1kywfVinVp")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req.Filename = "./" + req.Filename

	err = cdnlib.DownloadFileFromCache(
		cli.CacheCli.Host, peer.ID(bPeerId), req.Filename,
		types.CacheRequest{
			AccountId: cli.GetPublickey(),
			WantUrl:   req.Url,
		},
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	info, err := os.Stat(req.Filename)
	if err != nil {
		http.Error(w, "file not found", http.StatusInternalServerError)
		return
	}

	file, err := os.Open(req.Filename)
	if err != nil {
		http.Error(w, "file not found", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "filename="+filepath.Base(req.Filename))
	w.Header().Set("Content-Length", fmt.Sprint(info.Size()))

	_, err = io.Copy(w, file)
	if err != nil {
		http.Error(w, "error serving file", http.StatusInternalServerError)
		return
	}

}

func SetupHttpServer() {
	cli = InitTestClient()
	if cli == nil {
		log.Fatal("init test client error")
	}
	http.HandleFunc("/query", QueryCacheHandle)
	http.HandleFunc("/download", DownloadFromCacheHandle)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("run server error", err)
	}
}
