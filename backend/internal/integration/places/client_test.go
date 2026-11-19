package places

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"rantauplus/backend/internal/modules/assistant"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestGooglePlacesRequests(t *testing.T) {
	lat, lng := -7.2, 112.7
	for _, tc := range []struct {
		area, method, field string
		loc                 *assistant.Location
	}{
		{"UNAIR", "searchText", "textQuery", nil},
		{"", "searchNearby", "locationRestriction", &assistant.Location{Latitude: &lat, Longitude: &lng}},
	} {
		c := New("test")
		c.http.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if r.URL.Path != "/v1/places:"+tc.method || r.Header.Get("X-Goog-Api-Key") != "test" || body[tc.field] == nil {
				t.Fatalf("unexpected request %s %+v", r.URL, body)
			}
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"places":[{"displayName":{"text":"Warung contoh"},"priceRange":{"startPrice":{"currencyCode":"IDR","units":"20000"}}}]}`))}, nil
		})
		cards, err := c.FindFood(context.Background(), tc.area, tc.loc)
		if err != nil || len(cards) != 1 || cards[0].PriceRange.Start.Units != 20000 {
			t.Fatalf("places = %+v, %v", cards, err)
		}
	}
}
