/*
Copyright 2020 The Knative Authors

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
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
	v1alpha1 "knative.dev/eventing-contrib/azsb/channel/pkg/apis/messaging/v1alpha1"
)

// AZServicebusChannelLister helps list AZServicebusChannels.
type AZServicebusChannelLister interface {
	// List lists all AZServicebusChannels in the indexer.
	List(selector labels.Selector) (ret []*v1alpha1.AZServicebusChannel, err error)
	// AZServicebusChannels returns an object that can list and get AZServicebusChannels.
	AZServicebusChannels(namespace string) AZServicebusChannelNamespaceLister
	AZServicebusChannelListerExpansion
}

// aZServicebusChannelLister implements the AZServicebusChannelLister interface.
type aZServicebusChannelLister struct {
	indexer cache.Indexer
}

// NewAZServicebusChannelLister returns a new AZServicebusChannelLister.
func NewAZServicebusChannelLister(indexer cache.Indexer) AZServicebusChannelLister {
	return &aZServicebusChannelLister{indexer: indexer}
}

// List lists all AZServicebusChannels in the indexer.
func (s *aZServicebusChannelLister) List(selector labels.Selector) (ret []*v1alpha1.AZServicebusChannel, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.AZServicebusChannel))
	})
	return ret, err
}

// AZServicebusChannels returns an object that can list and get AZServicebusChannels.
func (s *aZServicebusChannelLister) AZServicebusChannels(namespace string) AZServicebusChannelNamespaceLister {
	return aZServicebusChannelNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// AZServicebusChannelNamespaceLister helps list and get AZServicebusChannels.
type AZServicebusChannelNamespaceLister interface {
	// List lists all AZServicebusChannels in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1alpha1.AZServicebusChannel, err error)
	// Get retrieves the AZServicebusChannel from the indexer for a given namespace and name.
	Get(name string) (*v1alpha1.AZServicebusChannel, error)
	AZServicebusChannelNamespaceListerExpansion
}

// aZServicebusChannelNamespaceLister implements the AZServicebusChannelNamespaceLister
// interface.
type aZServicebusChannelNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all AZServicebusChannels in the indexer for a given namespace.
func (s aZServicebusChannelNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.AZServicebusChannel, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.AZServicebusChannel))
	})
	return ret, err
}

// Get retrieves the AZServicebusChannel from the indexer for a given namespace and name.
func (s aZServicebusChannelNamespaceLister) Get(name string) (*v1alpha1.AZServicebusChannel, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("azservicebuschannel"), name)
	}
	return obj.(*v1alpha1.AZServicebusChannel), nil
}
