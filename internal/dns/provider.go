package dns

import (
    "context"

    "dns-failover/internal/model"
)

type Provider interface {

    GetRecords(
        ctx context.Context,
        zone string,
        record string,
    ) ([]model.Record, error)

    UpdateRecords(
        ctx context.Context,
        zone string,
        records []model.Record,
    ) error
}