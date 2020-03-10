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

package fake

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
	v1alpha1 "knative.dev/eventing-contrib/azsb/pkg/apis/messaging/v1alpha1"
)

// FakeAZServicebusChannels implements AZServicebusChannelInterface
type FakeAZServicebusChannels struct {
	Fake *FakeMessagingV1alpha1
	ns   string
}

var azservicebuschannelsResource = schema.GroupVersionResource{Group: "messaging.knative.dev", Version: "v1alpha1", Resource: "azservicebuschannels"}

var azservicebuschannelsKind = schema.GroupVersionKind{Group: "messaging.knative.dev", Version: "v1alpha1", Kind: "AZServicebusChannel"}

// Get takes name of the aZServicebusChannel, and returns the corresponding aZServicebusChannel object, and an error if there is any.
func (c *FakeAZServicebusChannels) Get(name string, options v1.GetOptions) (result *v1alpha1.AZServicebusChannel, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(azservicebuschannelsResource, c.ns, name), &v1alpha1.AZServicebusChannel{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.AZServicebusChannel), err
}

// List takes label and field selectors, and returns the list of AZServicebusChannels that match those selectors.
func (c *FakeAZServicebusChannels) List(opts v1.ListOptions) (result *v1alpha1.AZServicebusChannelList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(azservicebuschannelsResource, azservicebuschannelsKind, c.ns, opts), &v1alpha1.AZServicebusChannelList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.AZServicebusChannelList{ListMeta: obj.(*v1alpha1.AZServicebusChannelList).ListMeta}
	for _, item := range obj.(*v1alpha1.AZServicebusChannelList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested aZServicebusChannels.
func (c *FakeAZServicebusChannels) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(azservicebuschannelsResource, c.ns, opts))

}

// Create takes the representation of a aZServicebusChannel and creates it.  Returns the server's representation of the aZServicebusChannel, and an error, if there is any.
func (c *FakeAZServicebusChannels) Create(aZServicebusChannel *v1alpha1.AZServicebusChannel) (result *v1alpha1.AZServicebusChannel, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(azservicebuschannelsResource, c.ns, aZServicebusChannel), &v1alpha1.AZServicebusChannel{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.AZServicebusChannel), err
}

// Update takes the representation of a aZServicebusChannel and updates it. Returns the server's representation of the aZServicebusChannel, and an error, if there is any.
func (c *FakeAZServicebusChannels) Update(aZServicebusChannel *v1alpha1.AZServicebusChannel) (result *v1alpha1.AZServicebusChannel, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(azservicebuschannelsResource, c.ns, aZServicebusChannel), &v1alpha1.AZServicebusChannel{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.AZServicebusChannel), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeAZServicebusChannels) UpdateStatus(aZServicebusChannel *v1alpha1.AZServicebusChannel) (*v1alpha1.AZServicebusChannel, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(azservicebuschannelsResource, "status", c.ns, aZServicebusChannel), &v1alpha1.AZServicebusChannel{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.AZServicebusChannel), err
}

// Delete takes name of the aZServicebusChannel and deletes it. Returns an error if one occurs.
func (c *FakeAZServicebusChannels) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(azservicebuschannelsResource, c.ns, name), &v1alpha1.AZServicebusChannel{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeAZServicebusChannels) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(azservicebuschannelsResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.AZServicebusChannelList{})
	return err
}

// Patch applies the patch and returns the patched aZServicebusChannel.
func (c *FakeAZServicebusChannels) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.AZServicebusChannel, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(azservicebuschannelsResource, c.ns, name, pt, data, subresources...), &v1alpha1.AZServicebusChannel{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.AZServicebusChannel), err
}
