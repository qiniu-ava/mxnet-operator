/*
Copyright 2017 The Kubernetes Authors.

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

package v1

import (
	"time"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	arbv1 "github.com/kubernetes-incubator/kube-arbitrator/pkg/apis/v1alpha1"
	"github.com/kubernetes-incubator/kube-arbitrator/pkg/client/informers/internalinterfaces"
	v1 "github.com/kubernetes-incubator/kube-arbitrator/pkg/client/listers/v1alpha1"
)

// QueueJobInformer provides access to a shared informer and lister for
// QueueJobs.
type QueueJobInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1.QueueJobLister
}

type queueJobInformer struct {
	factory internalinterfaces.SharedInformerFactory
}

// NewQueueJobInformer constructs a new informer for QueueJob type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewQueueJobInformer(client *rest.RESTClient, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	source := cache.NewListWatchFromClient(
		client,
		arbv1.QueueJobPlural,
		namespace,
		fields.Everything())

	return cache.NewSharedIndexInformer(
		source,
		&arbv1.QueueJob{},
		resyncPeriod,
		indexers,
	)
}

func defaultQueueJobInformer(client *rest.RESTClient, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewQueueJobInformer(client, meta_v1.NamespaceAll, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
}

func (f *queueJobInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&arbv1.QueueJob{}, defaultQueueJobInformer)
}

func (f *queueJobInformer) Lister() v1.QueueJobLister {
	return v1.NewQueueJobLister(f.Informer().GetIndexer())
}
