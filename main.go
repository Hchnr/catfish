/*
Copyright 2016 The Kubernetes Authors.

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

// Note: the example only works with the code within the same release/branch.
package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	"github.com/hchnr/catfish/util"
	"github.com/hchnr/catfish/common"
)

func init() {

}

// find pod candidate to update randomly
func findPod(clientset *kubernetes.Clientset) (*corev1.Pod, error ) {
    var pods []corev1.Pod
	for _, ns := range common.Config.Cluster.Namespaces {
	    nspods, err := clientset.CoreV1().Pods(ns).List(metav1.ListOptions{})
		if err != nil {
		    return nil, err
		}
		pods = append(pods, nspods.Items...)
	} 
	util.Logger.Info("Total Pods: ", len(pods))
	pod := pods[rand.Intn(len(pods))]
	util.Logger.Info("Selected pods: ", pod.ObjectMeta.Name)
	return &pod, nil
}

// check whether the pod candidate can be updated
func checkPod(clientset *kubernetes.Clientset, pod *corev1.Pod) (bool) {
    util.Logger.Info("Check Pod: ", pod.ObjectMeta.Name)

    // check for pods name protection
	for _, keyword := range common.Config.Cluster.Protects {
	    if !strings.Contains(pod.ObjectMeta.Name, keyword) {
		    util.Logger.Info("Check not passed for pod name protect.")
		    return false
		}
	}

	// check pod's create time
	util.Logger.Info("Create at: ", pod.ObjectMeta.CreationTimestamp.Time, " Delta time: ", metav1.Now().Time.Sub(pod.ObjectMeta.CreationTimestamp.Time))
	oneDay, err := time.ParseDuration(common.Config.Cluster.Duration)
	if err != nil {
	    panic(err.Error())
   	}
	if metav1.Now().Time.Sub(pod.ObjectMeta.CreationTimestamp.Time) < oneDay {
	    util.Logger.Info("Check not passed for create time.")
		return false
	}

	// check for deployment
	labels := pod.ObjectMeta.Labels
	deploymentsLists, err := clientset.AppsV1().Deployments("").List(metav1.ListOptions{})
	if err != nil {
	    panic(err.Error())
   	}

	var deploymentfound appsv1.Deployment 
	isok := true
	for _, deployment := range deploymentsLists.Items {
	    // TODO: resolve labels only, later on resolve expressions also 
	    selector := deployment.Spec.Selector.MatchLabels
		for key := range selector {
		    if labels[key] != selector[key] {
			    isok = false
			    break
			}
		}
	    if isok {
		    deploymentfound = deployment
			break
		}
	}
	if isok {
        util.Logger.Info("Deployment found: ", deploymentfound.ObjectMeta.Name )	
    } else {
        util.Logger.Info("Deployment not found: ")	
	}

    if common.Config.Cluster.IsPrtDep {
	
	}

	return true
}

func updatePod(clientset *kubernetes.Clientset, pod *corev1.Pod) (error) {
	util.Logger.Info("Recreate at: ",  metav1.Now().Time)
    err := clientset.CoreV1().Pods(pod.ObjectMeta.Namespace).Delete(pod.ObjectMeta.Name, new(metav1.DeleteOptions));
    return err
}


func main() {
    util.Logger.Info("[Catfish boot now]:")

	var kubeconfig *string
	if home := homeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	for {
	    for {
		    podfound, err := findPod(clientset)
		    if err != nil {
			    panic(err.Error())
	    	}

		    if checkPod(clientset, podfound) {
			    err := updatePod(clientset, podfound)
		        if err != nil {
		    	    panic(err.Error())
	        	}
			    break
			}
	    }


		/*
		for index, pod := range pods.Items {
			fmt.Printf("Pod #%d: %s\n", index, pod.ObjectMeta.Name)
			fmt.Printf("Create at: %s \n", pod.ObjectMeta.CreationTimestamp.Time)
			fmt.Printf("Delta time: %s\n", metav1.Now().Time.Sub(pod.ObjectMeta.CreationTimestamp.Time))
			oneDay, err := time.ParseDuration("24h")
		    if err != nil {
			    panic(err.Error())
	    	}
			if metav1.Now().Time.Sub(pod.ObjectMeta.CreationTimestamp.Time) > oneDay {
			    fmt.Printf("True\n")
			}

	    }
        */

		// Examples for error handling:
		// - Use helper functions like e.g. errors.IsNotFound()
		// - And/or cast to StatusError and use its properties like e.g. ErrStatus.Message
		namespace := "default"
		pod := "example-xxxxx"
		_, err = clientset.CoreV1().Pods(namespace).Get(pod, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			fmt.Printf("Pod %s in namespace %s not found\n", pod, namespace)
		} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
			fmt.Printf("Error getting pod %s in namespace %s: %v\n",
				pod, namespace, statusError.ErrStatus.Message)
		} else if err != nil {
			panic(err.Error())
		} else {
			fmt.Printf("Found pod %s in namespace %s\n", pod, namespace)
		}

		time.Sleep(10 * time.Second)
	}
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
