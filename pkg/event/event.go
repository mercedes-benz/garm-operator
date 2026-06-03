// SPDX-License-Identifier: MIT

package event

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	CreatingEvent = "Creating"
	UpdatingEvent = "Updating"
	DeletingEvent = "Deleting"
	ScalingEvent  = "Scaling"
	ErrorEvent    = "Error"
	InfoEvent     = "Info"
)

func Creating(recorder events.EventRecorder, obj client.Object, action, msg string) {
	recorder.Eventf(obj, nil, corev1.EventTypeNormal, CreatingEvent, action, msg)
}

func Updating(recorder events.EventRecorder, obj client.Object, action, msg string) {
	recorder.Eventf(obj, nil, corev1.EventTypeNormal, UpdatingEvent, action, msg)
}

func Deleting(recorder events.EventRecorder, obj client.Object, action, msg string) {
	recorder.Eventf(obj, nil, corev1.EventTypeNormal, DeletingEvent, action, msg)
}

func Scaling(recorder events.EventRecorder, obj client.Object, action, msg string) {
	recorder.Eventf(obj, nil, corev1.EventTypeNormal, ScalingEvent, action, msg)
}

func Info(recorder events.EventRecorder, obj client.Object, action, msg string) {
	recorder.Eventf(obj, nil, corev1.EventTypeNormal, InfoEvent, action, msg)
}

func Error(recorder events.EventRecorder, obj client.Object, action, msg string) {
	recorder.Eventf(obj, nil, corev1.EventTypeWarning, ErrorEvent, action, msg)
}
