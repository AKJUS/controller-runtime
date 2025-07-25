/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package client

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/apply"
)

var _ Reader = &unstructuredClient{}
var _ Writer = &unstructuredClient{}

type unstructuredClient struct {
	resources  *clientRestResources
	paramCodec runtime.ParameterCodec
}

// Create implements client.Client.
func (uc *unstructuredClient) Create(ctx context.Context, obj Object, opts ...CreateOption) error {
	u, ok := obj.(runtime.Unstructured)
	if !ok {
		return fmt.Errorf("unstructured client did not understand object: %T", obj)
	}

	gvk := u.GetObjectKind().GroupVersionKind()

	o, err := uc.resources.getObjMeta(obj)
	if err != nil {
		return err
	}

	createOpts := &CreateOptions{}
	createOpts.ApplyOptions(opts)

	result := o.Post().
		NamespaceIfScoped(o.namespace, o.isNamespaced()).
		Resource(o.resource()).
		Body(obj).
		VersionedParams(createOpts.AsCreateOptions(), uc.paramCodec).
		Do(ctx).
		Into(obj)

	u.GetObjectKind().SetGroupVersionKind(gvk)
	return result
}

// Update implements client.Client.
func (uc *unstructuredClient) Update(ctx context.Context, obj Object, opts ...UpdateOption) error {
	u, ok := obj.(runtime.Unstructured)
	if !ok {
		return fmt.Errorf("unstructured client did not understand object: %T", obj)
	}

	gvk := u.GetObjectKind().GroupVersionKind()

	o, err := uc.resources.getObjMeta(obj)
	if err != nil {
		return err
	}

	updateOpts := UpdateOptions{}
	updateOpts.ApplyOptions(opts)

	result := o.Put().
		NamespaceIfScoped(o.namespace, o.isNamespaced()).
		Resource(o.resource()).
		Name(o.name).
		Body(obj).
		VersionedParams(updateOpts.AsUpdateOptions(), uc.paramCodec).
		Do(ctx).
		Into(obj)

	u.GetObjectKind().SetGroupVersionKind(gvk)
	return result
}

// Delete implements client.Client.
func (uc *unstructuredClient) Delete(ctx context.Context, obj Object, opts ...DeleteOption) error {
	if _, ok := obj.(runtime.Unstructured); !ok {
		return fmt.Errorf("unstructured client did not understand object: %T", obj)
	}

	o, err := uc.resources.getObjMeta(obj)
	if err != nil {
		return err
	}

	deleteOpts := DeleteOptions{}
	deleteOpts.ApplyOptions(opts)

	return o.Delete().
		NamespaceIfScoped(o.namespace, o.isNamespaced()).
		Resource(o.resource()).
		Name(o.name).
		Body(deleteOpts.AsDeleteOptions()).
		Do(ctx).
		Error()
}

// DeleteAllOf implements client.Client.
func (uc *unstructuredClient) DeleteAllOf(ctx context.Context, obj Object, opts ...DeleteAllOfOption) error {
	if _, ok := obj.(runtime.Unstructured); !ok {
		return fmt.Errorf("unstructured client did not understand object: %T", obj)
	}

	o, err := uc.resources.getObjMeta(obj)
	if err != nil {
		return err
	}

	deleteAllOfOpts := DeleteAllOfOptions{}
	deleteAllOfOpts.ApplyOptions(opts)

	return o.Delete().
		NamespaceIfScoped(deleteAllOfOpts.ListOptions.Namespace, o.isNamespaced()).
		Resource(o.resource()).
		VersionedParams(deleteAllOfOpts.AsListOptions(), uc.paramCodec).
		Body(deleteAllOfOpts.AsDeleteOptions()).
		Do(ctx).
		Error()
}

