package conditions

import (
	"github.com/mercedes-benz/garm-operator/api/v1alpha1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConditionStatusObject interface {
	SetConditions(conditions []metav1.Condition)
	GetConditions() []metav1.Condition
}

func Get(o ConditionStatusObject, t v1alpha1.ConditionType) *metav1.Condition {
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
	o.SetConditions(conditions)
}

func Has(o ConditionStatusObject, t v1alpha1.ConditionType) bool {
	return Get(o, t) != nil
}

func Remove(o ConditionStatusObject, t v1alpha1.ConditionType) {
	if o == nil {
		return
	}

	conditions := o.GetConditions()
	apimeta.RemoveStatusCondition(&conditions, string(t))
	o.SetConditions(conditions)
}

func TrueCondition(t v1alpha1.ConditionType, reason, message string) *metav1.Condition {
	return &metav1.Condition{
		Type:    string(t),
		Status:  metav1.ConditionTrue,
		Reason:  reason,
		Message: message,
	}
}

func FalseCondition(t v1alpha1.ConditionType, reason, message string) *metav1.Condition {
	return &metav1.Condition{
		Type:    string(t),
		Status:  metav1.ConditionFalse,
		Reason:  reason,
		Message: message,
	}
}

func UnknownCondition(t v1alpha1.ConditionType, reason, message string) *metav1.Condition {
	return &metav1.Condition{
		Type:    string(t),
		Status:  metav1.ConditionUnknown,
		Reason:  reason,
		Message: message,
	}
}

func MarkTrue(o ConditionStatusObject, t v1alpha1.ConditionType, reason, message string) {
	Set(o, *TrueCondition(t, reason, message))
}

func MarkFalse(o ConditionStatusObject, t v1alpha1.ConditionType, reason, message string) {
	Set(o, *FalseCondition(t, reason, message))
}

func MarkUnknown(o ConditionStatusObject, t v1alpha1.ConditionType, reason, message string) {
	Set(o, *UnknownCondition(t, reason, message))
}
