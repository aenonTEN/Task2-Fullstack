package httpserver

import (
	"context"
	"time"

	"eaglepoint/backend/internal/persistence"
)

type sqlAuditWriter struct {
	store *persistence.Store
}

func (w *sqlAuditWriter) AppendAudit(rec auditRecord) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return w.store.AppendAudit(ctx, persistence.AuditRecord{
		ID:       rec.ID,
		TraceID:  rec.TraceID,
		ActorID:  rec.ActorID,
		Action:   rec.Action,
		Entity:   rec.Entity,
		EntityID: rec.EntityID,
		Before:   rec.Before,
		After:    rec.After,
		CreatedAt: func() time.Time {
			t, err := time.Parse(time.RFC3339, rec.CreatedAt)
			if err != nil {
				return time.Now().UTC()
			}
			return t
		}(),
	})
}

func (w *sqlAuditWriter) ListAudit() ([]auditRecord, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	recs, err := w.store.ListAudit(ctx, 100)
	if err != nil {
		return nil, err
	}
	out := make([]auditRecord, 0, len(recs))
	for _, r := range recs {
		out = append(out, auditRecord{
			ID:        r.ID,
			TraceID:   r.TraceID,
			ActorID:   r.ActorID,
			Action:    r.Action,
			Entity:    r.Entity,
			EntityID:  r.EntityID,
			Before:    r.Before,
			After:     r.After,
			CreatedAt: r.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return out, nil
}

