//go:build !ignore_autogenerated

/*
MIT License

Copyright (c) 2024 ngrok, Inc.

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BindingConfiguration) DeepCopyInto(out *BindingConfiguration) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BindingConfiguration.
func (in *BindingConfiguration) DeepCopy() *BindingConfiguration {
	if in == nil {
		return nil
	}
	out := new(BindingConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *BindingConfiguration) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BindingConfigurationList) DeepCopyInto(out *BindingConfigurationList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]BindingConfiguration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BindingConfigurationList.
func (in *BindingConfigurationList) DeepCopy() *BindingConfigurationList {
	if in == nil {
		return nil
	}
	out := new(BindingConfigurationList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *BindingConfigurationList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BindingConfigurationSpec) DeepCopyInto(out *BindingConfigurationSpec) {
	*out = *in
	if in.AllowedURLs != nil {
		in, out := &in.AllowedURLs, &out.AllowedURLs
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	in.ProjectedMetadata.DeepCopyInto(&out.ProjectedMetadata)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BindingConfigurationSpec.
func (in *BindingConfigurationSpec) DeepCopy() *BindingConfigurationSpec {
	if in == nil {
		return nil
	}
	out := new(BindingConfigurationSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BindingConfigurationStatus) DeepCopyInto(out *BindingConfigurationStatus) {
	*out = *in
	if in.Endpoints != nil {
		in, out := &in.Endpoints, &out.Endpoints
		*out = make([]BindingEndpoint, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BindingConfigurationStatus.
func (in *BindingConfigurationStatus) DeepCopy() *BindingConfigurationStatus {
	if in == nil {
		return nil
	}
	out := new(BindingConfigurationStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BindingEndpoint) DeepCopyInto(out *BindingEndpoint) {
	*out = *in
	out.Ref = in.Ref
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BindingEndpoint.
func (in *BindingEndpoint) DeepCopy() *BindingEndpoint {
	if in == nil {
		return nil
	}
	out := new(BindingEndpoint)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EndpointBinding) DeepCopyInto(out *EndpointBinding) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EndpointBinding.
func (in *EndpointBinding) DeepCopy() *EndpointBinding {
	if in == nil {
		return nil
	}
	out := new(EndpointBinding)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *EndpointBinding) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EndpointBindingList) DeepCopyInto(out *EndpointBindingList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]EndpointBinding, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EndpointBindingList.
func (in *EndpointBindingList) DeepCopy() *EndpointBindingList {
	if in == nil {
		return nil
	}
	out := new(EndpointBindingList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *EndpointBindingList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EndpointBindingSpec) DeepCopyInto(out *EndpointBindingSpec) {
	*out = *in
	in.TargetMetadata.DeepCopyInto(&out.TargetMetadata)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EndpointBindingSpec.
func (in *EndpointBindingSpec) DeepCopy() *EndpointBindingSpec {
	if in == nil {
		return nil
	}
	out := new(EndpointBindingSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EndpointBindingStatus) DeepCopyInto(out *EndpointBindingStatus) {
	*out = *in
	out.BindingEndpoint = in.BindingEndpoint
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EndpointBindingStatus.
func (in *EndpointBindingStatus) DeepCopy() *EndpointBindingStatus {
	if in == nil {
		return nil
	}
	out := new(EndpointBindingStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TargetMetadata) DeepCopyInto(out *TargetMetadata) {
	*out = *in
	if in.Labels != nil {
		in, out := &in.Labels, &out.Labels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Annotations != nil {
		in, out := &in.Annotations, &out.Annotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TargetMetadata.
func (in *TargetMetadata) DeepCopy() *TargetMetadata {
	if in == nil {
		return nil
	}
	out := new(TargetMetadata)
	in.DeepCopyInto(out)
	return out
}
