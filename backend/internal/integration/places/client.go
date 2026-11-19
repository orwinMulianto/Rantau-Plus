package places

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"rantauplus/backend/internal/modules/assistant"
)

type Client struct {
	key  string
	http *http.Client
}

func New(key string) *Client {
	return &Client{key: key, http: &http.Client{Timeout: 6 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}

func (c *Client) FindFood(ctx context.Context, area string, location *assistant.Location) ([]assistant.Place, error) {
	if c.key == "" {
		return nil, errors.New("Google Places belum dikonfigurasi")
	}
	method := "searchText"
	body := map[string]any{"languageCode": "id", "regionCode": "ID", "textQuery": "tempat makan di " + area, "pageSize": 5}
	if area == "" {
		if location == nil || location.Latitude == nil || location.Longitude == nil {
			return nil, errors.New("lokasi belum tersedia")
		}
		method = "searchNearby"
		body = map[string]any{"languageCode": "id", "regionCode": "ID", "maxResultCount": 5, "includedTypes": []string{"restaurant"},
			"rankPreference": "DISTANCE", "locationRestriction": map[string]any{"circle": map[string]any{"center": location, "radius": 2000}}}
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://places.googleapis.com/v1/places:"+method, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Goog-Api-Key", c.key)
	req.Header.Set("X-Goog-FieldMask", "places.displayName,places.formattedAddress,places.googleMapsUri,places.currentOpeningHours.openNow,places.priceLevel,places.priceRange,places.attributions")
	response, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, errors.New("Google Places gagal merespons")
	}
	var result struct {
		Places []struct {
			DisplayName         struct{ Text string }
			FormattedAddress    string
			GoogleMapsURI       string
			CurrentOpeningHours struct{ OpenNow *bool }
			PriceLevel          string
			PriceRange          *assistant.PriceRange
			Attributions        json.RawMessage
		}
	}
	if err = json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&result); err != nil {
		return nil, err
	}
	var cards []assistant.Place
	for _, p := range result.Places {
		if p.DisplayName.Text == "" {
			continue
		}
		cards = append(cards, assistant.Place{Name: p.DisplayName.Text, Address: p.FormattedAddress, MapsURL: p.GoogleMapsURI,
			OpenNow: p.CurrentOpeningHours.OpenNow, PriceLevel: p.PriceLevel, PriceRange: p.PriceRange, Attributions: p.Attributions})
	}
	return cards, nil
}
