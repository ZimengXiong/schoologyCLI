package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var unsafeName = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

type exportSummary struct {
	CreatedAt       string   `json:"created_at"`
	Sections        int      `json:"sections"`
	SuccessfulCalls int      `json:"successful_calls"`
	FilesDownloaded int      `json:"files_downloaded"`
	SyllabusItems   []string `json:"syllabus_items"`
	Warnings        []string `json:"warnings,omitempty"`
}

func runExport(args []string) error {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	output := fs.String("output", "schoology-export", "archive directory")
	activeOnly := fs.Bool("active-only", false, "exclude inactive sections")
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return err
	}
	c, err := newClientFromEnv()
	if err != nil {
		return err
	}
	sections, err := c.Sections()
	if err != nil {
		return err
	}
	root, err := filepath.Abs(*output)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		return err
	}
	s := exportSummary{CreatedAt: time.Now().Format(time.RFC3339)}
	writeFileJSON(filepath.Join(root, "sections.json"), sections)

	endpoints := []string{"assignments", "documents", "pages", "discussions", "events", "updates", "albums", "web_packages", "packages", "resources"}
	for _, sec := range sections {
		if *activeOnly && sec.Active != 1 {
			continue
		}
		s.Sections++
		name := fmt.Sprintf("%s_%d", safeName(sec.CourseTitle+"_"+sec.SectionTitle), sec.ID)
		dir := filepath.Join(root, "sections", name)
		if err := os.MkdirAll(filepath.Join(dir, "files"), 0700); err != nil {
			return err
		}
		var detail any
		if err := c.getJSON(fmt.Sprintf("/sections/%d", sec.ID), &detail); err == nil {
			writeFileJSON(filepath.Join(dir, "section.json"), detail)
			s.SuccessfulCalls++
		}
		for _, ep := range endpoints {
			path := fmt.Sprintf("/sections/%d/%s?limit=200&with_attachments=1", sec.ID, ep)
			var data any
			if err := c.getJSON(path, &data); err != nil {
				s.Warnings = append(s.Warnings, fmt.Sprintf("section %d %s: %v", sec.ID, ep, err))
				continue
			}
			s.SuccessfulCalls++
			writeFileJSON(filepath.Join(dir, ep+".json"), data)
			if strings.Contains(strings.ToLower(mustJSON(data)), "syllab") {
				s.SyllabusItems = append(s.SyllabusItems, filepath.Join("sections", name, ep+".json"))
			}
			s.FilesDownloaded += downloadAttachments(c, data, filepath.Join(dir, "files"), &s.Warnings)
		}
	}
	writeFileJSON(filepath.Join(root, "manifest.json"), s)
	writeFileJSON(filepath.Join(root, "syllabi-index.json"), s.SyllabusItems)
	fmt.Printf("Exported %d sections to %s (%d files downloaded, %d syllabus matches, %d warnings)\n", s.Sections, root, s.FilesDownloaded, len(s.SyllabusItems), len(s.Warnings))
	return nil
}

func downloadAttachments(c *Client, v any, dir string, warnings *[]string) int {
	seen := map[string]bool{}
	count := 0
	var walk func(any)
	walk = func(x any) {
		switch t := x.(type) {
		case map[string]any:
			var rawURL, filename string
			for k, val := range t {
				lk := strings.ToLower(k)
				if str, ok := val.(string); ok {
					if (lk == "download_path" || lk == "download_url" || lk == "url") && (strings.HasPrefix(str, "http://") || strings.HasPrefix(str, "https://")) {
						rawURL = str
					}
					if lk == "filename" || (lk == "title" && filename == "") {
						filename = str
					}
				}
			}
			if rawURL != "" && filename != "" && !seen[rawURL] {
				seen[rawURL] = true
				if err := downloadFile(c, rawURL, filepath.Join(dir, safeName(filename))); err != nil {
					*warnings = append(*warnings, "download "+filename+": "+err.Error())
				} else {
					count++
				}
			}
			for _, val := range t {
				walk(val)
			}
		case []any:
			for _, val := range t {
				walk(val)
			}
		}
	}
	walk(v)
	return count
}

func downloadFile(c *Client, raw, dest string) error {
	req, err := http.NewRequest(http.MethodGet, raw, nil)
	if err != nil {
		return err
	}
	u, _ := url.Parse(raw)
	base, _ := url.Parse(c.BaseURL)
	if u != nil && base != nil && u.Host == base.Host {
		req.Header.Set("Authorization", c.authorizationHeader())
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %s", resp.Status)
	}
	f, err := os.OpenFile(uniquePath(dest), os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func uniquePath(p string) string {
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return p
	}
	ext := filepath.Ext(p)
	base := strings.TrimSuffix(p, ext)
	for i := 2; ; i++ {
		q := fmt.Sprintf("%s_%d%s", base, i, ext)
		if _, err := os.Stat(q); os.IsNotExist(err) {
			return q
		}
	}
}
func safeName(s string) string {
	s = strings.Trim(unsafeName.ReplaceAllString(s, "_"), "._-")
	if s == "" {
		return "untitled"
	}
	return s
}
func writeFileJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0600)
}
func mustJSON(v any) string { b, _ := json.Marshal(v); return string(b) }
