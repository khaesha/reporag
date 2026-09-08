package records

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDiscover(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "nested")
	if err := os.Mkdir(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		filepath.Join(root, "repository_2024_data.json"),
		filepath.Join(nested, "repository_computer_science_2025_data.json"),
		filepath.Join(root, "ignored.json"),
	} {
		if err := os.WriteFile(path, []byte("[]"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	files, err := Discover(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 || files[0].Path != filepath.Join(nested, "repository_computer_science_2025_data.json") || files[0].Year != 2025 || files[1].Path != filepath.Join(root, "repository_2024_data.json") || files[1].Year != 2024 {
		t.Fatalf("unexpected files: %+v", files)
	}
}

func TestDiscoverRejectsInvalidYear(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "repository_2201_data.json")
	if err := os.WriteFile(path, []byte("[]"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Discover(root); err == nil || !strings.Contains(err.Error(), path) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNormalize(t *testing.T) {
	abstract := "  useful\n abstract  "
	date := "17 Sep 2024 00:32"
	empty := "  "
	document, err := Normalize(Source{
		Title:         "  A   title ",
		Abstract:      &abstract,
		Authors:       []string{" Alice  Example ", " "},
		Subjects:      &empty,
		DateDeposited: &date,
		URI:           " http://example.test/1 ",
	}, 2024)
	if err != nil {
		t.Fatal(err)
	}
	if document.Title != "A title" || document.Abstract == nil || *document.Abstract != "useful abstract" {
		t.Fatalf("unexpected normalization: %+v", document)
	}
	if len(document.Authors) != 1 || document.Subjects != nil {
		t.Fatalf("unexpected optional fields: %+v", document)
	}
	if document.DateDeposited == nil || !document.DateDeposited.Equal(time.Date(2024, 9, 17, 0, 32, 0, 0, time.UTC)) {
		t.Fatalf("unexpected date: %v", document.DateDeposited)
	}
	if document.SearchText != "A title A title Alice Example useful abstract" {
		t.Fatalf("unexpected search text: %q", document.SearchText)
	}
}

func TestReadReportsRecordLocation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "repository_2024_data.json")
	contents := `[{"title":"valid","uri":"u1"},{"title":"bad","authors":"not-an-array","uri":"u2"}]`
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}

	_, count, err := Read(File{Path: path, Year: 2024})
	if err == nil || count != 2 || !strings.Contains(err.Error(), path+"[1]") {
		t.Fatalf("unexpected result: count=%d error=%v", count, err)
	}
}

func TestNormalizeRejectsRequiredFieldsAndDates(t *testing.T) {
	badDate := "2024-09-17"
	for _, source := range []Source{
		{URI: "u1"},
		{Title: "title"},
		{Title: "title", URI: "u1", DateDeposited: &badDate},
	} {
		if _, err := Normalize(source, 2024); err == nil {
			t.Fatalf("expected error for %+v", source)
		}
	}
}
