package gs

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/storage/v1"

	"go.polydawn.net/repeatr/rio"
)

func httpClient(auth *oauth2.Token) *http.Client {
	return &http.Client{
		Transport: &oauth2.Transport{
			Source: oauth2.StaticTokenSource(auth),
		},
	}
}

func makeGsObjectService(token *oauth2.Token) *storage.ObjectsService {
	httpClient := httpClient(token)
	service, err := storage.New(httpClient)
	if err != nil {
		panic(GsCredentialsInvalidError.Wrap(err))
	}
	objService := storage.NewObjectsService(service)
	return objService
}

func makeGsReader(bucketName string, path string, token *oauth2.Token) io.ReadCloser {
	service := makeGsObjectService(token)
	response, err := service.Get(bucketName, path).Download()
	if err != nil {
		panic(rio.WarehouseIOError.Wrap(err))
	}
	return response.Body
}

func makeGsWriter(bucketName string, path string, token *oauth2.Token) (io.WriteCloser, <-chan error) {
	reader, writer := io.Pipe()
	service := makeGsObjectService(token)
	object := &storage.Object{Name: path}
	errCh := make(chan error, 1)
	go func() {
		// TODO: multipart or resumable upload using `ResumableMedia`
		_, err := service.Insert(bucketName, object).Media(reader).Do()
		if err != nil {
			errCh <- rio.WarehouseIOError.Wrap(err)
		}
		close(errCh)
	}()
	return writer, errCh
}

func reloc(bucketName, oldPath, newPath string, token *oauth2.Token) {
	var response *storage.RewriteResponse
	var err error
	service := makeGsObjectService(token)
	obj := &storage.Object{}
	rewrite := service.Rewrite(bucketName, oldPath, bucketName, newPath, obj)
	// Arbitrary limits, backoff required due to eventual consistency
	limit := 100
	backoff := 10 * time.Nanosecond
	for i := 0; i < limit; i++ {
		response, err = rewrite.Do()
		if response.ServerResponse.HTTPStatusCode == http.StatusNotFound && backoff < time.Minute {
			time.Sleep(backoff)
			backoff = backoff + backoff
			continue
		}
		if err != nil {
			panic(rio.WarehouseIOError.Wrap(err))
		}
		if response.Done {
			break
		}
		rewrite = rewrite.RewriteToken(response.RewriteToken)
	}
	if !response.Done {
		panic(rio.WarehouseIOError.Wrap(fmt.Errorf("RewriteGsDidNotComplete")))
	}
	err = service.Delete(bucketName, oldPath).Do()
	if err != nil {
		panic(rio.WarehouseIOError.Wrap(err))
	}
}
