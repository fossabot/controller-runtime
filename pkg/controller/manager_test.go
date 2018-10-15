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

package controller

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tsungming/controller-runtime/pkg/client"
	"github.com/tsungming/controller-runtime/pkg/controller/reconcile"
	"github.com/tsungming/controller-runtime/pkg/internal/informer"
	"github.com/tsungming/controller-runtime/pkg/internal/informer/informertest"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

var TestConfig *rest.Config

var _ = Describe("controller", func() {
	var stop chan struct{}

	rec := reconcile.Func(func(reconcile.Request) (reconcile.Result, error) {
		return reconcile.Result{}, nil
	})
	BeforeEach(func() {
		stop = make(chan struct{})
	})

	AfterEach(func() {
		close(stop)
	})

	Describe("Creating a Manager", func() {

		It("should return an error if there is no Config", func() {
			m, err := NewManager(ManagerArgs{})
			Expect(m).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("must specify Config"))

		})

		It("should return an error if it can't create a RestMapper", func() {
			expected := fmt.Errorf("expected error: RestMapper")
			m, err := NewManager(ManagerArgs{
				Config:         TestConfig,
				MapperProvider: func(c *rest.Config) (meta.RESTMapper, error) { return nil, expected },
			})
			Expect(m).To(BeNil())
			Expect(err).To(Equal(expected))

		})
	})

	Describe("Staring a Manager", func() {

		It("should Start each Controller", func() {
			// TODO(community): write this
		})

		It("should return an error if any Controllers fail to stop", func(done Done) {
			m, err := NewManager(ManagerArgs{Config: TestConfig})
			Expect(err).NotTo(HaveOccurred())
			c, err := m.NewController(Args{Name: "foo"}, rec)
			Expect(err).NotTo(HaveOccurred())
			ctrl, ok := c.(*controller)
			Expect(ok).To(BeTrue())

			// Make Controller startup fail
			ctrl.waitForCache = func(stopCh <-chan struct{}, cacheSyncs ...cache.InformerSynced) bool { return false }
			err = m.Start(stop)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("caches to sync"))

			close(done)
		})
	})

	Describe("Manager", func() {
		It("should provide a function to get the Config", func() {
			m, err := NewManager(ManagerArgs{Config: TestConfig})
			Expect(err).NotTo(HaveOccurred())
			mrg, ok := m.(*controllerManager)
			Expect(ok).To(BeTrue())
			Expect(m.GetConfig()).To(Equal(mrg.config))
		})

		It("should provide a function to get the Client", func() {
			m, err := NewManager(ManagerArgs{Config: TestConfig})
			Expect(err).NotTo(HaveOccurred())
			mrg, ok := m.(*controllerManager)
			Expect(ok).To(BeTrue())
			Expect(m.GetClient()).To(Equal(mrg.client))
		})

		It("should provide a function to get the Scheme", func() {
			m, err := NewManager(ManagerArgs{Config: TestConfig})
			Expect(err).NotTo(HaveOccurred())
			mrg, ok := m.(*controllerManager)
			Expect(ok).To(BeTrue())
			Expect(m.GetScheme()).To(Equal(mrg.scheme))
		})

		It("should provide a function to get the FieldIndexer", func() {
			m, err := NewManager(ManagerArgs{Config: TestConfig})
			Expect(err).NotTo(HaveOccurred())
			mrg, ok := m.(*controllerManager)
			Expect(ok).To(BeTrue())
			Expect(m.GetFieldIndexer()).To(Equal(mrg.fieldIndexes))
		})
	})

	Describe("Creating a Controller", func() {
		It("should return an error if Name is not Specified", func() {
			m, err := NewManager(ManagerArgs{Config: TestConfig})
			Expect(err).NotTo(HaveOccurred())
			c, err := m.NewController(Args{}, rec)
			Expect(c).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("must specify Name for Controller"))
		})

		It("should return an error if Reconcile is not Specified", func() {
			m, err := NewManager(ManagerArgs{Config: TestConfig})
			Expect(err).NotTo(HaveOccurred())
			c, err := m.NewController(Args{Name: "foo"}, nil)
			Expect(c).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("must specify Reconcile"))

		})

		It("should immediately start the Controller if the ControllerManager has already Started", func() {
			m, err := NewManager(ManagerArgs{Config: TestConfig})
			Expect(err).NotTo(HaveOccurred())
			mrg, ok := m.(*controllerManager)
			Expect(ok).To(BeTrue())

			// Make Controller startup fail
			go func() {
				defer GinkgoRecover()
				Expect(m.Start(stop)).NotTo(HaveOccurred())
			}()
			Eventually(func() bool { return mrg.started }).Should(BeTrue())

			c, err := m.NewController(Args{Name: "Foo"}, rec)
			Expect(err).NotTo(HaveOccurred())
			ctrl, ok := c.(*controller)
			Expect(ok).To(BeTrue())

			// Wait for Controller to start
			Eventually(func() bool { return ctrl.started }).Should(BeTrue())
		})

		It("should provide an inject function for providing dependencies", func(done Done) {
			m, err := NewManager(ManagerArgs{Config: TestConfig})
			Expect(err).NotTo(HaveOccurred())
			mrg, ok := m.(*controllerManager)
			Expect(ok).To(BeTrue())

			mrg.informers = &informertest.FakeInformers{}

			c, err := m.NewController(Args{Name: "foo"}, rec)
			Expect(err).NotTo(HaveOccurred())
			ctrl, ok := c.(*controller)
			Expect(ok).To(BeTrue())

			By("Injecting the dependencies")
			err = ctrl.inject(&injectable{
				scheme: func(scheme *runtime.Scheme) error {
					defer GinkgoRecover()
					Expect(scheme).To(Equal(mrg.scheme))
					return nil
				},
				config: func(config *rest.Config) error {
					defer GinkgoRecover()
					Expect(config).To(Equal(mrg.config))
					return nil
				},
				client: func(client client.Interface) error {
					defer GinkgoRecover()
					Expect(client).To(Equal(mrg.client))
					return nil
				},
				informers: func(informers informer.Informers) error {
					defer GinkgoRecover()
					Expect(informers).To(Equal(mrg.informers))
					return nil
				},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Returning an error if dependency injection fails")

			expected := fmt.Errorf("expected error")
			err = ctrl.inject(&injectable{
				client: func(client client.Interface) error {
					return expected
				},
			})
			Expect(err).To(Equal(expected))

			err = ctrl.inject(&injectable{
				scheme: func(scheme *runtime.Scheme) error {
					return expected
				},
			})
			Expect(err).To(Equal(expected))

			err = ctrl.inject(&injectable{
				config: func(config *rest.Config) error {
					return expected
				},
			})
			Expect(err).To(Equal(expected))

			err = ctrl.inject(&injectable{
				informers: func(informers informer.Informers) error {
					return expected
				},
			})
			Expect(err).To(Equal(expected))

			close(done)
		})
	})
})

type injectable struct {
	scheme    func(scheme *runtime.Scheme) error
	client    func(client.Interface) error
	config    func(config *rest.Config) error
	informers func(informer.Informers) error
}

func (i *injectable) InjectInformers(informers informer.Informers) error {
	if i.informers == nil {
		return nil
	}
	return i.informers(informers)
}

func (i *injectable) InjectConfig(config *rest.Config) error {
	if i.config == nil {
		return nil
	}
	return i.config(config)
}

func (i *injectable) InjectClient(c client.Interface) error {
	if i.client == nil {
		return nil
	}
	return i.client(c)
}

func (i *injectable) InjectScheme(scheme *runtime.Scheme) error {
	if i.scheme == nil {
		return nil
	}
	return i.scheme(scheme)
}