// Patch implements client.Client.
func (uc *unstructuredClient) Patch(ctx context.Context, obj Object, patch Patch, opts ...PatchOption) error {
	if _, ok := obj.(runtime.Unstructured); !ok {
		return fmt.Errorf("unstructured client did not understand object: %T", obj)
	}

	o, err := uc.resources.getObjMeta(obj)
	if err != nil {
		return err
	}

	data, err := patch.Data(obj)
	if err != nil {
		return err
	}

	patchOpts := &PatchOptions{}
	patchOpts.ApplyOptions(opts)

	return o.Patch(patch.Type()).
		NamespaceIfScoped(o.namespace, o.isNamespaced()).
		Resource(o.resource()).
		Name(o.name).
		VersionedParams(patchOpts.AsPatchOptions(), uc.paramCodec).
		Body(data).
		Do(ctx).
		Into(obj)
}

func (uc *unstructuredClient) Apply(ctx context.Context, obj runtime.ApplyConfiguration, opts ...ApplyOption) error {
	unstructuredApplyConfig, ok := obj.(*unstructuredApplyConfiguration)
	if !ok {
		return fmt.Errorf("bug: unstructured client got an applyconfiguration that was not %T but %T", &unstructuredApplyConfiguration{}, obj)
	}
	o, err := uc.resources.getObjMeta(unstructuredApplyConfig.Unstructured)
	if err != nil {
		return err
	}

	req, err := apply.NewRequest(o, obj)
	if err != nil {
		return fmt.Errorf("failed to create apply request: %w", err)
	}
	applyOpts := &ApplyOptions{}
	applyOpts.ApplyOptions(opts)

	return req.
		NamespaceIfScoped(o.namespace, o.isNamespaced()).
		Resource(o.resource()).
		Name(o.name).
		VersionedParams(applyOpts.AsPatchOptions(), uc.paramCodec).
		Do(ctx).
		Into(unstructuredApplyConfig.Unstructured)
}

// Get implements client.Client.
func (uc *unstructuredClient) Get(ctx context.Context, key ObjectKey, obj Object, opts ...GetOption) error {
	u, ok := obj.(runtime.Unstructured)
	if !ok {
		return fmt.Errorf("unstructured client did not understand object: %T", obj)
	}

	gvk := u.GetObjectKind().GroupVersionKind()

	getOpts := GetOptions{}
	getOpts.ApplyOptions(opts)

	r, err := uc.resources.getResource(obj)
	if err != nil {
		return err
	}

	result := r.Get().
		NamespaceIfScoped(key.Namespace, r.isNamespaced()).
		Resource(r.resource()).
		VersionedParams(getOpts.AsGetOptions(), uc.paramCodec).
		Name(key.Name).
		Do(ctx).
		Into(obj)

	u.GetObjectKind().SetGroupVersionKind(gvk)

	return result
}

// List implements client.Client.
func (uc *unstructuredClient) List(ctx context.Context, obj ObjectList, opts ...ListOption) error {
	u, ok := obj.(runtime.Unstructured)
	if !ok {
		return fmt.Errorf("unstructured client did not understand object: %T", obj)
	}

	gvk := u.GetObjectKind().GroupVersionKind()
	gvk.Kind = strings.TrimSuffix(gvk.Kind, "List")

	r, err := uc.resources.getResource(obj)
	if err != nil {
		return err
	}

	listOpts := ListOptions{}
	listOpts.ApplyOptions(opts)

	return r.Get().
		NamespaceIfScoped(listOpts.Namespace, r.isNamespaced()).
		Resource(r.resource()).
		VersionedParams(listOpts.AsListOptions(), uc.paramCodec).
		Do(ctx).
		Into(obj)
}

