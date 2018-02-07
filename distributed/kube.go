/*
Copyright © 2018 the InMAP authors.
This file is part of InMAP.

InMAP is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

InMAP is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with InMAP.  If not, see <http://www.gnu.org/licenses/>.
*/

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/container/v1"
	oauthsvc "google.golang.org/api/oauth2/v2"

	batch "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
)

// Run 'gcloud beta auth application-default login'
// and 'gcloud container clusters get-credentials cluster-1 --zone us-central1-a --project inmap-973'
// before running this.
// The first command authenticates you to deal with google kubernetes engine,
// and the second command sets up the kubernetes configuration to point and
// authenticate to our google cloud kubernetes cluster.

func main() {
	// This part shows how to connect to (and manage) the kubernetes engine
	// instance. We might not actually need to use this.
	src, err := google.DefaultTokenSource(oauth2.NoContext, oauthsvc.UserinfoEmailScope)
	if err != nil {
		log.Fatalf("Unable to acquire token source: %v", err)
	}
	client := oauth2.NewClient(context.Background(), src)
	containerService, err := container.New(client)
	if err != nil {
		log.Fatalf("Unable to create api service: %v", err)
	}
	fmt.Println("xxxxxxxxxxxxxxx", containerService)

	cluster, err := containerService.Projects.Zones.Clusters.Get("inmap-973", "us-central1-a", "cluster-1").Do()
	if err != nil {
		log.Fatalf("Unable to get cluster config: %v", err)
	}

	fmt.Printf("cluster %+v\n", cluster)

	log.Printf("UserInfo: %v", userInfo(client).Email)

	//////////////////////////////////////////////////////
	// This second part creates a batch job in our existing kubernetes cluster
	// and prints the status of all the existing jobs.

	config, err := clientcmd.BuildConfigFromFlags("", os.ExpandEnv("$HOME/.kube/config"))
	check(err)
	clientset, err := kubernetes.NewForConfig(config)
	check(err)

	batchClient := clientset.BatchV1()
	jobs := batchClient.Jobs("default")
	fmt.Println("xxxxxxxxxxx", batchClient)
	_, err = jobs.Create(batchJob)
	check(err)

	jobList, err := jobs.List(meta.ListOptions{})
	check(err)
	for _, job := range jobList.Items {
		fmt.Printf("job status: %+v\n", job.Status)
	}
}

func userInfo(c *http.Client) *oauthsvc.Userinfoplus {
	service, err := oauthsvc.New(c)
	if err != nil {
		log.Fatalf("Unable to create api service: %v", err)
	}
	ui, err := service.Userinfo.Get().Do()
	if err != nil {
		log.Fatalf("Unable to get userinfo: ", err)
	}
	return ui
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

var falseVal bool

var batchJob = &batch.Job{
	TypeMeta: meta.TypeMeta{
		Kind:       "Job",
		APIVersion: "v1",
	},
	ObjectMeta: meta.ObjectMeta{
		Name:   "k8sexp-testjob-" + time.Now().Format("15-04-05"),
		Labels: make(map[string]string),
	},
	Spec: batch.JobSpec{
		Template: core.PodTemplateSpec{
			ObjectMeta: meta.ObjectMeta{
				Name: "k8sexp-testpod",
			},
			Spec: core.PodSpec{
				InitContainers: []core.Container{}, // Doesn't seem obligatory(?)...
				Containers: []core.Container{
					{
						Name:    "k8sexp-testimg",
						Image:   "perl",
						Command: []string{"sleep", "10"},
						SecurityContext: &core.SecurityContext{
							Privileged: &falseVal,
						},
						ImagePullPolicy: core.PullPolicy(core.PullIfNotPresent),
					},
				},
				RestartPolicy: core.RestartPolicyOnFailure,
			},
		},
	},
}
