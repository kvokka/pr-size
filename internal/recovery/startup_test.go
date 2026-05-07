package recovery

import (
	"bytes"
	"context"
	"errors"
	"log"
	"strings"
	"testing"
	"time"

	"pr-size-labeler/internal/config"
	"pr-size-labeler/internal/githubapi"
)

type fakeAppTokenSource struct {
	token string
	err   error
}

func (f fakeAppTokenSource) AppToken(context.Context) (string, error) {
	return f.token, f.err
}

type fakeDeliveryClient struct {
	deliveries            []githubapi.AppHookDelivery
	installations         []githubapi.AppInstallation
	listErr               error
	listInstallationsErr  error
	redeliveryErrByID     map[int64]error
	requestedCutoff       time.Time
	redeliveredDeliveries []int64
}

func (f *fakeDeliveryClient) ListAppHookDeliveriesSince(_ context.Context, cutoff time.Time) ([]githubapi.AppHookDelivery, error) {
	f.requestedCutoff = cutoff
	return f.deliveries, f.listErr
}

func (f *fakeDeliveryClient) RedeliverAppHookDelivery(_ context.Context, deliveryID int64) error {
	f.redeliveredDeliveries = append(f.redeliveredDeliveries, deliveryID)
	if err, ok := f.redeliveryErrByID[deliveryID]; ok {
		return err
	}
	return nil
}

func (f *fakeDeliveryClient) ListAppInstallations(context.Context) ([]githubapi.AppInstallation, error) {
	return f.installations, f.listInstallationsErr
}

type fakeInstallationRecoverer struct {
	recovered []int64
	errByID   map[int64]error
}

func (f *fakeInstallationRecoverer) RecoverInstallation(_ context.Context, installationID int64) error {
	f.recovered = append(f.recovered, installationID)
	if err, ok := f.errByID[installationID]; ok {
		return err
	}
	return nil
}

func TestStartupRecoveryDisabled(t *testing.T) {
	client := &fakeDeliveryClient{}
	var logs bytes.Buffer
	runner := NewStartupRecovery(log.New(&logs, "", 0), fakeAppTokenSource{token: "app-jwt"}, func(token string) AppClient {
		if token != "app-jwt" {
			t.Fatalf("unexpected token %q", token)
		}
		return client
	}, nil)

	err := runner.Run(context.Background(), config.Env{StartupFailedDeliveryRecoveryEnabled: false, StartupFailedDeliveryRecoveryLookback: 2 * time.Hour})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(client.redeliveredDeliveries) != 0 {
		t.Fatalf("expected no redeliveries, got %v", client.redeliveredDeliveries)
	}
	if !strings.Contains(logs.String(), "startup_failed_delivery_recovery enabled=false") {
		t.Fatalf("expected disabled log, got %s", logs.String())
	}
}

func TestStartupRecoveryRedeliversFailedOnly(t *testing.T) {
	now := time.Date(2026, 3, 16, 12, 0, 0, 0, time.UTC)
	client := &fakeDeliveryClient{
		deliveries: []githubapi.AppHookDelivery{
			{ID: 1, Status: "OK", Event: "pull_request", DeliveredAt: now.Add(-30 * time.Minute)},
			{ID: 2, Status: "INVALID_HTTP_RESPONSE", Event: "pull_request", Action: "opened", DeliveredAt: now.Add(-20 * time.Minute)},
			{ID: 3, Status: "TIMEOUT", Event: "pull_request", Action: "synchronize", DeliveredAt: now.Add(-10 * time.Minute)},
		},
		redeliveryErrByID: map[int64]error{3: errors.New("boom")},
	}
	var logs bytes.Buffer
	runner := NewStartupRecovery(log.New(&logs, "", 0), fakeAppTokenSource{token: "app-jwt"}, func(token string) AppClient {
		if token != "app-jwt" {
			t.Fatalf("unexpected token %q", token)
		}
		return client
	}, nil)
	runner.now = func() time.Time { return now }

	err := runner.Run(context.Background(), config.Env{StartupFailedDeliveryRecoveryEnabled: true, StartupFailedDeliveryRecoveryLookback: 2 * time.Hour, LogPrivateDetails: true})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if got := client.requestedCutoff; !got.Equal(now.Add(-2 * time.Hour)) {
		t.Fatalf("cutoff = %s, want %s", got, now.Add(-2*time.Hour))
	}
	if len(client.redeliveredDeliveries) != 2 || client.redeliveredDeliveries[0] != 2 || client.redeliveredDeliveries[1] != 3 {
		t.Fatalf("unexpected redeliveries: %v", client.redeliveredDeliveries)
	}
	for _, want := range []string{
		"startup_failed_delivery_recovery enabled=true lookback=2h0m0s",
		"delivery_id=2",
		"delivery_id=3",
		"redelivery_success=false error=boom",
		"summary listed=3 failed=2 redelivered=1 redelivery_failures=1",
	} {
		if !strings.Contains(logs.String(), want) {
			t.Fatalf("expected logs to contain %q, got %s", want, logs.String())
		}
	}
}

