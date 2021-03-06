package fs

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func TestNewFilterDefault(t *testing.T) {
	f, err := NewFilter()
	if err != nil {
		t.Fatal(err)
	}
	if f.DeleteExcluded != false {
		t.Errorf("DeleteExcluded want false got %v", f.DeleteExcluded)
	}
	if f.MinSize != 0 {
		t.Errorf("MinSize want 0 got %v", f.MinSize)
	}
	if f.MaxSize != 0 {
		t.Errorf("MaxSize want 0 got %v", f.MaxSize)
	}
	if len(f.rules) != 0 {
		t.Errorf("rules want non got %v", f.rules)
	}
	if f.files != nil {
		t.Errorf("files want none got %v", f.files)
	}
}

// return a pointer to the string
func stringP(s string) *string {
	return &s
}

// testFile creates a temp file with the contents
func testFile(t *testing.T, contents string) *string {
	out, err := ioutil.TempFile("", "filter_test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := out.Close()
		if err != nil {
			t.Error(err)
		}
	}()
	_, err = out.Write([]byte(contents))
	if err != nil {
		t.Fatal(err)
	}
	s := out.Name()
	return &s
}

func TestNewFilterFull(t *testing.T) {
	mins := int64(100 * 1024)
	maxs := int64(1000 * 1024)
	emptyString := ""
	isFalse := false
	isTrue := true

	// Set up the input
	deleteExcluded = &isTrue
	filterRule = stringP("- filter1")
	filterFrom = testFile(t, "#comment\n+ filter2\n- filter3\n")
	excludeRule = stringP("exclude1")
	excludeFrom = testFile(t, "#comment\nexclude2\nexclude3\n")
	includeRule = stringP("include1")
	includeFrom = testFile(t, "#comment\ninclude2\ninclude3\n")
	filesFrom = testFile(t, "#comment\nfiles1\nfiles2\n")
	minSize = SizeSuffix(mins)
	maxSize = SizeSuffix(maxs)

	rm := func(p string) {
		err := os.Remove(p)
		if err != nil {
			t.Logf("error removing %q: %v", p, err)
		}
	}
	// Reset the input
	defer func() {
		rm(*filterFrom)
		rm(*excludeFrom)
		rm(*includeFrom)
		rm(*filesFrom)
		minSize = 0
		maxSize = 0
		deleteExcluded = &isFalse
		filterRule = &emptyString
		filterFrom = &emptyString
		excludeRule = &emptyString
		excludeFrom = &emptyString
		includeRule = &emptyString
		includeFrom = &emptyString
		filesFrom = &emptyString
	}()

	f, err := NewFilter()
	if err != nil {
		t.Fatal(err)
	}
	if f.DeleteExcluded != true {
		t.Errorf("DeleteExcluded want true got %v", f.DeleteExcluded)
	}
	if f.MinSize != mins {
		t.Errorf("MinSize want %v got %v", mins, f.MinSize)
	}
	if f.MaxSize != maxs {
		t.Errorf("MaxSize want %v got %v", maxs, f.MaxSize)
	}
	got := f.DumpFilters()
	want := `+ (^|/)include1$
- (^|/)[^/]*$
+ (^|/)include2$
+ (^|/)include3$
- (^|/)[^/]*$
- (^|/)exclude1$
- (^|/)exclude2$
- (^|/)exclude3$
- (^|/)filter1$
+ (^|/)filter2$
- (^|/)filter3$`
	if got != want {
		t.Errorf("rules want %s got %s", want, got)
	}
	if len(f.files) != 2 {
		t.Errorf("files want 2 got %v", f.files)
	}
	for _, name := range []string{"files1", "files2"} {
		_, ok := f.files[name]
		if !ok {
			t.Errorf("Didn't find file %q in f.files", name)
		}
	}
}

type includeTest struct {
	in   string
	size int64
	want bool
}

func testInclude(t *testing.T, f *Filter, tests []includeTest) {
	for _, test := range tests {
		got := f.Include(test.in, test.size)
		if test.want != got {
			t.Errorf("%q,%d: want %v got %v", test.in, test.size, test.want, got)
		}
	}
}

func TestNewFilterIncludeFiles(t *testing.T) {
	f, err := NewFilter()
	if err != nil {
		t.Fatal(err)
	}
	err = f.AddFile("file1.jpg")
	if err != nil {
		t.Error(err)
	}
	err = f.AddFile("/file2.jpg")
	if err != nil {
		t.Error(err)
	}
	testInclude(t, f, []includeTest{
		{"file1.jpg", 0, true},
		{"file2.jpg", 1, true},
		{"potato/file2.jpg", 2, false},
		{"file3.jpg", 3, false},
	})
}

