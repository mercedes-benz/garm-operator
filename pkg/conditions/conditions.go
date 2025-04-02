// SPDX-License-Identifier: MIT

package conditions

import (
	"sort"
	"time"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConditionStatusObject interface {
	InitializeConditions()
	SetConditions(conditions []metav1.Condition)
	GetConditions() []metav1.Condition
}

func Get(o ConditionStatusObject, t ConditionType) *metav1.Condition {
	if o == nil {
		return nil
	}

	conditions := o.GetConditions()
	if conditions == nil {
		return nil
	}

	return apimeta.FindStatusCondition(conditions, string(t))
}

func Set(o ConditionStatusObject, condition metav1.Condition) {
	if o == nil {
		return
	}

	conditions := o.GetConditions()
	apimeta.SetStatusCondition(&conditions, condition)
	sort.Slice(conditions, func(i, j int) bool {
		return lexicographicLess(&conditions[i], &conditions[j])
	})
	o.SetConditions(conditions)
}

func Has(o ConditionStatusObject, t ConditionType) bool {
	return Get(o, t) != nil
}

func Remove(o ConditionStatusObject, t ConditionType) {
	if o == nil {
		return
	}

	conditions := o.GetConditions()
	apimeta.RemoveStatusCondition(&conditions, string(t))
	o.SetConditions(conditions)
}

func TrueCondition(t ConditionType, reason ConditionReason, message string) *metav1.Condition {
	return &metav1.Condition{
		Type:    string(t),
		Status:  metav1.ConditionTrue,
		Reason:  string(reason),
		Message: message,
	}
}

func FalseCondition(t ConditionType, reason ConditionReason, message string) *metav1.Condition {
	return &metav1.Condition{
		Type:    string(t),
		Status:  metav1.ConditionFalse,
		Reason:  string(reason),
		Message: message,
	}
}

func UnknownCondition(t ConditionType, reason ConditionReason, message string) *metav1.Condition {
	return &metav1.Condition{
		Type:    string(t),
		Status:  metav1.ConditionUnknown,
		Reason:  string(reason),
		Message: message,
	}
}

func MarkTrue(o ConditionStatusObject, t ConditionType, reason ConditionReason, message string) {
	Set(o, *TrueCondition(t, reason, message))
}

func MarkFalse(o ConditionStatusObject, t ConditionType, reason ConditionReason, message string) {
	Set(o, *FalseCondition(t, reason, message))
}

func MarkUnknown(o ConditionStatusObject, t ConditionType, reason ConditionReason, message string) {
	Set(o, *UnknownCondition(t, reason, message))
}

// lexicographicLess returns true if a condition is less than another with regard to the
// to order of conditions designed for convenience of the consumer, i.e. kubectl.
// According to this order the Ready condition always goes first, followed by all the other
// conditions sorted by Type.
func lexicographicLess(i, j *metav1.Condition) bool {
	return (i.Type == string(ReadyCondition) || i.Type < j.Type) && j.Type != string(ReadyCondition)
}

func NilLastTransitionTime(o ConditionStatusObject) {
	conditions := o.GetConditions()
	for i := range conditions {
		time := metav1.NewTime(time.Now())
		time.Reset()
		conditions[i].LastTransitionTime = time
	}
	o.SetConditions(conditions)
}
