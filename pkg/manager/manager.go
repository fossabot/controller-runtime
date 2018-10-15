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

package manager

import (
	"fmt"

	"github.com/tsungming/controller-runtime/pkg/cache"
	"github.com/tsungming/controller-runtime/pkg/client"
	"github.com/tsungming/controller-runtime/pkg/client/apiutil"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

// Manager initializes shared dependencies such as Caches and Clients, and provides them to runnables.
type Manager interface {
	// Add will set reqeusted dependencies on the component, and cause the component to be
	// started when Start is called
	Add(Runnable) error

	// SetFields will set dependencies on an object.
	SetFields(interface{}) error

	// Start starts all registered Controllers and blocks until the Stop channel is closed.
	// Returns an error if there is an error starting any controller.
	Start(<-chan struct{}) error

	// GetConfig returns an initialized Config
	GetConfig() *rest.Config

	// GetScheme returns and initialized Scheme
	GetScheme() *runtime.Scheme

	// GetClient returns a client configured with the Config
	GetClient() client.Client

	// GetFieldIndexer returns a client.FieldIndexer configured with the client
	GetFieldIndexer() client.FieldIndexer

	// GetCache returns a cache.Cache
	GetCache() cache.Cache
}

// Options are the arguments for creating a new Manager
type Options struct {
	// Scheme is the scheme used to resolve runtime.Objects to GroupVersionKinds / Resources
	// Defaults to the kubernetes/client-go scheme.Scheme
	Scheme *runtime.Scheme

	// Mapper is the rest mapper used to map go types to Kubernetes APIs
	MapperProvider func(c *rest.Config) (meta.RESTMapper, error)

	// Dependency injection for testing
	newCache  func(config *rest.Config, opts cache.Options) (cache.Cache, error)
	newClient func(config *rest.Config, options client.Options) (client.Client, error)
}

// Runnable allows a component to be started.
type Runnable interface {
	// Start starts running the component.  The component will stop running when the channel is closed.
	// Start blocks until the channel is closed or an error occurs.
	Start(<-chan struct{}) error
}

// RunnableFunc implements Runnable
type RunnableFunc func(<-chan struct{}) error

// Start implements Runnable
func (r RunnableFunc) Start(s <-chan struct{}) error {
	return r(s)
}

// New returns a new Manager
func New(config *rest.Config, options Options) (Manager, error) {
	cm := &controllerManager{config: config, scheme: options.Scheme, errChan: make(chan error)}

	// Initialize a rest.config if none was specified
	if cm.config == nil {
		return nil, fmt.Errorf("must specify Config")
	}

	// Use the Kubernetes client-go scheme if none is specified
	if cm.scheme == nil {
		cm.scheme = scheme.Scheme
	}

	// Create a new RESTMapper for mapping GroupVersionKinds to Resources
	if options.MapperProvider == nil {
		options.MapperProvider = apiutil.NewDiscoveryRESTMapper
	}
	mapper, err := options.MapperProvider(cm.config)
	if err != nil {
		log.Error(err, "Failed to get API Group-Resources")
		return nil, err
	}

	// Allow newClient to be mocked
	if options.newClient == nil {
		options.newClient = client.New
	}
	// Create the Client for Write operations.
	writeObj, err := options.newClient(cm.config, client.Options{Scheme: cm.scheme, Mapper: mapper})
	if err != nil {
		return nil, err
	}

	// TODO(directxman12): Figure out how to allow users to request a client without requesting a watch
	if options.newCache == nil {
		options.newCache = cache.New
	}
	cm.cache, err = options.newCache(cm.config, cache.Options{Scheme: cm.scheme, Mapper: mapper})
	if err != nil {
		return nil, err
	}

	cm.fieldIndexes = cm.cache
	cm.client = client.DelegatingClient{ReadInterface: cm.cache, WriteInterface: writeObj}
	return cm, nil
}
