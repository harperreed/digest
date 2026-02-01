// ABOUTME: OPML parsing and writing library for RSS feed subscriptions
// ABOUTME: Supports folders, feed management, and round-trip XML serialization

package opml

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Document represents an OPML document with a title and hierarchical outlines
type Document struct {
	Title    string
	Outlines []Outline
	feedURLs map[string]bool // URL index for O(1) lookups
}

// Outline represents a node in the OPML tree structure
// Can be either a folder (with Children) or a feed (with XMLURL)
type Outline struct {
	Text     string
	Title    string
	Type     string
	XMLURL   string
	Children []Outline
}

// Feed is a convenience struct representing a single RSS feed with folder information
type Feed struct {
	URL    string
	Title  string
	Folder string
}

// XML structs for parsing and writing OPML files
type opmlXML struct {
	XMLName xml.Name `xml:"opml"`
	Version string   `xml:"version,attr"`
	Head    headXML  `xml:"head"`
	Body    bodyXML  `xml:"body"`
}

type headXML struct {
	Title string `xml:"title"`
}

type bodyXML struct {
	Outlines []outlineXML `xml:"outline"`
}

type outlineXML struct {
	Text     string       `xml:"text,attr"`
	Title    string       `xml:"title,attr,omitempty"`
	Type     string       `xml:"type,attr,omitempty"`
	XMLURL   string       `xml:"xmlUrl,attr,omitempty"`
	Children []outlineXML `xml:"outline,omitempty"`
}

// NewDocument creates a new empty OPML document with the given title
func NewDocument(title string) *Document {
	return &Document{
		Title:    title,
		Outlines: []Outline{},
		feedURLs: make(map[string]bool),
	}
}

// Parse reads OPML data from an io.Reader and returns a Document
func Parse(r io.Reader) (*Document, error) {
	var opml opmlXML
	decoder := xml.NewDecoder(r)
	if err := decoder.Decode(&opml); err != nil {
		return nil, fmt.Errorf("failed to decode OPML: %w", err)
	}

	doc := &Document{
		Title:    opml.Head.Title,
		Outlines: make([]Outline, len(opml.Body.Outlines)),
	}

	for i, outline := range opml.Body.Outlines {
		doc.Outlines[i] = convertOutlineFromXML(outline)
	}

	doc.rebuildURLIndex()
	return doc, nil
}

// rebuildURLIndex rebuilds the feedURLs map from the current outline structure
func (d *Document) rebuildURLIndex() {
	d.feedURLs = make(map[string]bool)
	for _, feed := range d.AllFeeds() {
		d.feedURLs[feed.URL] = true
	}
}

// ensureURLIndex initializes the feedURLs map if nil (defensive programming)
func (d *Document) ensureURLIndex() {
	if d.feedURLs == nil {
		d.rebuildURLIndex()
	}
}

// ParseFile reads OPML data from a file and returns a Document
func ParseFile(path string) (*Document, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	return Parse(file)
}

// AllFeeds returns a flat list of all feeds in the document with their folder information
func (d *Document) AllFeeds() []Feed {
	feeds := make([]Feed, 0, len(d.Outlines))
	for _, outline := range d.Outlines {
		feeds = append(feeds, collectFeeds(outline, "")...)
	}
	return feeds
}

// Folders returns a list of all folder names in the document
func (d *Document) Folders() []string {
	folderSet := make(map[string]bool)
	for _, outline := range d.Outlines {
		if outline.XMLURL == "" {
			folderSet[outline.Text] = true
		}
	}

	folders := make([]string, 0, len(folderSet))
	for folder := range folderSet {
		folders = append(folders, folder)
	}
	return folders
}

// FeedsInFolder returns all feeds in a specific folder
// Pass empty string to get root-level feeds
func (d *Document) FeedsInFolder(folder string) []Feed {
	var feeds []Feed

	if folder == "" {
		// Root level feeds
		for _, outline := range d.Outlines {
			if outline.XMLURL != "" {
				feeds = append(feeds, Feed{
					URL:    outline.XMLURL,
					Title:  getOutlineTitle(outline),
					Folder: "",
				})
			}
		}
	} else {
		// Feeds in specific folder
		for _, outline := range d.Outlines {
			if outline.Text == folder && outline.XMLURL == "" {
				for _, child := range outline.Children {
					if child.XMLURL != "" {
						feeds = append(feeds, Feed{
							URL:    child.XMLURL,
							Title:  getOutlineTitle(child),
							Folder: folder,
						})
					}
				}
			}
		}
	}

	return feeds
}

// AddFolder adds a folder to the document (idempotent)
func (d *Document) AddFolder(name string) error {
	// Check if folder already exists
	for _, outline := range d.Outlines {
		if outline.Text == name && outline.XMLURL == "" {
			return nil // Already exists
		}
	}

	// Add new folder
	d.Outlines = append(d.Outlines, Outline{
		Text:     name,
		Children: []Outline{},
	})
	return nil
}

