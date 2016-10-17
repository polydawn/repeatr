package gs

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/storage/v1"
)

func httpClient(auth *oauth2.Token) *http.Client {
	return &http.Client{
		Transport: &oauth2.Transport{
			Source: oauth2.StaticTokenSource(auth),
		},
	}
}

func makeGsObjectService(token *oauth2.Token) (*storage.ObjectsService, error) {
	httpClient := httpClient(token)
	service, err := storage.New(httpClient)
	if err != nil {
		return nil, err
	}
	objService := storage.NewObjectsService(service)
	return objService, nil
}

func makeGsWriter(bucketName string, path string, token *oauth2.Token) (io.WriteCloser, <-chan error, error) {
	reader, writer := io.Pipe()
	service, err := makeGsObjectService(token)
	if err != nil {
		return nil, nil, err
	}
	object := &storage.Object{Name: path}
	errCh := make(chan error, 1)
	go func() {
		// TODO: multipart or resumable upload using `ResumableMedia`
		_, err := service.Insert(bucketName, object).Media(reader).Do()
		if err != nil {
			errCh <- err
		}
		close(errCh)
	}()
	return writer, errCh, nil
}

func reloc(bucketName, oldPath, newPath string, token *oauth2.Token) error {
	service, err := makeGsObjectService(token)
	if err != nil {
		return err
	}
	obj := &storage.Object{}
	rewrite := service.Rewrite(bucketName, oldPath, bucketName, newPath, obj)
	// Arbitrary limits, backoff required due to eventual consistency
	limit := 100
	backoff := 10 * time.Nanosecond
	var response *storage.RewriteResponse
	for i := 0; i < limit; i++ {
		response, err = rewrite.Do()
		if response.ServerResponse.HTTPStatusCode == http.StatusNotFound && backoff < time.Minute {
			time.Sleep(backoff)
			backoff = backoff + backoff
			continue
		}
		if err != nil {
			return err
		}
		if response.Done {
			break
		}
		rewrite = rewrite.RewriteToken(response.RewriteToken)
	}
	if !response.Done {
		return fmt.Errorf("RewriteGsDidNotComplete")
	}
	err = service.Delete(bucketName, oldPath).Do()
	if err != nil {
		return err
	}
	return nil
}
