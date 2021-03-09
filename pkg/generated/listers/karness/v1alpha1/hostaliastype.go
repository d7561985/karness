/*
Author d7561985@gmail.com

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

// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/d7561985/karness/pkg/apis/karness/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// HostAliasTypeLister helps list HostAliasTypes.
// All objects returned here must be treated as read-only.
type HostAliasTypeLister interface {
	// List lists all HostAliasTypes in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.HostAliasType, err error)
	// HostAliasTypes returns an object that can list and get HostAliasTypes.
	HostAliasTypes(namespace string) HostAliasTypeNamespaceLister
	HostAliasTypeListerExpansion
}

// hostAliasTypeLister implements the HostAliasTypeLister interface.
type hostAliasTypeLister struct {
	indexer cache.Indexer
}

// NewHostAliasTypeLister returns a new HostAliasTypeLister.
func NewHostAliasTypeLister(indexer cache.Indexer) HostAliasTypeLister {
	return &hostAliasTypeLister{indexer: indexer}
}

// List lists all HostAliasTypes in the indexer.
func (s *hostAliasTypeLister) List(selector labels.Selector) (ret []*v1alpha1.HostAliasType, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.HostAliasType))
	})
	return ret, err
}

// HostAliasTypes returns an object that can list and get HostAliasTypes.
func (s *hostAliasTypeLister) HostAliasTypes(namespace string) HostAliasTypeNamespaceLister {
	return hostAliasTypeNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// HostAliasTypeNamespaceLister helps list and get HostAliasTypes.
// All objects returned here must be treated as read-only.
type HostAliasTypeNamespaceLister interface {
	// List lists all HostAliasTypes in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.HostAliasType, err error)
	// Get retrieves the HostAliasType from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.HostAliasType, error)
	HostAliasTypeNamespaceListerExpansion
}

// hostAliasTypeNamespaceLister implements the HostAliasTypeNamespaceLister
// interface.
type hostAliasTypeNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all HostAliasTypes in the indexer for a given namespace.
func (s hostAliasTypeNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.HostAliasType, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.HostAliasType))
	})
	return ret, err
}

// Get retrieves the HostAliasType from the indexer for a given namespace and name.
func (s hostAliasTypeNamespaceLister) Get(name string) (*v1alpha1.HostAliasType, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("hostaliastype"), name)
	}
	return obj.(*v1alpha1.HostAliasType), nil
}