// AddFeed adds a feed to the document, optionally in a folder
// Creates the folder if it doesn't exist
// Returns an error if a feed with the same URL already exists
func (d *Document) AddFeed(url, title, folder string) error {
	// Ensure URL index is initialized
	d.ensureURLIndex()
	// O(1) check using URL index
	if d.feedURLs[url] {
		return fmt.Errorf("feed with URL %s already exists", url)
	}

	feed := Outline{
		Text:   title,
		Title:  title,
		Type:   "rss",
		XMLURL: url,
	}

	if folder == "" {
		// Add to root
		d.Outlines = append(d.Outlines, feed)
	} else {
		// Find or create folder
		folderIndex := -1
		for i, outline := range d.Outlines {
			if outline.Text == folder && outline.XMLURL == "" {
				folderIndex = i
				break
			}
		}

		if folderIndex == -1 {
			// Create folder
			d.Outlines = append(d.Outlines, Outline{
				Text:     folder,
				Children: []Outline{feed},
			})
		} else {
			// Add to existing folder
			d.Outlines[folderIndex].Children = append(d.Outlines[folderIndex].Children, feed)
		}
	}

	// Update URL index
	d.feedURLs[url] = true
	return nil
}

// MoveFeed moves a feed to a different folder
// Pass empty string for newFolder to move to root level
func (d *Document) MoveFeed(url, newFolder string) error {
	// Find the feed and get its title
	var feed *Feed
	for _, f := range d.AllFeeds() {
		if f.URL == url {
			feed = &f
			break
		}
	}

	if feed == nil {
		return fmt.Errorf("feed not found: %s", url)
	}

	// Remove from current location
	if err := d.RemoveFeed(url); err != nil {
		return fmt.Errorf("failed to remove feed: %w", err)
	}

	// Add to new location (use existing title)
	d.addFeedInternal(url, feed.Title, newFolder)

	return nil
}

// addFeedInternal adds a feed without checking for duplicates
func (d *Document) addFeedInternal(url, title, folder string) {
	d.ensureURLIndex()
	feed := Outline{
		Text:   title,
		Title:  title,
		Type:   "rss",
		XMLURL: url,
	}

	if folder == "" {
		// Add to root
		d.Outlines = append(d.Outlines, feed)
	} else {
		// Find or create folder
		folderIndex := -1
		for i, outline := range d.Outlines {
			if outline.Text == folder && outline.XMLURL == "" {
				folderIndex = i
				break
			}
		}

		if folderIndex == -1 {
			// Create folder
			d.Outlines = append(d.Outlines, Outline{
				Text:     folder,
				Children: []Outline{feed},
			})
		} else {
			// Add to existing folder
			d.Outlines[folderIndex].Children = append(d.Outlines[folderIndex].Children, feed)
		}
	}

	// Update URL index
	d.feedURLs[url] = true
}

// RemoveFeed removes a feed from the document by URL
func (d *Document) RemoveFeed(url string) error {
	d.ensureURLIndex()
	// Check root level
	for i, outline := range d.Outlines {
		if outline.XMLURL == url {
			d.Outlines = append(d.Outlines[:i], d.Outlines[i+1:]...)
			delete(d.feedURLs, url)
			return nil
		}
	}

	// Check folders
	for i, outline := range d.Outlines {
		if outline.XMLURL == "" && len(outline.Children) > 0 {
			for j, child := range outline.Children {
				if child.XMLURL == url {
					d.Outlines[i].Children = append(
						d.Outlines[i].Children[:j],
						d.Outlines[i].Children[j+1:]...,
					)
					delete(d.feedURLs, url)
					return nil
				}
			}
		}
	}

	return fmt.Errorf("feed not found: %s", url)
}

// Write writes the OPML document to an io.Writer
func (d *Document) Write(w io.Writer) error {
	opml := opmlXML{
		Version: "2.0",
		Head: headXML{
			Title: d.Title,
		},
		Body: bodyXML{
			Outlines: make([]outlineXML, len(d.Outlines)),
		},
	}

	for i, outline := range d.Outlines {
		opml.Body.Outlines[i] = convertOutlineToXML(outline)
	}

	encoder := xml.NewEncoder(w)
	encoder.Indent("", "  ")

	if _, err := w.Write([]byte(xml.Header)); err != nil {
		return fmt.Errorf("failed to write XML header: %w", err)
	}

	if err := encoder.Encode(opml); err != nil {
		return fmt.Errorf("failed to encode OPML: %w", err)
	}

	return nil
}

// WriteFile writes the OPML document to a file
func (d *Document) WriteFile(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	return d.Write(file)
}

// Helper functions

func convertOutlineFromXML(x outlineXML) Outline {
	o := Outline{
		Text:     x.Text,
		Title:    x.Title,
		Type:     x.Type,
		XMLURL:   x.XMLURL,
		Children: make([]Outline, len(x.Children)),
	}

	for i, child := range x.Children {
		o.Children[i] = convertOutlineFromXML(child)
	}

	return o
}

func convertOutlineToXML(o Outline) outlineXML {
	x := outlineXML{
		Text:     o.Text,
		Title:    o.Title,
		Type:     o.Type,
		XMLURL:   o.XMLURL,
		Children: make([]outlineXML, len(o.Children)),
	}

	for i, child := range o.Children {
		x.Children[i] = convertOutlineToXML(child)
	}

	return x
}

func collectFeeds(outline Outline, folder string) []Feed {
	var feeds []Feed

	if outline.XMLURL != "" {
		// This is a feed
		feeds = append(feeds, Feed{
			URL:    outline.XMLURL,
			Title:  getOutlineTitle(outline),
			Folder: folder,
		})
	}

	// Recurse into children
	childFolder := folder
	if outline.XMLURL == "" && len(outline.Children) > 0 {
		// This is a folder
		childFolder = outline.Text
	}

	for _, child := range outline.Children {
		feeds = append(feeds, collectFeeds(child, childFolder)...)
	}

	return feeds
}

func getOutlineTitle(outline Outline) string {
	if outline.Title != "" {
		return outline.Title
	}
	return outline.Text
}
