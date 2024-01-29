package conditions

import (
	"sort"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/mercedes-benz/garm-operator/api/v1alpha1"
)

type ConditionStatusObject interface {
	SetConditions(conditions []metav1.Condition)
	GetConditions() []metav1.Condition
}

func Get(o ConditionStatusObject, t v1alpha1.ConditionType) *metav1.Condition {
	conditions := o.GetConditions()
	if conditions == nil {
		return nil
	}

	for _, condition := range conditions {
		if condition.Type == string(t) {
			return &condition
		}
	}
	return nil
}

func Set(o ConditionStatusObject, condition *metav1.Condition) {
	if o == nil || condition == nil {
		return
	}

	conditions := o.GetConditions()
	exists := false
	for i := range conditions {
		existingCondition := conditions[i]
		if existingCondition.Type == condition.Type {
			exists = true
			if !equals(&existingCondition, condition) {
				condition.LastTransitionTime = metav1.NewTime(time.Now().UTC().Truncate(time.Second))
				conditions[i] = *condition
				break
			}
			condition.LastTransitionTime = existingCondition.LastTransitionTime
			break
		}
	}

	if !exists {
		if condition.LastTransitionTime.IsZero() {
			condition.LastTransitionTime = metav1.NewTime(time.Now().UTC().Truncate(time.Second))
		}
		conditions = append(conditions, *condition)
	}

	// Sorts conditions for convenience of the consumer, i.e. kubectl.
	sort.Slice(conditions, func(i, j int) bool {
		return lexicographicLess(&conditions[i], &conditions[j])
	})

	o.SetConditions(conditions)
}

func Has(o ConditionStatusObject, t v1alpha1.ConditionType) bool {
	return Get(o, t) != nil
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
	Set(o, TrueCondition(t, reason, message))
}

func MarkFalse(o ConditionStatusObject, t v1alpha1.ConditionType, reason, message string) {
	Set(o, FalseCondition(t, reason, message))
}

func MarkUnknown(o ConditionStatusObject, t v1alpha1.ConditionType, reason, message string) {
	Set(o, UnknownCondition(t, reason, message))
}

func Remove(o ConditionStatusObject, t v1alpha1.ConditionType) {
	if o == nil {
		return
	}

	conditions := o.GetConditions()
	newConditions := make([]metav1.Condition, 0, len(conditions))
	for _, condition := range conditions {
		if condition.Type != string(t) {
			newConditions = append(newConditions, condition)
		}
	}
	o.SetConditions(newConditions)
}

// lexicographicLess returns true if a condition is less than another with regards to the
// to order of conditions designed for convenience of the consumer, i.e. kubectl.
// According to this order the Ready condition always goes first, followed by all the other
// conditions sorted by Type.
func lexicographicLess(i, j *metav1.Condition) bool {
	return (i.Type == string(v1alpha1.ReadyCondition) || i.Type < j.Type) &&
		j.Type != string(v1alpha1.ReadyCondition)
}

func equals(i, j *metav1.Condition) bool {
	return i.Type == j.Type &&
		i.Status == j.Status &&
		i.Reason == j.Reason &&
		i.Message == j.Message
}
