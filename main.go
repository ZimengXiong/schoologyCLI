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
	"os/exec"
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
	ID           int64  `json:"id"`
	NameFirst    string `json:"name_first"`
	NameLast     string `json:"name_last"`
	PrimaryEmail string `json:"primary_email"`
	Role         string `json:"role"`
}

type section struct {
	ID           int64    `json:"id,string"`
	CourseTitle  string   `json:"course_title"`
	CourseCode   string   `json:"course_code"`
	SectionTitle string   `json:"section_title"`
	Active       int      `json:"active"`
	Links        apiLinks `json:"links"`
}

type assignment struct {
	ID          int64           `json:"id"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Due         string          `json:"due"`
	Published   int             `json:"published"`
	Available   int             `json:"available"`
	Completed   int             `json:"completed"`
	WebURL      string          `json:"web_url"`
	Attachments json.RawMessage `json:"attachments,omitempty"`
	Tags        json.RawMessage `json:"tags,omitempty"`
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
	Course       string `json:"course"`
	SectionID    int64  `json:"section_id"`
	AssignmentID int64  `json:"assignment_id"`
	Title        string `json:"title"`
	Due          string `json:"due"`
	WebURL       string `json:"web_url"`
}

type revision struct {
	RevisionID int64 `json:"revision_id"`
	UID        int64 `json:"uid"`
	Created    int64 `json:"created"`
	Late       int   `json:"late"`
	Draft      int   `json:"draft"`
	NumItems   int   `json:"num_items"`
}

type revisionsPage struct {
	Revision []revision `json:"revision"`
	Links    apiLinks   `json:"links"`
}

type update struct {
	ID          int64  `json:"id"`
	Body        string `json:"body"`
	UID         int64  `json:"uid"`
	DisplayName string `json:"display_name"`
	Created     int64  `json:"created"`
	LastUpdated int64  `json:"last_updated"`
}

type updatesPage struct {
	Update []update `json:"update"`
	Links  apiLinks `json:"links"`
}

type document struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	CourseFID int64  `json:"course_fid"`
	Available int    `json:"available"`
	Published int    `json:"published"`
	Completed int    `json:"completed"`
	URL       string `json:"url"`
}

type documentsPage struct {
	Document []document `json:"document"`
	Links    apiLinks   `json:"links"`
}

type page struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	Parent    int64  `json:"parent"`
	Published int    `json:"published"`
	Available int    `json:"available"`
	Completed int    `json:"completed"`
	Created   int64  `json:"created"`
}

type pagesPage struct {
	Page  []page   `json:"page"`
	Links apiLinks `json:"links"`
}

type auditItem struct {
	Course             string    `json:"course"`
	SectionID          int64     `json:"section_id"`
	AssignmentID       int64     `json:"assignment_id"`
	Title              string    `json:"title"`
	Description        string    `json:"description,omitempty"`
	Due                string    `json:"due"`
	WebURL             string    `json:"web_url"`
	SubmissionStatus   string    `json:"submission_status"`
	LatestSubmission   *revision `json:"latest_submission,omitempty"`
	SubmissionError    string    `json:"submission_error,omitempty"`
	ExternalSubmission bool      `json:"external_submission"`
	Actionable         bool      `json:"actionable"`
}

type auditResult struct {
	GeneratedAt  string      `json:"generated_at"`
	Days         int         `json:"days"`
	CourseFilter string      `json:"course_filter,omitempty"`
	Items        []auditItem `json:"items"`
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
	case "submissions":
		return runSubmissions(args[1:])
	case "assignment":
		return runAssignment(args[1:])
	case "updates":
		return runUpdates(args[1:])
	case "documents":
		return runDocuments(args[1:])
	case "pages":
		return runPages(args[1:])
	case "audit", "todos":
		return runAudit(args[0], args[1:])
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runSubmissions(args []string) error {
	fs := flag.NewFlagSet("submissions", flag.ContinueOnError)
	sectionID := fs.Int64("section", 0, "section ID")
	assignmentID := fs.Int64("assignment", 0, "assignment/grade item ID")
	jsonOut := fs.Bool("json", false, "output JSON")
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *sectionID == 0 || *assignmentID == 0 {
		return errors.New("submissions requires --section <id> --assignment <id>")
	}

	client, err := newClientFromEnv()
	if err != nil {
		return err
	}
	me, err := client.Me()
	if err != nil {
		return err
	}
	revisions, err := client.Submissions(*sectionID, *assignmentID, me.ID)
	if err != nil {
		return err
	}

	if *jsonOut {
		return writeJSON(struct {
			Submitted bool       `json:"submitted"`
			Revisions []revision `json:"revisions"`
		}{Submitted: hasSubmittedRevision(revisions), Revisions: revisions})
	}

	fmt.Printf("Submitted:\t%t\n", hasSubmittedRevision(revisions))
	tw := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "REVISION_ID\tCREATED\tLATE\tDRAFT\tITEMS")
	for _, r := range revisions {
		created := time.Unix(r.Created, 0).Format("2006-01-02 15:04")
		fmt.Fprintf(tw, "%d\t%s\t%d\t%d\t%d\n", r.RevisionID, created, r.Late, r.Draft, r.NumItems)
	}
	return tw.Flush()
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
	course := fs.String("course", "", "filter course or section title (case-insensitive)")
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
	if *course != "" {
		filtered := sections[:0]
		for _, s := range sections {
			if sectionMatches(s, *course) {
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

func runAssignment(args []string) error {
	fs := flag.NewFlagSet("assignment", flag.ContinueOnError)
	sectionID := fs.Int64("section", 0, "section ID")
	assignmentID := fs.Int64("assignment", 0, "assignment ID")
	jsonOut := fs.Bool("json", false, "output JSON")
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *sectionID == 0 || *assignmentID == 0 {
		return errors.New("assignment requires --section <id> --assignment <id>")
	}
	client, err := newClientFromEnv()
	if err != nil {
		return err
	}
	a, err := client.Assignment(*sectionID, *assignmentID)
	if err != nil {
		return err
	}
	if *jsonOut {
		return writeJSON(a)
	}
	fmt.Printf("ID:\t%d\nTitle:\t%s\nDue:\t%s\nURL:\t%s\nDescription:\t%s\n", a.ID, a.Title, displayTime(a.Due), a.WebURL, a.Description)
	return nil
}

func runUpdates(args []string) error {
	fs := flag.NewFlagSet("updates", flag.ContinueOnError)
	sectionID := fs.Int64("section", 0, "section ID")
	limit := fs.Int("limit", 0, "limit results after fetch")
	jsonOut := fs.Bool("json", false, "output JSON")
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *sectionID == 0 {
		return errors.New("updates requires --section <id>")
	}
	client, err := newClientFromEnv()
	if err != nil {
		return err
	}
	items, err := client.Updates(*sectionID)
	if err != nil {
		return err
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Created > items[j].Created })
	if *limit > 0 && *limit < len(items) {
		items = items[:*limit]
	}
	if *jsonOut {
		return writeJSON(items)
	}
	tw := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "CREATED\tAUTHOR\tBODY")
	for _, item := range items {
		fmt.Fprintf(tw, "%s\t%s\t%s\n", displayUnix(item.Created), item.DisplayName, oneLine(item.Body))
	}
	return tw.Flush()
}

func runDocuments(args []string) error {
	fs := flag.NewFlagSet("documents", flag.ContinueOnError)
	sectionID := fs.Int64("section", 0, "section ID")
	jsonOut := fs.Bool("json", false, "output JSON")
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *sectionID == 0 {
		return errors.New("documents requires --section <id>")
	}
	client, err := newClientFromEnv()
	if err != nil {
		return err
	}
	items, err := client.Documents(*sectionID)
	if err != nil {
		return err
	}
	if *jsonOut {
		return writeJSON(items)
	}
	tw := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "DOCUMENT_ID\tTITLE\tURL")
	for _, item := range items {
		fmt.Fprintf(tw, "%d\t%s\t%s\n", item.ID, item.Title, item.URL)
	}
	return tw.Flush()
}

func runPages(args []string) error {
	fs := flag.NewFlagSet("pages", flag.ContinueOnError)
	sectionID := fs.Int64("section", 0, "section ID")
	jsonOut := fs.Bool("json", false, "output JSON")
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *sectionID == 0 {
		return errors.New("pages requires --section <id>")
	}
	client, err := newClientFromEnv()
	if err != nil {
		return err
	}
	items, err := client.Pages(*sectionID)
	if err != nil {
		return err
	}
	if *jsonOut {
		return writeJSON(items)
	}
	tw := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "PAGE_ID\tTITLE\tPUBLISHED\tAVAILABLE")
	for _, item := range items {
		fmt.Fprintf(tw, "%d\t%s\t%d\t%d\n", item.ID, item.Title, item.Published, item.Available)
	}
	return tw.Flush()
}

func runAudit(command string, args []string) error {
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	days := fs.Int("days", 14, "include assignments due through N days from now")
	course := fs.String("course", "", "filter course or section title (case-insensitive)")
	includeOverdue := fs.Bool("include-overdue", true, "include overdue assignments")
	jsonOut := fs.Bool("json", false, "output JSON")
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *days < 0 {
		return errors.New("--days must be zero or greater")
	}
	client, err := newClientFromEnv()
	if err != nil {
		return err
	}
	result, err := client.Audit(*days, *course, *includeOverdue)
	if err != nil {
		return err
	}
	if command == "todos" {
		filtered := result.Items[:0]
		for _, item := range result.Items {
			if item.Actionable {
				filtered = append(filtered, item)
			}
		}
		result.Items = filtered
	}
	if *jsonOut {
		return writeJSON(result)
	}
	tw := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "DUE\tCOURSE\tSTATUS\tTITLE\tURL")
	for _, item := range result.Items {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", displayTime(item.Due), item.Course, item.SubmissionStatus, item.Title, item.WebURL)
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
	if key == "" {
		key = keychainCredential("consumer-key")
	}
	if secret == "" {
		secret = keychainCredential("consumer-secret")
	}
	baseURL := strings.TrimSpace(os.Getenv("SCHOOLOGY_API_BASE"))
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	if key == "" || secret == "" {
		return nil, errors.New("configure SCHOOLOGY_KEY and SCHOOLOGY_SECRET or the schoology-cli macOS Keychain entries")
	}
	return &Client{
		Key:     key,
		Secret:  secret,
		BaseURL: strings.TrimRight(baseURL, "/"),
		Client:  &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func keychainCredential(account string) string {
	if _, err := exec.LookPath("security"); err != nil {
		return ""
	}
	out, err := exec.Command("security", "find-generic-password", "-s", "schoology-cli", "-a", account, "-w").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
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

func (c *Client) Assignment(sectionID, assignmentID int64) (assignment, error) {
	var out assignment
	path := fmt.Sprintf("/sections/%d/assignments/%d?with_attachments=TRUE&with_tags=TRUE", sectionID, assignmentID)
	if err := c.getJSON(path, &out); err != nil {
		return assignment{}, err
	}
	return out, nil
}

func (c *Client) Updates(sectionID int64) ([]update, error) {
	var out []update
	next := fmt.Sprintf("/sections/%d/updates?limit=200", sectionID)
	for next != "" {
		var page updatesPage
		if err := c.getJSON(next, &page); err != nil {
			return nil, err
		}
		out = append(out, page.Update...)
		next = nextPath(page.Links.Next)
	}
	return out, nil
}

func (c *Client) Documents(sectionID int64) ([]document, error) {
	var out []document
	next := fmt.Sprintf("/sections/%d/documents?limit=200", sectionID)
	for next != "" {
		var page documentsPage
		if err := c.getJSON(next, &page); err != nil {
			return nil, err
		}
		out = append(out, page.Document...)
		next = nextPath(page.Links.Next)
	}
	return out, nil
}

func (c *Client) Pages(sectionID int64) ([]page, error) {
	var out []page
	next := fmt.Sprintf("/sections/%d/pages?limit=200", sectionID)
	for next != "" {
		var response pagesPage
		if err := c.getJSON(next, &response); err != nil {
			return nil, err
		}
		out = append(out, response.Page...)
		next = nextPath(response.Links.Next)
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

func (c *Client) Submissions(sectionID, gradeItemID, userID int64) ([]revision, error) {
	var out []revision
	next := fmt.Sprintf("/sections/%d/submissions/%d/%d?limit=200", sectionID, gradeItemID, userID)
	for next != "" {
		var page revisionsPage
		if err := c.getJSON(next, &page); err != nil {
			return nil, err
		}
		out = append(out, page.Revision...)
		next = nextPath(page.Links.Next)
	}
	return out, nil
}

func hasSubmittedRevision(revisions []revision) bool {
	for _, r := range revisions {
		if r.Draft != 1 {
			return true
		}
	}
	return false
}

func submissionState(revisions []revision) (string, *revision) {
	var latest *revision
	for i := range revisions {
		if revisions[i].Draft == 1 {
			continue
		}
		if latest == nil || revisions[i].Created > latest.Created {
			copy := revisions[i]
			latest = &copy
		}
	}
	if latest != nil {
		return "submitted", latest
	}
	if len(revisions) > 0 {
		return "draft", nil
	}
	return "not_submitted", nil
}

func (c *Client) Audit(days int, courseFilter string, includeOverdue bool) (auditResult, error) {
	me, err := c.Me()
	if err != nil {
		return auditResult{}, err
	}
	sections, err := c.Sections()
	if err != nil {
		return auditResult{}, err
	}
	now := time.Now()
	cutoff := now.Add(time.Duration(days) * 24 * time.Hour)
	result := auditResult{GeneratedAt: now.Format(time.RFC3339), Days: days, CourseFilter: courseFilter, Items: []auditItem{}}
	for _, s := range sections {
		if s.Active != 1 || !sectionMatches(s, courseFilter) {
			continue
		}
		assignments, fetchErr := c.Assignments(s.ID)
		if fetchErr != nil {
			return auditResult{}, fmt.Errorf("assignments for %s: %w", s.CourseTitle, fetchErr)
		}
		for _, a := range assignments {
			if a.Due == "" {
				continue
			}
			due, parseErr := parseSchoologyTime(a.Due)
			if parseErr != nil || due.After(cutoff) || (!includeOverdue && due.Before(now)) {
				continue
			}
			item := auditItem{Course: s.CourseTitle, SectionID: s.ID, AssignmentID: a.ID, Title: a.Title, Description: a.Description, Due: a.Due, WebURL: a.WebURL}
			item.ExternalSubmission = isExternalSubmission(a)
			revisions, submissionErr := c.Submissions(s.ID, a.ID, me.ID)
			if submissionErr != nil {
				item.SubmissionStatus = "unknown"
				item.SubmissionError = submissionErr.Error()
			} else {
				item.SubmissionStatus, item.LatestSubmission = submissionState(revisions)
			}
			if item.ExternalSubmission && item.SubmissionStatus != "submitted" {
				item.SubmissionStatus = "external"
			}
			item.Actionable = item.SubmissionStatus != "submitted"
			result.Items = append(result.Items, item)
		}
	}
	sort.Slice(result.Items, func(i, j int) bool {
		if result.Items[i].Due == result.Items[j].Due {
			return result.Items[i].Course < result.Items[j].Course
		}
		return result.Items[i].Due < result.Items[j].Due
	})
	return result, nil
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
				Course:       s.CourseTitle,
				SectionID:    s.ID,
				AssignmentID: e.AssignmentID,
				Title:        e.Title,
				Due:          e.Start,
				WebURL:       e.WebURL,
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

func displayUnix(value int64) string {
	if value == 0 {
		return "-"
	}
	return time.Unix(value, 0).Format("2006-01-02 15:04")
}

func oneLine(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func sectionMatches(s section, filter string) bool {
	needle := strings.ToLower(strings.TrimSpace(filter))
	if needle == "" {
		return true
	}
	haystack := strings.ToLower(strings.Join([]string{s.CourseTitle, s.CourseCode, s.SectionTitle}, " "))
	return strings.Contains(haystack, needle)
}

func isExternalSubmission(a assignment) bool {
	text := strings.ToLower(a.Title + " " + a.Description + " " + a.WebURL)
	return strings.Contains(text, "turnitin")
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
	fmt.Fprintln(w, "  schoologyCLI sections [--all] [--course text] [--json]")
	fmt.Fprintln(w, "  schoologyCLI assignments --section <id> [--limit N] [--incomplete] [--json]")
	fmt.Fprintln(w, "  schoologyCLI assignment --section <id> --assignment <id> [--json]")
	fmt.Fprintln(w, "  schoologyCLI upcoming [--days N] [--json]")
	fmt.Fprintln(w, "  schoologyCLI submissions --section <id> --assignment <id> [--json]")
	fmt.Fprintln(w, "  schoologyCLI updates --section <id> [--limit N] [--json]")
	fmt.Fprintln(w, "  schoologyCLI documents --section <id> [--json]")
	fmt.Fprintln(w, "  schoologyCLI pages --section <id> [--json]")
	fmt.Fprintln(w, "  schoologyCLI audit [--days N] [--course text] [--include-overdue=false] [--json]")
	fmt.Fprintln(w, "  schoologyCLI todos [--days N] [--course text] [--include-overdue=false] [--json]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Environment:")
	fmt.Fprintln(w, "  SCHOOLOGY_KEY       Schoology consumer key")
	fmt.Fprintln(w, "  SCHOOLOGY_SECRET    Schoology consumer secret")
	fmt.Fprintln(w, "  SCHOOLOGY_API_BASE  Optional API base URL (default https://api.schoology.com/v1)")
	fmt.Fprintln(w, "  macOS Keychain fallback: service schoology-cli, accounts consumer-key and consumer-secret")
}
