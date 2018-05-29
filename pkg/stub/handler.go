package stub

import (
	"context"

	"github.com/qiniu-ava/mxnet-operator/pkg/apis/qiniu/v1alpha1"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/sirupsen/logrus"
)

func NewHandler() sdk.Handler {
	return &Handler{}
}

type Handler struct {
	// Fill me
}

func (h *Handler) Handle(ctx context.Context, event sdk.Event) error {
	switch o := event.Object.(type) {
	case *v1alpha1.MXJob:
		err := h.syncMXJob(ctx, o, event.Deleted)
		if err != nil {
			logrus.Errorf("Failed to sync mxjob: %v", err)
			return err
		}
	}
	return nil
}

func (h *Handler) syncMXJob(ctx context.Context, job *v1alpha1.MXJob, deleted bool) error {
	return nil
}
