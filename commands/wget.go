package commands

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/abiosoft/readline"
	"github.com/juju/ratelimit"
	"josephlewis.net/honeyssh/core/vos"
)

// wgetSocketControl prevents basic SSRF attacks by only allowing certain kinds
// of connections.
func wgetSocketControl(network string, address string, conn syscall.RawConn) error {
	switch network {
	case "tcp", "tcp4", "tcp6", "udp", "udp4", "udp6":
		// Accept types used for HTTP1/2/3.
	default:
		// This is likely something like the attacker trying to open a UNIX socket,
		// bail.
		return fmt.Errorf("unknown network type: %v", network)
	}

	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("bad network address: %v", address)
	}

	ipAddress := net.ParseIP(host)
	if ipAddress == nil {
		return fmt.Errorf("bad network address: %v", address)
	}

	if ipAddress.IsLoopback() || ipAddress.IsPrivate() {
		// Prevent loopback or fetches to private networks.
		return fmt.Errorf("couldn't resolve: %s", address)
	}

	return nil
}

var wgetDialer = &net.Dialer{
	Timeout: 5 * time.Second,
	Control: wgetSocketControl,
}

var wgetTransport = &http.Transport{
	Proxy:                 http.ProxyFromEnvironment,
	ForceAttemptHTTP2:     true,
	MaxIdleConns:          10,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   5 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
	DialContext:           wgetDialer.DialContext,
}

var wgetHTTPClient = &http.Client{
	Transport: wgetTransport,
}

func newReadline(virtOS vos.VOS) (*readline.Instance, error) {
	cfg := &readline.Config{
		Stdin:  virtOS.Stdin(),
		Stdout: virtOS.Stdout(),
		Stderr: virtOS.Stderr(),
		FuncGetWidth: func() int {
			return virtOS.GetPTY().Width
		},
		FuncIsTerminal: func() bool {
			return virtOS.GetPTY().IsPTY
		},
	}

	if err := cfg.Init(); err != nil {
		return nil, err
	}

	return readline.NewEx(cfg)
}

// Wget implements a wget command.
func Wget(virtOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "wget [OPTION...] [URL]...",
		Short: "Remove empty directories.",

		NeverBail: true,
	}

	return cmd.RunEachArg(virtOS, func(rawURL string) error {
		// Do this first, otherwise url.Parse has issues paring URLs with ports.
		if !strings.Contains(rawURL, "://") {
			rawURL = "http://" + rawURL
		}

		parsedURL, err := url.Parse(rawURL)
		if err != nil {
			return fmt.Errorf("invalid URL: %s", rawURL)
		}

		w := virtOS.Stdout()

		fmt.Fprintf(w, "--%s--  %s\n", virtOS.Now().Format("2006-01-02 15:04:05"), parsedURL.String())
		fmt.Fprintf(w, "Resolving %s...\n", parsedURL.Host)
		fmt.Fprintf(w, "Connecting to %s...\n", parsedURL.Host)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		downloadFd, err := virtOS.DownloadPath(parsedURL.String())
		if err != nil {
			return errors.New("couldn't create output file")
		}
		defer downloadFd.Close()

		fmt.Fprint(w, "HTTP request sent, awaiting response...")
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), nil)
		if err != nil {
			return err
		}

		response, err := wgetHTTPClient.Do(request)
		if err != nil {
			return err
		}
		defer response.Body.Close()
		fmt.Fprintf(w, " %s\n", response.Status)

		var contentLength int
		if lengthStr := response.Header.Get("Content-Length"); lengthStr != "" {
			contentLength, err = strconv.Atoi(lengthStr)
			if err != nil {
				// Ignore error parsing length, we'll still try to download.
				contentLength = 0
			}
		}
		contentType := "application/binary"
		if cth := response.Header.Get("Content-Type"); cth != "" {
			contentType = cth
		}
		fmt.Fprintf(w, "Length %d (%s) [%s]\n", contentLength, BytesToHuman(int64(contentLength)), contentType)

		destName := "index.html"
		if base := path.Base(parsedURL.Path); base != "." && base != "/" {
			destName = base
		}
		fmt.Fprintf(w, "Saving to %s\n", destName)
		fmt.Fprintln(w)

		localFd, err := virtOS.Create(destName)
		if err != nil {
			return errors.New("couldn't create output file")
		}
		defer localFd.Close()

		// Rate limit to 2mbps
		tokenBucket := ratelimit.NewBucketWithRate(2*1000*1000, 2*1000*1000)

		countWriter := &countWriter{
			totalBytes: contentLength,
			fileName:   destName,
			virtOS:     virtOS,
		}
		if _, err := io.Copy(io.MultiWriter(downloadFd, localFd, countWriter), ratelimit.Reader(response.Body, tokenBucket)); err != nil {
			return err
		}

		fmt.Fprint(w, "\n")

		fmt.Fprintf(w, "%s - %q saved\n", virtOS.Now().Format("2006-01-02 15:04:05"), destName)

		return nil
	})
}

type countWriter struct {
	bytesWritten int

	totalBytes int
	fileName   string
	startTime  time.Time

	virtOS vos.VOS
}

func (c *countWriter) Write(b []byte) (int, error) {
	c.bytesWritten += len(b)
	c.UpdateOutput()
	return len(b), nil
}

func (c *countWriter) UpdateOutput() {
	if c.startTime.IsZero() {
		c.startTime = c.virtOS.Now()
	}

	var percent float64
	if c.totalBytes > 0 {
		percent = 100 * (float64(c.bytesWritten) / float64(c.totalBytes))
	}

	deltaS := (c.virtOS.Now().Sub(c.startTime)) / time.Second
	var kbps float64
	if deltaS > 0 {
		kbps = float64(c.bytesWritten) / 1000.0 / float64(deltaS)
	}

	var timeLeft time.Duration
	if c.totalBytes > 0 {
		remainingBytes := (c.totalBytes - c.bytesWritten)
		timeLeft = time.Duration(float64(remainingBytes/1000)/kbps) * time.Second
	}

	progress := strings.Repeat("=", int(percent)/5) + ">"

	// index.html          100%[===================>]  13.79K  --.-KB/s    in 0.001s
	fmt.Fprintf(c.virtOS.Stdout(), "\r") // move back to the beginning of the line.
	fmt.Fprintf(c.virtOS.Stdout(),
		"%-20.20s % 3.0f%%[%-20.20s] %s  %3.1fKB/s    in %-6.6s",
		c.fileName,
		percent,
		progress,
		BytesToHuman(int64(c.bytesWritten)),
		kbps,
		timeLeft.String(),
	)
}

func init() {
	addBinCmd("wget", Wget)
}
