package commands

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/josephlewis42/honeyssh/core/vos"
	"github.com/juju/ratelimit"
)

// Curl implements a curl command.
func Curl(virtOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "curl [OPTIONS...] URL",
		Short: "Transfer a URL.",

		NeverBail: true,
	}

	quietPtr := cmd.Flags().BoolLong("silent", 's', "Silent mode")
	// Go's HTTP client follows redirects by default.
	_ = cmd.Flags().BoolLong("location", 'L', "Follow redirects")
	outputPtr := cmd.Flags().StringLong("output", 'o', "", "Write to location rather than stdout")

	return cmd.RunEachArg(virtOS, func(rawURL string) error {
		// Do this first, otherwise url.Parse has issues paring URLs with ports.
		if !strings.Contains(rawURL, "://") {
			rawURL = "http://" + rawURL
		}

		parsedURL, err := url.Parse(rawURL)
		if err != nil {
			return fmt.Errorf("invalid URL: %s", rawURL)
		}

		var destName string
		var destFd io.Writer
		var logFd io.Writer
		switch {
		case *outputPtr == "-" || *outputPtr == "":
			destName = "stdout"
			destFd = virtOS.Stdout()
			// If writing to the terminal, don't show progress.
			logFd = io.Discard

		default:
			destName = *outputPtr
			osFd, err := virtOS.Create(*outputPtr)
			if err != nil {
				return errors.New("couldn't create output file")
			}
			defer osFd.Close()
			destFd = osFd
			logFd = virtOS.Stdout()
		}

		if *quietPtr {
			logFd = io.Discard
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		downloadFd, err := virtOS.DownloadPath(parsedURL.String())
		if err != nil {
			return errors.New("couldn't create output file")
		}
		defer downloadFd.Close()

		request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), nil)
		if err != nil {
			return err
		}
		request.Header.Set("User-Agent", "curl/7.72.0")

		response, err := wgetHTTPClient.Do(request)
		if err != nil {
			return err
		}
		defer response.Body.Close()

		var contentLength int
		if lengthStr := response.Header.Get("Content-Length"); lengthStr != "" {
			contentLength, err = strconv.Atoi(lengthStr)
			if err != nil {
				// Ignore error parsing length, we'll still try to download.
				contentLength = 0
			}
		}

		// Rate limit to 2mbps
		tokenBucket := ratelimit.NewBucketWithRate(2*1000*1000, 2*1000*1000)

		countWriter := &countWriter{
			totalBytes: contentLength,
			fileName:   destName,
			virtOS:     virtOS,
			output:     logFd,
		}
		if _, err := io.Copy(io.MultiWriter(downloadFd, destFd, countWriter), ratelimit.Reader(response.Body, tokenBucket)); err != nil {
			return err
		}

		return nil
	})
}

func init() {
	addBinCmd("curl", Curl)
}