func TestNewFilterMinSize(t *testing.T) {
	f, err := NewFilter()
	if err != nil {
		t.Fatal(err)
	}
	f.MinSize = 100
	testInclude(t, f, []includeTest{
		{"file1.jpg", 100, true},
		{"file2.jpg", 101, true},
		{"potato/file2.jpg", 99, false},
	})
}

func TestNewFilterMaxSize(t *testing.T) {
	f, err := NewFilter()
	if err != nil {
		t.Fatal(err)
	}
	f.MaxSize = 100
	testInclude(t, f, []includeTest{
		{"file1.jpg", 100, true},
		{"file2.jpg", 101, false},
		{"potato/file2.jpg", 99, true},
	})
}

func TestNewFilterMatches(t *testing.T) {
	f, err := NewFilter()
	if err != nil {
		t.Fatal(err)
	}
	add := func(s string) {
		err := f.AddRule(s)
		if err != nil {
			t.Fatal(err)
		}
	}
	add("+ cleared")
	add("!")
	add("- file1.jpg")
	add("+ file2.png")
	add("+ *.jpg")
	add("- *.png")
	add("- /potato")
	add("+ /sausage1")
	add("+ /sausage2*")
	add("+ /sausage3**")
	add("- *")
	testInclude(t, f, []includeTest{
		{"cleared", 100, false},
		{"file1.jpg", 100, false},
		{"file2.png", 100, true},
		{"afile2.png", 100, false},
		{"file3.jpg", 101, true},
		{"file4.png", 101, false},
		{"potato", 101, false},
		{"sausage1", 101, true},
		{"sausage1/potato", 101, false},
		{"sausage2potato", 101, true},
		{"sausage2/potato", 101, false},
		{"sausage3/potato", 101, true},
		{"unicorn", 99, false},
	})
}

func TestFilterForEachLine(t *testing.T) {
	file := testFile(t, `; comment
one
# another comment


two
 # indented comment
three  
four    
five
  six  `)
	defer func() {
		err := os.Remove(*file)
		if err != nil {
			t.Error(err)
		}
	}()
	lines := []string{}
	err := forEachLine(*file, func(s string) error {
		lines = append(lines, s)
		return nil
	})
	if err != nil {
		t.Error(err)
	}
	got := strings.Join(lines, ",")
	want := "one,two,three,four,five,six"
	if want != got {
		t.Errorf("want %q got %q", want, got)
	}
}

func TestFilterMatchesFromDocs(t *testing.T) {
	for _, test := range []struct {
		glob     string
		included bool
		file     string
	}{
		{"file.jpg", true, "file.jpg"},
		{"file.jpg", true, "directory/file.jpg"},
		{"file.jpg", false, "afile.jpg"},
		{"file.jpg", false, "directory/afile.jpg"},
		{"/file.jpg", true, "file.jpg"},
		{"/file.jpg", false, "afile.jpg"},
		{"/file.jpg", false, "directory/file.jpg"},
		{"*.jpg", true, "file.jpg"},
		{"*.jpg", true, "directory/file.jpg"},
		{"*.jpg", false, "file.jpg/anotherfile.png"},
		{"dir/**", true, "dir/file.jpg"},
		{"dir/**", true, "dir/dir1/dir2/file.jpg"},
		{"dir/**", false, "directory/file.jpg"},
		{"dir/**", false, "adir/file.jpg"},
		{"l?ss", true, "less"},
		{"l?ss", true, "lass"},
		{"l?ss", false, "floss"},
		{"h[ae]llo", true, "hello"},
		{"h[ae]llo", true, "hallo"},
		{"h[ae]llo", false, "hullo"},
		{"{one,two}_potato", true, "one_potato"},
		{"{one,two}_potato", true, "two_potato"},
		{"{one,two}_potato", false, "three_potato"},
		{"{one,two}_potato", false, "_potato"},
		{"\\*.jpg", true, "*.jpg"},
		{"\\\\.jpg", true, "\\.jpg"},
		{"\\[one\\].jpg", true, "[one].jpg"},
	} {
		f, err := NewFilter()
		if err != nil {
			t.Fatal(err)
		}
		err = f.Add(true, test.glob)
		if err != nil {
			t.Fatal(err)
		}
		err = f.Add(false, "*")
		if err != nil {
			t.Fatal(err)
		}
		included := f.Include(test.file, 0)
		if included != test.included {
			t.Logf("%q match %q: want %v got %v", test.glob, test.file, test.included, included)
		}
	}
}