func (uc *unstructuredClient) GetSubResource(ctx context.Context, obj, subResourceObj Object, subResource string, opts ...SubResourceGetOption) error {
	if _, ok := obj.(runtime.Unstructured); !ok {
		return fmt.Errorf("unstructured client did not understand object: %T", obj)
	}

	if _, ok := subResourceObj.(runtime.Unstructured); !ok {
		return fmt.Errorf("unstructured client did not understand object: %T", subResourceObj)
	}

	if subResourceObj.GetName() == "" {
		subResourceObj.SetName(obj.GetName())
	}

	o, err := uc.resources.getObjMeta(obj)
	if err != nil {
		return err
	}

	getOpts := &SubResourceGetOptions{}
	getOpts.ApplyOptions(opts)

	return o.Get().
		NamespaceIfScoped(o.namespace, o.isNamespaced()).
		Resource(o.resource()).
		Name(o.name).
		SubResource(subResource).
		VersionedParams(getOpts.AsGetOptions(), uc.paramCodec).
		Do(ctx).
		Into(subResourceObj)
}

func (uc *unstructuredClient) CreateSubResource(ctx context.Context, obj, subResourceObj Object, subResource string, opts ...SubResourceCreateOption) error {
	if _, ok := obj.(runtime.Unstructured); !ok {
		return fmt.Errorf("unstructured client did not understand object: %T", obj)
	}

	if _, ok := subResourceObj.(runtime.Unstructured); !ok {
		return fmt.Errorf("unstructured client did not understand object: %T", subResourceObj)
	}

	if subResourceObj.GetName() == "" {
		subResourceObj.SetName(obj.GetName())
	}

	o, err := uc.resources.getObjMeta(obj)
	if err != nil {
		return err
	}

	createOpts := &SubResourceCreateOptions{}
	createOpts.ApplyOptions(opts)

	return o.Post().
		NamespaceIfScoped(o.namespace, o.isNamespaced()).
		Resource(o.resource()).
		Name(o.name).
		SubResource(subResource).
		Body(subResourceObj).
		VersionedParams(createOpts.AsCreateOptions(), uc.paramCodec).
		Do(ctx).
		Into(subResourceObj)
}

func (uc *unstructuredClient) UpdateSubResource(ctx context.Context, obj Object, subResource string, opts ...SubResourceUpdateOption) error {
	if _, ok := obj.(runtime.Unstructured); !ok {
		return fmt.Errorf("unstructured client did not understand object: %T", obj)
	}

	o, err := uc.resources.getObjMeta(obj)
	if err != nil {
		return err
	}

	updateOpts := SubResourceUpdateOptions{}
	updateOpts.ApplyOptions(opts)

	body := obj
	if updateOpts.SubResourceBody != nil {
		body = updateOpts.SubResourceBody
	}
	if body.GetName() == "" {
		body.SetName(obj.GetName())
	}
	if body.GetNamespace() == "" {
		body.SetNamespace(obj.GetNamespace())
	}

	return o.Put().
		NamespaceIfScoped(o.namespace, o.isNamespaced()).
		Resource(o.resource()).
		Name(o.name).
		SubResource(subResource).
		Body(body).
		VersionedParams(updateOpts.AsUpdateOptions(), uc.paramCodec).
		Do(ctx).
		Into(body)
}

func (uc *unstructuredClient) PatchSubResource(ctx context.Context, obj Object, subResource string, patch Patch, opts ...SubResourcePatchOption) error {
	u, ok := obj.(runtime.Unstructured)
	if !ok {
		return fmt.Errorf("unstructured client did not understand object: %T", obj)
	}

	gvk := u.GetObjectKind().GroupVersionKind()

	o, err := uc.resources.getObjMeta(obj)
	if err != nil {
		return err
	}

	patchOpts := &SubResourcePatchOptions{}
	patchOpts.ApplyOptions(opts)

	body := obj
	if patchOpts.SubResourceBody != nil {
		body = patchOpts.SubResourceBody
	}

	data, err := patch.Data(body)
	if err != nil {
		return err
	}

	result := o.Patch(patch.Type()).
		NamespaceIfScoped(o.namespace, o.isNamespaced()).
		Resource(o.resource()).
		Name(o.name).
		SubResource(subResource).
		Body(data).
		VersionedParams(patchOpts.AsPatchOptions(), uc.paramCodec).
		Do(ctx).
		Into(body)

	u.GetObjectKind().SetGroupVersionKind(gvk)
	return result
}
