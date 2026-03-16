package githubapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type AppHookDelivery struct {
	ID          int64     `json:"id"`
	GUID        string    `json:"guid"`
	DeliveredAt time.Time `json:"delivered_at"`
	Status      string    `json:"status"`
	StatusCode  int       `json:"status_code"`
	Event       string    `json:"event"`
	Action      string    `json:"action"`
	Redelivery  bool      `json:"redelivery"`
}

func (c *Client) ListAppHookDeliveriesSince(ctx context.Context, cutoff time.Time) ([]AppHookDelivery, error) {
	endpoint := "app/hook/deliveries?per_page=100"
	deliveries := []AppHookDelivery{}
	for endpoint != "" {
		page, nextEndpoint, err := c.listAppHookDeliveriesPage(ctx, endpoint)
		if err != nil {
			return nil, err
		}
		reachedCutoff := false
		for _, delivery := range page {
			if !delivery.DeliveredAt.IsZero() && delivery.DeliveredAt.Before(cutoff) {
				reachedCutoff = true
				continue
			}
			deliveries = append(deliveries, delivery)
		}
		if reachedCutoff {
			break
		}
		endpoint = nextEndpoint
	}
	return deliveries, nil
}

func (c *Client) RedeliverAppHookDelivery(ctx context.Context, deliveryID int64) error {
	resp, err := c.do(ctx, http.MethodPost, fmt.Sprintf("app/hook/deliveries/%d/attempts", deliveryID), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("redeliver app hook delivery: unexpected status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) listAppHookDeliveriesPage(ctx context.Context, endpoint string) ([]AppHookDelivery, string, error) {
	resp, err := c.do(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("GET %s: unexpected status %d", endpoint, resp.StatusCode)
	}
	var deliveries []AppHookDelivery
	if err := json.NewDecoder(resp.Body).Decode(&deliveries); err != nil {
		return nil, "", err
	}
	return deliveries, nextPageURL(resp.Header.Values("Link")), nil
}
