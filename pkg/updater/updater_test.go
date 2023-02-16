package updater

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckUpdate(t *testing.T) {
	type updateInputs struct {
		buildStream    string
		currentVersion string
		checkStream    string
	}
	cases := []struct {
		name string
		cli  updateInputs
		// serverResponse is the server's answer for the latest version, or empty string to represent an error
		serverResponse string
		expectUpdate   bool
		expectError    bool
	}{
		{
			name: `same stream, server has same version`,
			cli: updateInputs{
				buildStream:    `open:latest`,
				currentVersion: `0.5.0`,
				checkStream:    `open:latest`,
			},
			serverResponse: `0.5.0`,
			expectUpdate:   false,
		},
		{
			name: `same stream, server has newer version`,
			cli: updateInputs{
				buildStream:    `open:latest`,
				currentVersion: `0.5.0`,
				checkStream:    `open:latest`,
			},
			serverResponse: `0.5.1`,
			expectUpdate:   true,
		},
		{
			name: `(weird case) same stream, server has older version`, // unexpected, but should downgrade, I guess?
			cli: updateInputs{
				buildStream:    `open:latest`,
				currentVersion: `0.5.1`,
				checkStream:    `open:latest`,
			},
			serverResponse: `0.5.0`,
			expectUpdate:   true,
		},
		{
			name: `different stream, server has newer version`,
			cli: updateInputs{
				buildStream:    `open:latest`,
				currentVersion: `0.5.0`,
				checkStream:    `pro:latest`,
			},
			serverResponse: `0.5.1`,
			expectUpdate:   true,
		},
		{
			name: `different stream, server has same version`,
			cli: updateInputs{
				buildStream:    `open:latest`,
				currentVersion: `0.5.0`,
				checkStream:    `pro:latest`,
			},
			serverResponse: `0.5.0`,
			expectUpdate:   true, // both are 0.5.0, but we want to update to switch editions
		},
		{
			name: `different stream, server has older version`,
			cli: updateInputs{
				buildStream:    `open:latest`,
				currentVersion: `0.5.1`,
				checkStream:    `pro:latest`,
			},
			serverResponse: `0.5.1`,
			expectUpdate:   true,
		},
		{
			name: `pin to same stream, same version`,
			cli: updateInputs{
				buildStream:    `open:latest`,
				currentVersion: `0.5.0`,
				checkStream:    `open:v0.5.0`,
			},
			serverResponse: `0.5.0`,
			// it's important that this is false, or else the user will keep getting messages to the effect of
			// "you're on v0.1.2, but there's a new version, v0.1.2".
			expectUpdate: false,
		},
		{
			name: `pin to same stream, newer version`,
			cli: updateInputs{
				buildStream:    `open:latest`,
				currentVersion: `0.5.0`,
				checkStream:    `open:v0.5.1`,
			},
			serverResponse: `0.5.1`,
			expectUpdate:   true,
		},
		{
			name: `pin to same stream, older version`,
			cli: updateInputs{
				buildStream:    `open:latest`,
				currentVersion: `0.5.1`,
				checkStream:    `open:v0.5.0`,
			},
			serverResponse: `0.5.0`,
			expectUpdate:   true,
		},
		{
			name: `pin to different stream, same version`,
			cli: updateInputs{
				buildStream:    `open:latest`,
				currentVersion: `0.5.0`,
				checkStream:    `pro:v0.5.0`,
			},
			serverResponse: `0.5.0`,
			expectUpdate:   true,
		},
		{
			name: `pin to different stream, newer version`,
			cli: updateInputs{
				buildStream:    `open:latest`,
				currentVersion: `0.5.0`,
				checkStream:    `pro:v0.5.1`,
			},
			serverResponse: `0.5.1`,
			expectUpdate:   true,
		},
		{
			name: `pin to different stream, older version`,
			cli: updateInputs{
				buildStream:    `open:latest`,
				currentVersion: `0.5.1`,
				checkStream:    `pro:v0.5.0`,
			},
			serverResponse: `0.5.0`,
			expectUpdate:   true,
		},
		{
			name: `pin to non-existent version`,
			cli: updateInputs{
				buildStream:    `open:latest`,
				currentVersion: `0.5.1`,
				checkStream:    `open:v0.5.0`,
			},
			expectError: true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			handler := interactions{assert: assert}
			rr := requestResponse{
				inUri:     `/update/check-latest-version?stream=` + tt.cli.checkStream,
				inMethod:  http.MethodGet,
				outStatus: http.StatusInternalServerError, // will be overwritten if serverResponse != ""
			}
			if tt.serverResponse != "" {
				rr.outStatus = http.StatusOK
				rr.outBody = map[string]string{`latest_version`: tt.serverResponse}
			}
			handler.interactions = append(handler.interactions, rr)
			server := httptest.NewServer(&handler)
			defer server.Close()

			updater := Updater{
				ServerURL:     server.URL,
				Stream:        tt.cli.checkStream,
				CurrentStream: tt.cli.buildStream,
			}
			needsUpdate, e := updater.CheckUpdate(tt.cli.currentVersion)

			if t.Failed() {
				return // this means the mocked server failed (ie ran out of request-responses)
			}
			if !assert.Empty(handler.interactions, "didn't see all expected interactions") {
				return
			}
			if tt.expectError {
				assert.Error(e)
			} else {
				if assert.NoError(e) {
					assert.Equal(needsUpdate, tt.expectUpdate)
				}
			}
		})
	}
}

type (
	requestResponse struct {
		inMethod  string
		inUri     string
		outStatus int
		outBody   any
	}

	interactions struct {
		assert       *assert.Assertions
		interactions []requestResponse
	}
)

func (s *interactions) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if len(s.interactions) == 0 {
		s.assert.Fail("no interactions left")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	curr := s.interactions[0]
	s.interactions = s.interactions[1:]

	if s.assert.Equal(curr.inMethod, r.Method) && s.assert.Equal(curr.inUri, r.URL.RequestURI()) {
		body, err := json.Marshal(curr.outBody)
		if !s.assert.NoError(err) {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(curr.outStatus)
		_, err = w.Write(body)
		s.assert.NoError(err)
	} else {
		s.assert.Fail("no interactions left")
		w.WriteHeader(http.StatusInternalServerError)
	}
}
