package pkg

import (
	"fmt"
	"regexp"

	"github.com/go-logr/logr"

	nodelabelsv1 "github.com/cloud-for-you/node-label-operator/api/v1"
	v1 "k8s.io/api/core/v1"
)

func AddLabels(node *v1.Node, labels nodelabelsv1.Labels, log logr.Logger) bool {
	if !labels.GetDeletionTimestamp().IsZero() {
		return false
	}

	log.Info(
		"Checking if labels need to be added to node",
		"node",
		node.Name,
		"label config",
		fmt.Sprintf("%+v", labels.Spec),
	)
	nodeModified := false

	for _, nodeNamePattern := range labels.Spec.NodeNamePatterns {

		pattern := fmt.Sprintf("%s%s%s", "^", nodeNamePattern, "$")
		match, err := regexp.MatchString(pattern, node.Name)

		if err != nil {
			log.Error(err, "Invalid regular expression, moving on to next rule")
			continue
		}
		if !match {
			continue
		}
		if node.Labels == nil {
			node.Labels = map[string]string{}
		}
		for name, value := range labels.Spec.Labels {
			if val, ok := node.Labels[name]; !ok || val != value {
				log.Info(
					"Adding label to node based on pattern",
					"node",
					node.Name,
					"pattern",
					nodeNamePattern,
					"labelName",
					name,
					"labelValue",
					value,
				)
				node.Labels[name] = value
				nodeModified = true
			}
		}
	}

	return nodeModified
}
