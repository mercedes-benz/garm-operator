package event

import (
	"fmt"
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
	recorder.Event(obj, corev1.EventTypeNormal, CreatingEvent, fmt.Sprintf("Creating %s: %s", obj.GetObjectKind().GroupVersionKind().Kind, msg))
}

func Updating(recorder record.EventRecorder, obj client.Object, msg string) {
	recorder.Event(obj, corev1.EventTypeNormal, UpdatingEvent, fmt.Sprintf("Updating %s: %s", obj.GetObjectKind().GroupVersionKind().Kind, msg))
}

func Deleting(recorder record.EventRecorder, obj client.Object, msg string) {
	recorder.Event(obj, corev1.EventTypeNormal, DeletingEvent, fmt.Sprintf("Deleting %s: %s", obj.GetObjectKind().GroupVersionKind().Kind, msg))
}

func Info(recorder record.EventRecorder, obj client.Object, msg string) {
	recorder.Event(obj, corev1.EventTypeNormal, InfoEvent, fmt.Sprintf("Info: %s", msg))
}

func Error(recorder record.EventRecorder, obj client.Object, err error) {
	recorder.Event(obj, corev1.EventTypeWarning, ErrorEvent, fmt.Sprintf("Error: %s", err.Error()))
}
