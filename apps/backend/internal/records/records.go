package records

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const dateLayout = "02 Jan 2006 15:04"

var (
	filePattern = regexp.MustCompile(`^repository_.*_data\.json$`)
	yearPattern = regexp.MustCompile(`(?:^|_)(\d{4})_data\.json$`)
)

type File struct {
	Path string
	Year int
}

type Source struct {
	Title          string   `json:"title"`
	Abstract       *string  `json:"abstract"`
	Authors        []string `json:"authors"`
	ItemType       *string  `json:"item_type"`
	Subjects       *string  `json:"subjects"`
	Divisions      *string  `json:"divisions"`
	DepositingUser *string  `json:"depositing_user"`
	DateDeposited  *string  `json:"date_deposited"`
	URI            string   `json:"uri"`
}

type Document struct {
	URI            string
	SourceYear     int
	Title          string
	Abstract       *string
	Authors        []string
	ItemType       *string
	Subjects       *string
	Divisions      *string
	DepositingUser *string
	DateDeposited  *time.Time
	SearchText     string
}

func Discover(root string) ([]File, error) {
	var files []File
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !filePattern.MatchString(entry.Name()) {
			return nil
		}

		match := yearPattern.FindStringSubmatch(entry.Name())
		if match == nil {
			return fmt.Errorf("%s: filename must end with a four-digit year before _data.json", path)
		}
		year, err := strconv.Atoi(match[1])
		if err != nil || year < 1900 || year > 2100 {
			return fmt.Errorf("%s: source year must be between 1900 and 2100", path)
		}
		files = append(files, File{Path: path, Year: year})
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, errors.New("no repository_*_data.json files found")
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

func Read(file File) ([]Document, int, error) {
	contents, err := os.ReadFile(file.Path)
	if err != nil {
		return nil, 0, fmt.Errorf("%s: %w", file.Path, err)
	}
	var rawRecords []json.RawMessage
	if err := json.Unmarshal(contents, &rawRecords); err != nil {
		return nil, 0, fmt.Errorf("%s: %w", file.Path, err)
	}

	documents := make([]Document, 0, len(rawRecords))
	for index, raw := range rawRecords {
		var source Source
		if err := json.Unmarshal(raw, &source); err != nil {
			return nil, len(rawRecords), fmt.Errorf("%s[%d]: %w", file.Path, index, err)
		}
		document, err := Normalize(source, file.Year)
		if err != nil {
			return nil, len(rawRecords), fmt.Errorf("%s[%d]: %w", file.Path, index, err)
		}
		documents = append(documents, document)
	}
	return documents, len(rawRecords), nil
}

func Normalize(source Source, year int) (Document, error) {
	title := normalize(source.Title)
	if title == "" {
		return Document{}, errors.New("title is required")
	}
	uri := normalize(source.URI)
	if uri == "" {
		return Document{}, errors.New("uri is required")
	}

	authors := make([]string, 0, len(source.Authors))
	for _, author := range source.Authors {
		if value := normalize(author); value != "" {
			authors = append(authors, value)
		}
	}
	abstract := normalizeOptional(source.Abstract)
	itemType := normalizeOptional(source.ItemType)
	subjects := normalizeOptional(source.Subjects)
	divisions := normalizeOptional(source.Divisions)
	depositingUser := normalizeOptional(source.DepositingUser)

	var deposited *time.Time
	if value := normalizeOptional(source.DateDeposited); value != nil {
		parsed, err := time.ParseInLocation(dateLayout, *value, time.UTC)
		if err != nil {
			return Document{}, fmt.Errorf("date_deposited must match %s", dateLayout)
		}
		deposited = &parsed
	}

	searchParts := []string{title, title}
	searchParts = append(searchParts, authors...)
	for _, value := range []*string{subjects, divisions, abstract} {
		if value != nil {
			searchParts = append(searchParts, *value)
		}
	}

	return Document{
		URI:            uri,
		SourceYear:     year,
		Title:          title,
		Abstract:       abstract,
		Authors:        authors,
		ItemType:       itemType,
		Subjects:       subjects,
		Divisions:      divisions,
		DepositingUser: depositingUser,
		DateDeposited:  deposited,
		SearchText:     strings.Join(searchParts, " "),
	}, nil
}

func normalize(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func normalizeOptional(value *string) *string {
	if value == nil {
		return nil
	}
	normalized := normalize(*value)
	if normalized == "" {
		return nil
	}
	return &normalized
}
