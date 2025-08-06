// SPDX-License-Identifier: MIT

package v1beta1

import (
	"reflect"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func Test_attachedPools(t *testing.T) {
	tests := []struct {
		name           string
		imageName      string
		runtimeObjects []runtime.Object
		want           int
		wantErr        bool
	}{
		{
			name:      "image is not attached to any pool",
			imageName: "existing-image",
			runtimeObjects: []runtime.Object{
				&Pool{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pool1",
						Namespace: "default",
					},
					Spec: PoolSpec{
						ImageName: "another-image",
					},
				},
			},
			want:    0,
			wantErr: false,
		},
		{
			name:      "image is attached to a pool",
			imageName: "existing-image",
			runtimeObjects: []runtime.Object{
				&Pool{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pool1",
						Namespace: "default",
					},
					Spec: PoolSpec{
						ImageName: "existing-image",
					},
				},
			},
			want:    1,
			wantErr: false,
		},
		{
			name:      "image is attached to a pool which is in deletion",
			imageName: "existing-image",
			runtimeObjects: []runtime.Object{
				&Pool{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "pool1",
						Namespace:         "default",
						DeletionTimestamp: &metav1.Time{Time: time.Now()},
						Finalizers:        []string{"finalizer.garm-operator.mercedes-benz.com"},
					},
					Spec: PoolSpec{
						ImageName: "existing-image",
					},
				},
			},
			want:    0,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Register the types with the runtime scheme
			schemeBuilder := runtime.SchemeBuilder{
				AddToScheme,
			}

			err := schemeBuilder.AddToScheme(scheme.Scheme)
			if err != nil {
				t.Fatal(err)
			}

			// Create a fake client with the provided runtime objects
			client := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithRuntimeObjects(tt.runtimeObjects...)
			c = client.Build()

			got, err := attachedPools(t.Context(), tt.imageName)
			if (err != nil) != tt.wantErr {
				t.Errorf("attachedPools() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("attachedPools() = %v, want %v", got, tt.want)
			}
		})
	}
}
