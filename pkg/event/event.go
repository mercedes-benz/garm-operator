package event

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	CreatingEvent = "Creating"
	UpdatingEvent = "Updating"
	DeletingEvent = "Deleting"
	ErrorEvent    = "Error"
	InfoEvent     = "Info"
)

func Creating(recorder record.EventRecorder, obj client.Object, msg string) {
	recorder.Event(obj, corev1.EventTypeNormal, CreatingEvent, msg)
}

func Updating(recorder record.EventRecorder, obj client.Object, msg string) {
	recorder.Event(obj, corev1.EventTypeNormal, UpdatingEvent, msg)
}

func Deleting(recorder record.EventRecorder, obj client.Object, msg string) {
	recorder.Event(obj, corev1.EventTypeNormal, DeletingEvent, msg)
}

func Info(recorder record.EventRecorder, obj client.Object, msg string) {
	recorder.Event(obj, corev1.EventTypeNormal, InfoEvent, msg)
}

func Error(recorder record.EventRecorder, obj client.Object, err error) {
	recorder.Event(obj, corev1.EventTypeWarning, ErrorEvent, err.Error())
}
