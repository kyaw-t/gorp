package start

import (
	"bytes"
	"gorp/internals/config"
	"io"
	"io/ioutil"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

type NodePackage struct {
	scope string
	name  string
}

type NodeHandler struct {
	nodeConfig config.NodeConfig
	logger     slog.Logger
}

type ProxyRequestListOptions struct {
	urls   []string
	r      *http.Request
	w      http.ResponseWriter
	h      *NodeHandler
	logger slog.Logger
}

func parsePath(path string) (NodePackage, error) {

	// expected formats for path:
	// /node/:scope/:name/.... for scoped packages
	// /node/:name/.... for non-scoped packages
	nodePackage := NodePackage{}
	parts := strings.Split(path, "/")
	if strings.HasPrefix(parts[2], "@") {
		nodePackage.scope = parts[2]
		nodePackage.name = parts[3]
	} else {
		nodePackage.scope = ""
		nodePackage.name = parts[2]
	}
	return nodePackage, nil
}

func createProxyUrl(u string, path string) string {
	trimmedPath := strings.TrimPrefix(path, "/node")
	return u + trimmedPath
}

func wildcardToRegex(pattern string) string {
	pattern = regexp.QuoteMeta(pattern)
	pattern = strings.ReplaceAll(pattern, "\\*", ".*")
	return "^" + pattern + "$"
}

func matchWildCard(s string, pattern string) (bool, error) {
	matched, err := regexp.MatchString(wildcardToRegex(pattern), s)
	if err != nil {
		return false, err
	}
	return matched, nil
}

func findMapping(p NodePackage, nodeConfig config.NodeConfig) (string, error) {
	WILDCARD := "*"
	mappings := nodeConfig.Mappings

	for key, value := range mappings {
		if strings.Contains(key, WILDCARD) {
			name := ""
			if p.scope == "" {
				name = p.name
			} else {
				name = p.scope + "/" + p.name
			}
			matched, err := matchWildCard(name, key)
			if err != nil {
				return "", err
			}
			if matched {
				return value, nil
			}
		} else if key == p.scope+"/"+p.name {
			return value, nil
		}
	}
	return "", nil
}

func createProxyList(pkg NodePackage, path string, h *NodeHandler) ([]string, error) {
	urls := []string{}
	mapping, err := findMapping(pkg, h.nodeConfig)
	if err != nil {
		return urls, err
	}
	if mapping != "" {
		urls = append(urls, createProxyUrl(mapping, path))
		if h.nodeConfig.UseFallbackForMappings {
			for _, f := range h.nodeConfig.Fallback {
				urls = append(urls, createProxyUrl(f, path))
			}
		}
	} else {
		urls = append(urls, createProxyUrl(h.nodeConfig.Registry, path))
		for _, f := range h.nodeConfig.Fallback {
			urls = append(urls, createProxyUrl(f, path))
		}
	}
	return urls, nil
}

func proxyRequest(url *url.URL, r *http.Request) (*http.Response, error) {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequest(r.Method, url.String(), r.Body)
	if err != nil {
		return nil, err
	}

	for key, values := range r.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	req.Host = url.Host
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func copyResponse(w http.ResponseWriter, resp *http.Response) {
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	w.Write(body)
}

type copyDryRunResponseOptions struct {
	w       http.ResponseWriter
	resp    *http.Response
	target  string
	replace string
}

func copyDryRunResponse(options copyDryRunResponseOptions) {
	resp := options.resp
	w := options.w
	target := options.target
	replace := options.replace

	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	modifiedBody := strings.ReplaceAll(string(body), target, replace)
	w.Write([]byte(modifiedBody))
}

func proxyRequestList(options ProxyRequestListOptions) {

	urls := options.urls
	r := options.r
	w := options.w
	logger := options.logger
	h := options.h

	for idx, u := range urls {
		url, err := url.Parse(u)
		if err != nil {
			logger.Info("Failed to parse proxy url")
			http.Error(w, "Failed parse proxy url", http.StatusInternalServerError)
		}
		logger.Info("Proxying request", "from", r.URL.Path, "to", url.String())

		resp, err := proxyRequest(url, r)
		if err != nil {
			logger.Info("Failed to proxy request", "error", err)
			http.Error(w, "Failed to proxy request", http.StatusInternalServerError)
		}

		// If the repsonse is successful, we can return it
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {

			if h.nodeConfig.DryRun {
				logger.Info("Overriding tarball urls for dry run ")
				copyDryRunResponse(copyDryRunResponseOptions{
					w:       w,
					resp:    resp,
					target:  url.String(),
					replace: "http://localhost:3224",
				})
			}

			copyResponse(w, resp)
			resp.Body.Close()
			logger.Info("Successfully proxied request",
				"status", resp.Status,
				"from", r.URL.Path,
				"to", url.String())
			break

		} else {
			logger.Info("Failed to proxy request",
				"status", resp.Status,
				"from", r.URL.Path,
				"to", url.String())

			// If the response is not successful, we can keep trying
			// until we reach the last proxy in the list
			if idx == len(urls)-1 {
				logger.Info("All proxies failed. Returning last response",
					"status", resp.Status,
					"from", r.URL.Path,
					"to", url.String())
				copyResponse(w, resp)
				break
			}
		}
		defer resp.Body.Close()
	}
}

func (h *NodeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := h.logger

	if strings.Contains(r.URL.Path, ".tgz") && h.nodeConfig.DryRun {
		logger.Info("Received dry run request for tarball", "path", r.URL.Path)
		dummyTarball := []byte("This is a dummy tarball")
		dryRunResponse := &http.Response{
			StatusCode:    200,
			Header:        make(http.Header),
			ContentLength: int64(len(dummyTarball)),
			Body:          ioutil.NopCloser(bytes.NewReader(dummyTarball)),
		}

		dryRunResponse.Header.Set("Content-Type", "application/octet-stream")
		copyResponse(w, dryRunResponse)
		return
	}

	nodePackge, err := parsePath(r.URL.Path)
	if err != nil {
		logger.Info("Received unexpected package format", "package", r.URL.Path)
		http.Error(w, "Received unexpected package format", http.StatusInternalServerError)
	}

	logger.Info("Received request", "scope", nodePackge.scope, "name", nodePackge.name, "path", r.URL.Path)
	urls, err := createProxyList(nodePackge, r.URL.Path, h)
	if err != nil {
		logger.Info("Failed to create proxy list")
		http.Error(w, "Failed to create proxy list", http.StatusInternalServerError)
	}

	options := ProxyRequestListOptions{
		urls:   urls,
		r:      r,
		w:      w,
		h:      h,
		logger: logger,
	}
	proxyRequestList(options)
}
