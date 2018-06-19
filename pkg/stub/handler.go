package stub

import (
	"context"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/qiniu-ava/mxnet-operator/pkg/apis/qiniu/v1alpha1"
	"github.com/qiniu-ava/mxnet-operator/pkg/mx"
	"github.com/sirupsen/logrus"
)

// NewHandler creates a sdk Handler.
func NewHandler() (sdk.Handler, error) {
	h, e := mx.NewHandler()
	if e != nil {
		return nil, e
	}
	return &Handler{
		mxHandler: h,
	}, nil
}

// Handler is a sdk events processor.
type Handler struct {
	mxHandler *mx.Handler
}

// Handle processes events from controller.
func (h *Handler) Handle(ctx context.Context, event sdk.Event) error {
	switch o := event.Object.(type) {
	case *v1alpha1.MXJob:
		err := h.mxHandler.Sync(ctx, o, event.Deleted)
		if err != nil {
			logrus.Errorf("Failed to sync mxjob: %v", err)
			return err
		}
	default:
		logrus.Debug("unknown object")
	}
	return nil
}
