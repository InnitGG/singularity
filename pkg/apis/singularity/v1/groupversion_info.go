// Package v1 contains API Schema definitions for the singularity v1 API group
//+kubebuilder:object:generate=true
//+groupName=singularity.innit.gg
package v1

import (
	"innit.gg/singularity/pkg/apis/singularity"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// GroupVersion is group version used to register these objects
	GroupVersion = schema.GroupVersion{Group: singularity.GroupName, Version: "v1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)