func TestStartupRecoveryRedactsDeliveryDetailsByDefault(t *testing.T) {
	now := time.Date(2026, 3, 16, 12, 0, 0, 0, time.UTC)
	client := &fakeDeliveryClient{
		deliveries: []githubapi.AppHookDelivery{{ID: 2, Status: "INVALID_HTTP_RESPONSE", Event: "pull_request", Action: "opened", DeliveredAt: now.Add(-20 * time.Minute)}},
	}
	var logs bytes.Buffer
	runner := NewStartupRecovery(log.New(&logs, "", 0), fakeAppTokenSource{token: "app-jwt"}, func(token string) AppClient {
		if token != "app-jwt" {
			t.Fatalf("unexpected token %q", token)
		}
		return client
	}, nil)
	runner.now = func() time.Time { return now }

	err := runner.Run(context.Background(), config.Env{StartupFailedDeliveryRecoveryEnabled: true, StartupFailedDeliveryRecoveryLookback: 2 * time.Hour, LogPrivateDetails: false})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	for _, want := range []string{
		"startup_failed_delivery_recovery enabled=true",
		"startup_failed_delivery_recovery redelivery_success=true",
		"summary listed=1 failed=1 redelivered=1 redelivery_failures=0",
	} {
		if !strings.Contains(logs.String(), want) {
			t.Fatalf("expected logs to contain %q, got %s", want, logs.String())
		}
	}
	for _, forbidden := range []string{"lookback=", "cutoff=", "delivery_id=2", `event="pull_request"`, `action="opened"`, "status=", "delivered_at="} {
		if strings.Contains(logs.String(), forbidden) {
			t.Fatalf("expected logs not to contain %q, got %s", forbidden, logs.String())
		}
	}
}

func TestStartupRecoveryRecoversInstallationsWhenNoDeliveriesWereListed(t *testing.T) {
	client := &fakeDeliveryClient{
		installations: []githubapi.AppInstallation{{ID: 42}, {ID: 84}},
	}
	recoverer := &fakeInstallationRecoverer{errByID: map[int64]error{84: errors.New("recover boom")}}
	var logs bytes.Buffer
	runner := NewStartupRecovery(log.New(&logs, "", 0), fakeAppTokenSource{token: "app-jwt"}, func(token string) AppClient {
		if token != "app-jwt" {
			t.Fatalf("unexpected token %q", token)
		}
		return client
	}, recoverer)

	err := runner.Run(context.Background(), config.Env{StartupFailedDeliveryRecoveryEnabled: true, StartupFailedDeliveryRecoveryLookback: 2 * time.Hour, LogPrivateDetails: true})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(recoverer.recovered) != 2 || recoverer.recovered[0] != 42 || recoverer.recovered[1] != 84 {
		t.Fatalf("unexpected recovered installations: %v", recoverer.recovered)
	}
	for _, want := range []string{
		"startup_failed_delivery_recovery summary listed=0 failed=0 redelivered=0 redelivery_failures=0",
		"startup_installation_recovery installation_id=42 recovered=true",
		"startup_installation_recovery installation_id=84 recovered=false error=recover boom",
		"startup_installation_recovery summary listed=2 recovered=1 failures=1",
	} {
		if !strings.Contains(logs.String(), want) {
			t.Fatalf("expected logs to contain %q, got %s", want, logs.String())
		}
	}
}

func TestStartupRecoveryStillRecoversInstallationsWhenDeliveryListingFails(t *testing.T) {
	client := &fakeDeliveryClient{
		installations: []githubapi.AppInstallation{{ID: 42}},
		listErr:       errors.New("deliveries unavailable"),
	}
	recoverer := &fakeInstallationRecoverer{}
	var logs bytes.Buffer
	runner := NewStartupRecovery(log.New(&logs, "", 0), fakeAppTokenSource{token: "app-jwt"}, func(token string) AppClient {
		if token != "app-jwt" {
			t.Fatalf("unexpected token %q", token)
		}
		return client
	}, recoverer)

	err := runner.Run(context.Background(), config.Env{StartupFailedDeliveryRecoveryEnabled: true, StartupFailedDeliveryRecoveryLookback: 2 * time.Hour})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(recoverer.recovered) != 1 || recoverer.recovered[0] != 42 {
		t.Fatalf("unexpected recovered installations: %v", recoverer.recovered)
	}
	for _, want := range []string{
		"startup_failed_delivery_recovery skipped after error: deliveries unavailable",
		"startup_installation_recovery summary listed=1 recovered=1 failures=0",
	} {
		if !strings.Contains(logs.String(), want) {
			t.Fatalf("expected logs to contain %q, got %s", want, logs.String())
		}
	}
}
