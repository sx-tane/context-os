package googledrive

// These white-box tests cover package-local helpers extracted from the Google Drive connector split.

import (
	"errors"
	"net/http"
	"reflect"
	"testing"
	"time"

	"context-os/domain/contracts"
)

// TestFolderIDFromRequestAcceptsSupportedSources verifies metadata, direct IDs, custom URIs, Drive URLs, and environment fallback resolve folder IDs.
func TestFolderIDFromRequestAcceptsSupportedSources(t *testing.T) {
	t.Setenv(googleDriveFolderIDEnv, "env-folder")

	cases := []struct {
		name     string
		uri      string
		metadata map[string]string
		want     string
	}{
		{name: "metadata", metadata: map[string]string{metadataFolderID: "metadata-folder"}, want: "metadata-folder"},
		{name: "environment", metadata: map[string]string{}, want: "env-folder"},
		{name: "direct id", uri: "direct-folder", metadata: map[string]string{}, want: "direct-folder"},
		{name: "custom uri", uri: "googledrive://folder/custom-folder", metadata: map[string]string{}, want: "custom-folder"},
		{name: "drive url", uri: "https://drive.google.com/drive/folders/url-folder", metadata: map[string]string{}, want: "url-folder"},
		{name: "query id", uri: "https://drive.google.com/open?id=query-folder", metadata: map[string]string{}, want: "query-folder"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := folderIDFromRequest(tc.uri, tc.metadata)
			if err != nil {
				t.Fatalf("folderIDFromRequest() error = %v", err)
			}
			if got != tc.want {
				t.Errorf("folderIDFromRequest() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestFormatCSVAsTableConvertsRows verifies CSV exports are normalized into tab-separated text.
func TestFormatCSVAsTableConvertsRows(t *testing.T) {
	got, err := formatCSVAsTable([]byte("name,value\nalpha,1\n\"beta,gamma\",2\n"))
	if err != nil {
		t.Fatalf("formatCSVAsTable() error = %v", err)
	}
	want := "name\tvalue\nalpha\t1\nbeta,gamma\t2"
	if got != want {
		t.Fatalf("formatCSVAsTable() = %q, want %q", got, want)
	}
}

// TestFormatSlidesAsTextSkipsEmptySlides verifies Slides JSON is rendered as slide-grouped text without empty text runs.
func TestFormatSlidesAsTextSkipsEmptySlides(t *testing.T) {
	body := []byte(`{"slides":[{"pageElements":[{"shape":{"text":{"textElements":[{"textRun":{"content":" Title \n"}},{"textRun":{"content":"\n"}}]}}}]},{"pageElements":[{}]},{"pageElements":[{"shape":{"text":{"textElements":[{"textRun":{"content":"Next"}}]}}}]}]}`)
	got, err := formatSlidesAsText(body)
	if err != nil {
		t.Fatalf("formatSlidesAsText() error = %v", err)
	}
	want := "Slide 1\nTitle\n\nSlide 3\nNext"
	if got != want {
		t.Fatalf("formatSlidesAsText() = %q, want %q", got, want)
	}
}

// TestBackoffDurationUsesRetryAfter verifies Retry-After takes precedence over exponential defaults.
func TestBackoffDurationUsesRetryAfter(t *testing.T) {
	headers := http.Header{"Retry-After": []string{"3"}}
	if got := backoffDuration(2, headers); got != 3*time.Second {
		t.Fatalf("backoffDuration() = %s, want %s", got, 3*time.Second)
	}
	if got := backoffDuration(2, nil); got != 400*time.Millisecond {
		t.Fatalf("backoffDuration() default = %s, want %s", got, 400*time.Millisecond)
	}
}

// TestClassifyGoogleErrorDistinguishesTemporaryAndPermanent verifies Google API errors map to retryable pipeline error kinds.
func TestClassifyGoogleErrorDistinguishesTemporaryAndPermanent(t *testing.T) {
	cases := []struct {
		name          string
		err           error
		wantKind      contracts.ErrorKind
		wantRetryable bool
	}{
		{name: "rate limit", err: googleAPIError{status: http.StatusTooManyRequests}, wantKind: contracts.ErrorKindTemporary, wantRetryable: true},
		{name: "server", err: googleAPIError{status: http.StatusInternalServerError}, wantKind: contracts.ErrorKindTemporary, wantRetryable: true},
		{name: "unauthorized", err: googleAPIError{status: http.StatusUnauthorized}, wantKind: contracts.ErrorKindPermanent, wantRetryable: false},
		{name: "invalid request", err: googleAPIError{status: http.StatusBadRequest}, wantKind: contracts.ErrorKindInvalidRequest, wantRetryable: false},
		{name: "credentials", err: errors.New("credentials are missing"), wantKind: contracts.ErrorKindPermanent, wantRetryable: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotKind, gotRetryable := classifyGoogleError(tc.err)
			if gotKind != tc.wantKind {
				t.Errorf("classifyGoogleError() kind = %q, want %q", gotKind, tc.wantKind)
			}
			if gotRetryable != tc.wantRetryable {
				t.Errorf("classifyGoogleError() retryable = %t, want %t", gotRetryable, tc.wantRetryable)
			}
		})
	}
}

// TestMetadataHelpersReturnStableValues verifies extracted metadata helpers preserve cloning, escaping, path splitting, and event identity behavior.
func TestMetadataHelpersReturnStableValues(t *testing.T) {
	metadata := map[string]string{"a": "b"}
	cloned := cloneMetadata(metadata)
	cloned["a"] = "changed"
	if metadata["a"] != "b" {
		t.Fatalf("cloneMetadata() mutated original metadata")
	}
	if got := escapeDriveQuery("team's folder"); got != "team\\'s folder" {
		t.Fatalf("escapeDriveQuery() = %q, want %q", got, "team\\'s folder")
	}
	if got := splitPath("/a//b/"); !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatalf("splitPath() = %#v, want %#v", got, []string{"a", "b"})
	}
	if stableEventID("file", "time") != stableEventID(" file ", " time ") {
		t.Fatalf("stableEventID() did not trim inputs before hashing")
	}
}
