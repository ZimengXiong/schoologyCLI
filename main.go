package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"
)

const defaultBaseURL = "https://api.schoology.com/v1"

type Client struct {
	Key     string
	Secret  string
	BaseURL string
	Client  *http.Client
}

type apiLinks struct {
	Self string `json:"self"`
	Next string `json:"next"`
}

type user struct {
	ID         int64  `json:"id"`
	NameFirst  string `json:"name_first"`
	NameLast   string `json:"name_last"`
	PrimaryEmail string `json:"primary_email"`
	Role       string `json:"role"`
}

type section struct {
	ID           int64  `json:"id,string"`
	CourseTitle  string `json:"course_title"`
	CourseCode   string `json:"course_code"`
	SectionTitle string `json:"section_title"`
	Active       int    `json:"active"`
	Links        apiLinks `json:"links"`
}

type assignment struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Description string `json:"description"`
	Due       string `json:"due"`
	Published int    `json:"published"`
	Available int    `json:"available"`
	Completed int    `json:"completed"`
	WebURL    string `json:"web_url"`
}

type event struct {
	ID           int64  `json:"id"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	Start        string `json:"start"`
	Type         string `json:"type"`
	AssignmentID int64  `json:"assignment_id"`
	WebURL       string `json:"web_url"`
}

type sectionsPage struct {
	Section []section `json:"section"`
	Links   apiLinks  `json:"links"`
}

type assignmentsPage struct {
	Assignment []assignment `json:"assignment"`
	Links      apiLinks     `json:"links"`
}

type eventsPage struct {
	Event []event  `json:"event"`
	Links apiLinks `json:"links"`
}

type upcomingItem struct {
	Course    string `json:"course"`
	SectionID int64  `json:"section_id"`
	Title     string `json:"title"`
	Due       string `json:"due"`
	WebURL    string `json:"web_url"`
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printUsage(os.Stdout)
		return nil
	}

	switch args[0] {
	case "help", "-h", "--help":
		printUsage(os.Stdout)
		return nil
	case "me":
		return runMe(args[1:])
	case "sections":
		return runSections(args[1:])
	case "assignments":
		return runAssignments(args[1:])
	case "upcoming":
		return runUpcoming(args[1:])
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runMe(args []string) error {
	fs := flag.NewFlagSet("me", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "output JSON")
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return err
	}

	client, err := newClientFromEnv()
	if err != nil {
		return err
	}

	me, err := client.Me()
	if err != nil {
		return err
	}

	if *jsonOut {
		return writeJSON(me)
	}

	fmt.Printf("ID:\t%d\n", me.ID)
	fmt.Printf("Name:\t%s %s\n", me.NameFirst, me.NameLast)
	if me.PrimaryEmail != "" {
		fmt.Printf("Email:\t%s\n", me.PrimaryEmail)
	}
	if me.Role != "" {
		fmt.Printf("Role:\t%s\n", me.Role)
	}
	return nil
}

func runSections(args []string) error {
	fs := flag.NewFlagSet("sections", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "output JSON")
	all := fs.Bool("all", false, "include inactive sections")
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return err
	}

	client, err := newClientFromEnv()
	if err != nil {
		return err
	}

	sections, err := client.Sections()
	if err != nil {
		return err
	}
	if !*all {
		filtered := sections[:0]
		for _, s := range sections {
			if s.Active == 1 {
				filtered = append(filtered, s)
			}
		}
		sections = filtered
	}

	sort.Slice(sections, func(i, j int) bool {
		if sections[i].CourseTitle == sections[j].CourseTitle {
			return sections[i].SectionTitle < sections[j].SectionTitle
		}
		return sections[i].CourseTitle < sections[j].CourseTitle
	})

	if *jsonOut {
		return writeJSON(sections)
	}

	tw := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "SECTION_ID\tCOURSE\tSECTION\tACTIVE")
	for _, s := range sections {
		fmt.Fprintf(tw, "%d\t%s\t%s\t%d\n", s.ID, s.CourseTitle, s.SectionTitle, s.Active)
	}
	return tw.Flush()
}

func runAssignments(args []string) error {
	fs := flag.NewFlagSet("assignments", flag.ContinueOnError)
	sectionID := fs.Int64("section", 0, "section ID")
	limit := fs.Int("limit", 0, "limit results after fetch")
	jsonOut := fs.Bool("json", false, "output JSON")
	incompleteOnly := fs.Bool("incomplete", false, "show only assignments not marked completed")
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *sectionID == 0 {
		return errors.New("assignments requires --section <section-id>")
	}

	client, err := newClientFromEnv()
	if err != nil {
		return err
	}

	assignments, err := client.Assignments(*sectionID)
	if err != nil {
		return err
	}

	if *incompleteOnly {
		filtered := assignments[:0]
		for _, a := range assignments {
			if a.Completed != 1 {
				filtered = append(filtered, a)
			}
		}
		assignments = filtered
	}

	sort.Slice(assignments, func(i, j int) bool {
		return compareDue(assignments[i].Due, assignments[j].Due)
	})

	if *limit > 0 && *limit < len(assignments) {
		assignments = assignments[:*limit]
	}

	if *jsonOut {
		return writeJSON(assignments)
	}

	tw := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "ASSIGNMENT_ID\tDUE\tCOMPLETED\tTITLE\tURL")
	for _, a := range assignments {
		fmt.Fprintf(tw, "%d\t%s\t%d\t%s\t%s\n", a.ID, displayTime(a.Due), a.Completed, a.Title, a.WebURL)
	}
	return tw.Flush()
}

func runUpcoming(args []string) error {
	fs := flag.NewFlagSet("upcoming", flag.ContinueOnError)
	days := fs.Int("days", 14, "show assignment events due within N days")
	jsonOut := fs.Bool("json", false, "output JSON")
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return err
	}

	client, err := newClientFromEnv()
	if err != nil {
		return err
	}

	items, err := client.Upcoming(*days)
	if err != nil {
		return err
	}

	if *jsonOut {
		return writeJSON(items)
	}

	tw := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "DUE\tCOURSE\tTITLE\tURL")
	for _, item := range items {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", displayTime(item.Due), item.Course, item.Title, item.WebURL)
	}
	return tw.Flush()
}

func newClientFromEnv() (*Client, error) {
	key := strings.TrimSpace(os.Getenv("SCHOOLOGY_KEY"))
	secret := strings.TrimSpace(os.Getenv("SCHOOLOGY_SECRET"))
	baseURL := strings.TrimSpace(os.Getenv("SCHOOLOGY_API_BASE"))
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	if key == "" || secret == "" {
		return nil, errors.New("set SCHOOLOGY_KEY and SCHOOLOGY_SECRET")
	}
	return &Client{
		Key:     key,
		Secret:  secret,
		BaseURL: strings.TrimRight(baseURL, "/"),
		Client:  &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (c *Client) Me() (user, error) {
	var me user
	if err := c.getJSON("/users/me", &me); err != nil {
		return user{}, err
	}
	return me, nil
}

func (c *Client) Sections() ([]section, error) {
	me, err := c.Me()
	if err != nil {
		return nil, err
	}

	var out []section
	next := fmt.Sprintf("/users/%d/sections?limit=200", me.ID)
	for next != "" {
		var page sectionsPage
		if err := c.getJSON(next, &page); err != nil {
			return nil, err
		}
		out = append(out, page.Section...)
		next = nextPath(page.Links.Next)
	}
	return out, nil
}

func (c *Client) Assignments(sectionID int64) ([]assignment, error) {
	var out []assignment
	next := fmt.Sprintf("/sections/%d/assignments?limit=200", sectionID)
	for next != "" {
		var page assignmentsPage
		if err := c.getJSON(next, &page); err != nil {
			return nil, err
		}
		out = append(out, page.Assignment...)
		next = nextPath(page.Links.Next)
	}
	return out, nil
}

func (c *Client) Events(sectionID int64) ([]event, error) {
	var out []event
	next := fmt.Sprintf("/sections/%d/events?limit=200", sectionID)
	for next != "" {
		var page eventsPage
		if err := c.getJSON(next, &page); err != nil {
			return nil, err
		}
		out = append(out, page.Event...)
		next = nextPath(page.Links.Next)
	}
	return out, nil
}

func (c *Client) Upcoming(days int) ([]upcomingItem, error) {
	sections, err := c.Sections()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	cutoff := now.Add(time.Duration(days) * 24 * time.Hour)
	var items []upcomingItem

	for _, s := range sections {
		if s.Active != 1 {
			continue
		}

		events, err := c.Events(s.ID)
		if err != nil {
			return nil, err
		}

		for _, e := range events {
			if e.Type != "assignment" || e.Start == "" {
				continue
			}
			due, err := parseSchoologyTime(e.Start)
			if err != nil {
				continue
			}
			if due.Before(now) || due.After(cutoff) {
				continue
			}
			items = append(items, upcomingItem{
				Course:    s.CourseTitle,
				SectionID: s.ID,
				Title:     e.Title,
				Due:       e.Start,
				WebURL:    e.WebURL,
			})
		}
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Due == items[j].Due {
			if items[i].Course == items[j].Course {
				return items[i].Title < items[j].Title
			}
			return items[i].Course < items[j].Course
		}
		return items[i].Due < items[j].Due
	})
	return items, nil
}

func (c *Client) getJSON(pathOrURL string, out any) error {
	target := pathOrURL
	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
		target = c.BaseURL + pathOrURL
	}
	return c.getJSONURL(target, out, 0)
}

func (c *Client) getJSONURL(target string, out any, redirects int) error {
	if redirects > 5 {
		return errors.New("too many redirects")
	}

	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", c.authorizationHeader())

	transport := c.Client
	noRedirect := *transport
	noRedirect.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}

	resp, err := noRedirect.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if isRedirect(resp.StatusCode) {
		location := resp.Header.Get("Location")
		if location == "" {
			return fmt.Errorf("redirect without location from %s", target)
		}
		nextURL, err := resolveURL(target, location)
		if err != nil {
			return err
		}
		return c.getJSONURL(nextURL, out, redirects+1)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("schoology API %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) authorizationHeader() string {
	nonce := fmt.Sprintf("%016x", rand.Uint64())
	timestamp := time.Now().Unix()
	return fmt.Sprintf(
		`OAuth realm="Schoology API", oauth_consumer_key="%s", oauth_nonce="%s", oauth_signature_method="PLAINTEXT", oauth_timestamp="%d", oauth_token="", oauth_version="1.0", oauth_signature="%s%%26"`,
		c.Key,
		nonce,
		timestamp,
		c.Secret,
	)
}

func isRedirect(status int) bool {
	switch status {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther, http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		return true
	default:
		return false
	}
}

func resolveURL(base, next string) (string, error) {
	baseURL, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	nextURL, err := url.Parse(next)
	if err != nil {
		return "", err
	}
	return baseURL.ResolveReference(nextURL).String(), nil
}

func nextPath(next string) string {
	if next == "" {
		return ""
	}
	u, err := url.Parse(next)
	if err != nil {
		return next
	}
	if u.Scheme == "" && u.Host == "" {
		return next
	}
	if u.RawQuery == "" {
		return u.Path
	}
	return u.Path + "?" + u.RawQuery
}

func parseSchoologyTime(value string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02 15:04:05", value, time.Local)
}

func compareDue(a, b string) bool {
	switch {
	case a == "" && b == "":
		return false
	case a == "":
		return false
	case b == "":
		return true
	default:
		return a < b
	}
}

func displayTime(value string) string {
	if value == "" {
		return "-"
	}
	t, err := parseSchoologyTime(value)
	if err != nil {
		return value
	}
	return t.Format("2006-01-02 15:04")
}

func writeJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "schoologyCLI")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  schoologyCLI me [--json]")
	fmt.Fprintln(w, "  schoologyCLI sections [--all] [--json]")
	fmt.Fprintln(w, "  schoologyCLI assignments --section <id> [--limit N] [--incomplete] [--json]")
	fmt.Fprintln(w, "  schoologyCLI upcoming [--days N] [--json]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Environment:")
	fmt.Fprintln(w, "  SCHOOLOGY_KEY       Schoology consumer key")
	fmt.Fprintln(w, "  SCHOOLOGY_SECRET    Schoology consumer secret")
	fmt.Fprintln(w, "  SCHOOLOGY_API_BASE  Optional API base URL (default https://api.schoology.com/v1)")
}
