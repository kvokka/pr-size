package recovery

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"pr-size-labeler/internal/auth"
	"pr-size-labeler/internal/config"
	"pr-size-labeler/internal/githubapi"
)

type DeliveryClient interface {
	ListAppHookDeliveriesSince(ctx context.Context, cutoff time.Time) ([]githubapi.AppHookDelivery, error)
	RedeliverAppHookDelivery(ctx context.Context, deliveryID int64) error
}

type AppClient interface {
	DeliveryClient
	ListAppInstallations(ctx context.Context) ([]githubapi.AppInstallation, error)
}

type AppClientFactory func(token string) AppClient

type InstallationRecoverer interface {
	RecoverInstallation(ctx context.Context, installationID int64) error
}

type StartupRecovery struct {
	logger                *log.Logger
	now                   func() time.Time
	clientFactory         AppClientFactory
	appTokenSource        auth.AppTokenSource
	installationRecoverer InstallationRecoverer
}

func NewStartupRecovery(logger *log.Logger, appTokenSource auth.AppTokenSource, clientFactory AppClientFactory, installationRecoverer InstallationRecoverer) *StartupRecovery {
	if logger == nil {
		logger = log.Default()
	}
	return &StartupRecovery{
		logger:                logger,
		now:                   time.Now,
		clientFactory:         clientFactory,
		appTokenSource:        appTokenSource,
		installationRecoverer: installationRecoverer,
	}
}

func (r *StartupRecovery) Run(ctx context.Context, env config.Env) error {
	if !env.StartupFailedDeliveryRecoveryEnabled {
		r.logger.Printf("startup_failed_delivery_recovery enabled=false")
		return nil
	}
	cutoff := r.now().UTC().Add(-env.StartupFailedDeliveryRecoveryLookback)
	if env.LogPrivateDetails {
		r.logger.Printf("startup_failed_delivery_recovery enabled=true lookback=%s cutoff=%s", env.StartupFailedDeliveryRecoveryLookback, cutoff.Format(time.RFC3339))
	} else {
		r.logger.Printf("startup_failed_delivery_recovery enabled=true")
	}
	appToken, err := r.appTokenSource.AppToken(ctx)
	if err != nil {
		return fmt.Errorf("create app token for startup recovery: %w", err)
	}
	client := r.clientFactory(appToken)
	deliveries, err := client.ListAppHookDeliveriesSince(ctx, cutoff)
	if err != nil {
		if r.installationRecoverer == nil {
			return fmt.Errorf("list app hook deliveries: %w", err)
		}
		r.logger.Printf("startup_failed_delivery_recovery skipped after error: %v", err)
	} else {
		failed := 0
		redelivered := 0
		failedRedeliveries := 0
		for _, delivery := range deliveries {
			if strings.EqualFold(delivery.Status, "OK") {
				continue
			}
			failed++
			if env.LogPrivateDetails {
				r.logger.Printf("startup_failed_delivery_recovery delivery_id=%d event=%q action=%q status=%q delivered_at=%s redelivery=%t attempting_redelivery=true", delivery.ID, delivery.Event, delivery.Action, delivery.Status, delivery.DeliveredAt.Format(time.RFC3339), delivery.Redelivery)
			}
			if err := client.RedeliverAppHookDelivery(ctx, delivery.ID); err != nil {
				failedRedeliveries++
				if env.LogPrivateDetails {
					r.logger.Printf("startup_failed_delivery_recovery delivery_id=%d redelivery_success=false error=%v", delivery.ID, err)
				} else {
					r.logger.Printf("startup_failed_delivery_recovery redelivery_success=false")
				}
				continue
			}
			redelivered++
			if env.LogPrivateDetails {
				r.logger.Printf("startup_failed_delivery_recovery delivery_id=%d redelivery_success=true", delivery.ID)
			} else {
				r.logger.Printf("startup_failed_delivery_recovery redelivery_success=true")
			}
		}
		r.logger.Printf("startup_failed_delivery_recovery summary listed=%d failed=%d redelivered=%d redelivery_failures=%d", len(deliveries), failed, redelivered, failedRedeliveries)
	}
	if r.installationRecoverer != nil {
		if err := r.recoverInstallations(ctx, env, client); err != nil {
			return err
		}
	}
	return nil
}

func (r *StartupRecovery) recoverInstallations(ctx context.Context, env config.Env, client AppClient) error {
	if env.LogPrivateDetails {
		r.logger.Printf("startup_installation_recovery enabled=true")
	}
	installations, err := client.ListAppInstallations(ctx)
	if err != nil {
		return fmt.Errorf("list app installations: %w", err)
	}
	recovered := 0
	failed := 0
	for _, installation := range installations {
		if installation.ID == 0 {
			failed++
			r.logger.Printf("startup_installation_recovery skipped installation_id=0")
			continue
		}
		if env.LogPrivateDetails {
			r.logger.Printf("startup_installation_recovery installation_id=%d recovering=true", installation.ID)
		}
		if err := r.installationRecoverer.RecoverInstallation(ctx, installation.ID); err != nil {
			failed++
			if env.LogPrivateDetails {
				r.logger.Printf("startup_installation_recovery installation_id=%d recovered=false error=%v", installation.ID, err)
			} else {
				r.logger.Printf("startup_installation_recovery recovered=false")
			}
			continue
		}
		recovered++
		if env.LogPrivateDetails {
			r.logger.Printf("startup_installation_recovery installation_id=%d recovered=true", installation.ID)
		} else {
			r.logger.Printf("startup_installation_recovery recovered=true")
		}
	}
	r.logger.Printf("startup_installation_recovery summary listed=%d recovered=%d failures=%d", len(installations), recovered, failed)
	return nil
}
