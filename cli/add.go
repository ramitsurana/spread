package cli

import (
	"strings"

	"github.com/codegangsta/cli"

	"rsprd.com/spread/pkg/data"
	"rsprd.com/spread/pkg/deploy"
)

// Init sets up a Spread repository for versioning.
func (s SpreadCli) Add() *cli.Command {
	return &cli.Command{
		Name:        "add",
		Usage:       "spread add <path>",
		Description: "Stage objects to the index",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "namespace",
				Value: "default",
				Usage: "namespace to look for objects",
			},
			cli.StringFlag{
				Name:  "context",
				Value: "",
				Usage: "kubectl context to use for requests",
			},
			cli.BoolFlag{
				Name:  "no-export",
				Usage: "don't request Kube API server to export objects",
			},
		},
		Action: func(c *cli.Context) {
			// Download specified object from Kubernetes cluster
			// example: spread add rc/mattermost
			resource := c.Args().First()
			if len(resource) == 0 {
				s.fatalf("A resource to be added must be specified")
			}

			context := c.String("context")
			cluster, err := deploy.NewKubeClusterFromContext(context)
			if err != nil {
				s.fatalf("Failed to connect to Kubernetes cluster: %v", err)
			}

			// parse resource type and name
			parts := strings.Split(resource, "/")
			if len(parts) != 2 {
				s.fatalf("Unrecognized resource format")
			}

			kind, name := parts[0], parts[1]
			kind = kubeShortForm(kind)
			namespace := c.String("namespace")
			export := !c.Bool("no-export")
			req := cluster.Client.Get().Resource(kind).Namespace(namespace).Name(name)

			if export {
				req.Param("export", "true")
			}

			runObj, err := req.Do().Get()
			if err != nil {
				s.fatalf("Failed to retrieve resource '%s (namespace=%s)' from Kube API server: %v", resource, namespace, err)
			}

			kubeObj, err := deploy.AsKubeObject(runObj)
			if err != nil {
				s.fatalf("Unable to change into KubeObject: %v", err)
			}
			// TODO(DG): Clean this up
			gvk := kubeObj.GetObjectKind().GroupVersionKind()
			gvk.Version = "v1"
			kubeObj.GetObjectKind().SetGroupVersionKind(gvk)
			kubeObj.GetObjectMeta().SetNamespace(namespace)

			path, err := deploy.ObjectPath(kubeObj)
			if err != nil {
				s.fatalf("Failed to determine path to save object: %v", err)
			}

			obj, err := data.CreateObject(kubeObj.GetObjectMeta().GetName(), path, kubeObj)
			if err != nil {
				s.fatalf("failed to encode object: %v", err)
			}

			proj, err := s.project()
			if err != nil {
				s.fatalf("Not in a Spread project.")
			}

			err = proj.AddObjectToIndex(obj)
			if err != nil {
				s.fatalf("Failed to add object to Git index: %v", err)
			}
		},
	}
}