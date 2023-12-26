package commands

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/josephlewis42/honeyssh/core/vos"
	"github.com/juju/ratelimit"
)

func remoteFilename(url *url.URL) (string, error) {
	_, file := path.Split(url.Path)
	if file == "" {
		return "", errors.New("Remote file name has no length")
	}
	return file, nil
}

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
	_ = cmd.Flags().IntLong("max-redirs", 0, -1, "Set the maximum number of redirects to follow")

	outputPtr := cmd.Flags().StringLong("output", 'o', "", "Write to location rather than stdout")
	additionalHeaders := cmd.Flags().ListLong("header", 'H', "Append additional headers to the requrest")
	method := cmd.Flags().StringLong("request", 'X', http.MethodGet, "Specifies the request method to use (default: GET)")
	useRemoteName := cmd.Flags().BoolLong("remote-name", 'O', "Write output to a local file named like the remote file we get.")

	lastArgLookedLikeFlag := false
	return cmd.RunEachArg(virtOS, func(rawURL string) error {
		// The honeypot's flag parsing is limited, don't try to download things that look like flags
		// or their arguments.
		if strings.HasPrefix(rawURL, "-") {
			virtOS.LogInvalidInvocation(fmt.Errorf("flag parsed as argument: %q", rawURL))
			lastArgLookedLikeFlag = true
			return nil
		}
		if lastArgLookedLikeFlag && !strings.Contains(rawURL, ".") {
			lastArgLookedLikeFlag = false
			virtOS.LogInvalidInvocation(fmt.Errorf("argument received that doens't look like a URL, probably missing flag? %q", rawURL))
			return nil
		}
		lastArgLookedLikeFlag = false

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
		case *useRemoteName:
			destName, err = remoteFilename(parsedURL)
			if err != nil {
				return err
			}

			osFd, err := virtOS.Create(destName)
			if err != nil {
				return errors.New("couldn't create output file")
			}
			defer osFd.Close()
			destFd = osFd
			logFd = virtOS.Stdout()

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

		request, err := http.NewRequestWithContext(ctx, *method, parsedURL.String(), nil)
		if err != nil {
			return err
		}
		request.Header.Set("User-Agent", "curl/7.72.0")

		// Add additional headers specified on the command line.
		for _, header := range *additionalHeaders {
			headerParts := strings.SplitN(header, ": ", 2)
			switch len(headerParts) {
			case 2:
				key, value := headerParts[0], headerParts[1]
				request.Header[key] = append(request.Header[key], value)

			default:
				key, value := headerParts[0], ""
				request.Header[key] = append(request.Header[key], value)
			}
		}

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
	mustAddBinCmd("curl", Curl)
}
