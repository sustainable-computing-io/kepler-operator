package controllers

import (
	"context"

	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type KeplerKey struct {
	Name       string
	Namespace  string
	ObjectType string
}

type KeplerValues struct {
	ctx   context.Context
	obj   client.Object
	patch client.Patch
	list  client.ObjectList
}

type KeplerStatusClient struct {
	NameSpacedNameToObject map[KeplerKey]KeplerValues
}

func (k *KeplerStatusClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	k.NameSpacedNameToObject[KeplerKey{
		Name:       obj.GetName(),
		Namespace:  obj.GetNamespace(),
		ObjectType: obj.GetObjectKind().GroupVersionKind().Kind}] = KeplerValues{
		ctx:   ctx,
		obj:   obj,
		patch: nil,
		list:  nil,
	}
	return nil
}

// TODO: Implement when needed for unit testing
func (k *KeplerStatusClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	return nil
}

var _ client.Client = &KeplerClient{}

type KeplerClient struct {
	NameSpacedNameToObject map[KeplerKey]KeplerValues
	KeplerStatus           *KeplerStatusClient
}

func NewClient() *KeplerClient {
	return &KeplerClient{
		KeplerStatus: &KeplerStatusClient{},
	}
}

func (k *KeplerClient) Status() client.StatusWriter {
	return k.KeplerStatus
}

func (k *KeplerClient) Get(ctx context.Context, key types.NamespacedName, obj client.Object, opts ...client.GetOption) error {
	if k.NameSpacedNameToObject == nil {
		k.NameSpacedNameToObject = make(map[KeplerKey]KeplerValues)
	}
	kc := KeplerKey{
		Name:       key.Name,
		Namespace:  key.Namespace,
		ObjectType: obj.GetObjectKind().GroupVersionKind().Kind,
	}
	_, ok := k.NameSpacedNameToObject[kc]
	if !ok {
		return errors.NewNotFound(schema.GroupResource{}, "Not Found")
	} else {
		return nil
	}
}

// TODO: Implement when needed for unit testing
func (k *KeplerClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	return nil
}

// TODO: Implement when needed for unit testing
func (k *KeplerClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	return nil
}

func (k *KeplerClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	if k.NameSpacedNameToObject == nil {
		k.NameSpacedNameToObject = make(map[KeplerKey]KeplerValues)
	}
	k.NameSpacedNameToObject[KeplerKey{
		Name:       obj.GetName(),
		Namespace:  obj.GetNamespace(),
		ObjectType: obj.GetObjectKind().GroupVersionKind().Kind}] = KeplerValues{
		ctx:   ctx,
		obj:   obj,
		patch: nil,
		list:  nil,
	}
	result := k.NameSpacedNameToObject[KeplerKey{Name: obj.GetName(),
		Namespace:  obj.GetNamespace(),
		ObjectType: obj.GetObjectKind().GroupVersionKind().Kind}]
	returned := result.obj.(*appsv1.DaemonSet)
	fmt.Print("goodmorning\n")
	fmt.Print(returned.Spec.Template.Spec.Containers[0].VolumeMounts)
	return nil
}

// TODO: Implement when needed for unit testing
func (k *KeplerClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	return nil
}

// TODO: Implement when needed for unit testing
func (k *KeplerClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	return nil
}

func (k *KeplerClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	if k.NameSpacedNameToObject == nil {
		k.NameSpacedNameToObject = make(map[KeplerKey]KeplerValues)
	}
	k.NameSpacedNameToObject[KeplerKey{
		Name:       obj.GetName(),
		Namespace:  obj.GetNamespace(),
		ObjectType: obj.GetObjectKind().GroupVersionKind().Kind}] = KeplerValues{
		ctx:   ctx,
		obj:   obj,
		patch: nil,
		list:  nil,
	}
	return nil
}

// TODO: Implement when needed for unit testing
func (k *KeplerClient) RESTMapper() meta.RESTMapper {
	return nil
}

// TODO: Implement when needed for unit testing
func (k *KeplerClient) Scheme() *runtime.Scheme {
	return nil
}
