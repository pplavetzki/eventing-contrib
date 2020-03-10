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

// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
	v1alpha1 "knative.dev/eventing-contrib/azsb/pkg/apis/messaging/v1alpha1"
	scheme "knative.dev/eventing-contrib/azsb/pkg/client/clientset/versioned/scheme"
)

// AZServicebusChannelsGetter has a method to return a AZServicebusChannelInterface.
// A group's client should implement this interface.
type AZServicebusChannelsGetter interface {
	AZServicebusChannels(namespace string) AZServicebusChannelInterface
}

// AZServicebusChannelInterface has methods to work with AZServicebusChannel resources.
type AZServicebusChannelInterface interface {
	Create(*v1alpha1.AZServicebusChannel) (*v1alpha1.AZServicebusChannel, error)
	Update(*v1alpha1.AZServicebusChannel) (*v1alpha1.AZServicebusChannel, error)
	UpdateStatus(*v1alpha1.AZServicebusChannel) (*v1alpha1.AZServicebusChannel, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.AZServicebusChannel, error)
	List(opts v1.ListOptions) (*v1alpha1.AZServicebusChannelList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.AZServicebusChannel, err error)
	AZServicebusChannelExpansion
}

// aZServicebusChannels implements AZServicebusChannelInterface
type aZServicebusChannels struct {
	client rest.Interface
	ns     string
}

// newAZServicebusChannels returns a AZServicebusChannels
func newAZServicebusChannels(c *MessagingV1alpha1Client, namespace string) *aZServicebusChannels {
	return &aZServicebusChannels{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the aZServicebusChannel, and returns the corresponding aZServicebusChannel object, and an error if there is any.
func (c *aZServicebusChannels) Get(name string, options v1.GetOptions) (result *v1alpha1.AZServicebusChannel, err error) {
	result = &v1alpha1.AZServicebusChannel{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("azservicebuschannels").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of AZServicebusChannels that match those selectors.
func (c *aZServicebusChannels) List(opts v1.ListOptions) (result *v1alpha1.AZServicebusChannelList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.AZServicebusChannelList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("azservicebuschannels").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested aZServicebusChannels.
func (c *aZServicebusChannels) Watch(opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("azservicebuschannels").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch()
}

// Create takes the representation of a aZServicebusChannel and creates it.  Returns the server's representation of the aZServicebusChannel, and an error, if there is any.
func (c *aZServicebusChannels) Create(aZServicebusChannel *v1alpha1.AZServicebusChannel) (result *v1alpha1.AZServicebusChannel, err error) {
	result = &v1alpha1.AZServicebusChannel{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("azservicebuschannels").
		Body(aZServicebusChannel).
		Do().
		Into(result)
	return
}

// Update takes the representation of a aZServicebusChannel and updates it. Returns the server's representation of the aZServicebusChannel, and an error, if there is any.
func (c *aZServicebusChannels) Update(aZServicebusChannel *v1alpha1.AZServicebusChannel) (result *v1alpha1.AZServicebusChannel, err error) {
	result = &v1alpha1.AZServicebusChannel{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("azservicebuschannels").
		Name(aZServicebusChannel.Name).
		Body(aZServicebusChannel).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *aZServicebusChannels) UpdateStatus(aZServicebusChannel *v1alpha1.AZServicebusChannel) (result *v1alpha1.AZServicebusChannel, err error) {
	result = &v1alpha1.AZServicebusChannel{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("azservicebuschannels").
		Name(aZServicebusChannel.Name).
		SubResource("status").
		Body(aZServicebusChannel).
		Do().
		Into(result)
	return
}

// Delete takes name of the aZServicebusChannel and deletes it. Returns an error if one occurs.
func (c *aZServicebusChannels) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("azservicebuschannels").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *aZServicebusChannels) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	var timeout time.Duration
	if listOptions.TimeoutSeconds != nil {
		timeout = time.Duration(*listOptions.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("azservicebuschannels").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Timeout(timeout).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched aZServicebusChannel.
func (c *aZServicebusChannels) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.AZServicebusChannel, err error) {
	result = &v1alpha1.AZServicebusChannel{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("azservicebuschannels").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
