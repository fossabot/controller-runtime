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

package source_test

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tsungming/controller-runtime/pkg/internal/informer"
	logf "github.com/tsungming/controller-runtime/pkg/runtime/log"
	"github.com/tsungming/controller-runtime/pkg/test"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func TestSource(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecsWithDefaultAndCustomReporters(t, "Source Suite", []Reporter{test.NewlineReporter{}})
}

var testenv *test.Environment
var config *rest.Config
var clientset *kubernetes.Clientset
var icache informer.Informers
var stop = make(chan struct{})

var _ = BeforeSuite(func() {
	logf.SetLogger(logf.ZapLogger(true))

	testenv = &test.Environment{}

	var err error
	config, err = testenv.Start()
	Expect(err).NotTo(HaveOccurred())

	time.Sleep(1 * time.Second)

	clientset, err = kubernetes.NewForConfig(config)
	Expect(err).NotTo(HaveOccurred())

	icache = &informer.SelfPopulatingInformers{Config: config}
})

var _ = AfterSuite(func() {
	close(stop)
	testenv.Stop()
})
