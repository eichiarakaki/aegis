package component

import "github.com/eichiarakaki/aegis/internals/core"

// Describe returns the richest available view of a single component.
// ref follows the same resolution rules as Get.
func Describe(session *core.Session, ref string) (core.ComponentDescribeData, error) {
	c, err := resolveComponent(session, ref)
	if err != nil {
		return core.ComponentDescribeData{}, err
	}

	owners := session.GetTopicOwnersSnapshot()
	subscribed := make([]string, 0)
	for topic, ownerIDs := range owners {
		for _, id := range ownerIDs {
			if id == c.ID {
				subscribed = append(subscribed, topic)
				break
			}
		}
	}

	return core.ComponentDescribeData{
		SessionID: session.ID,
		Component: core.ComponentFullDetail{
			ID:               c.ID,
			Name:             c.Name,
			State:            string(c.State),
			TopicsSubscribed: subscribed,
			TopicsPublished:  []string{},
			Requires:         requiresMap(c.Capabilities.RequiresStreams),
			Metrics: core.ComponentMetrics{
				LastHeartbeat: c.LastHeartbeat,
			},
		},
	}, nil
}
