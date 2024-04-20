package start

import (
	"gorp/internals/config"
	"io"
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

func createProxtList(pkg NodePackage, path string, h *NodeHandler) ([]string, error) {
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

func proxyRequestList(urls []string, r *http.Request, w http.ResponseWriter, logger slog.Logger) {
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

	nodePackge, err := parsePath(r.URL.Path)
	if err != nil {
		logger.Info("Received unexpected package format", "package", r.URL.Path)
		http.Error(w, "Received unexpected package format", http.StatusInternalServerError)
	}

	logger.Info("Received request", "scope", nodePackge.scope, "name", nodePackge.name, "path", r.URL.Path)
	urls, err := createProxtList(nodePackge, r.URL.Path, h)
	if err != nil {
		logger.Info("Failed to create proxy list")
		http.Error(w, "Failed to create proxy list", http.StatusInternalServerError)
	}

	proxyRequestList(urls, r, w, logger)
}
