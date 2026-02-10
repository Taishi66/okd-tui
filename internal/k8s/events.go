package k8s

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jclamy/okd-tui/internal/domain"
)

func (c *Client) ListEvents(ctx context.Context) ([]domain.EventInfo, error) {
	eventList, err := c.clientset.CoreV1().Events(c.namespace).List(ctx, metav1.ListOptions{
		Limit: 500,
	})
	if err != nil {
		return nil, classifyError(err, c.serverURL)
	}

	events := make([]domain.EventInfo, 0, len(eventList.Items))
	for _, evt := range eventList.Items {
		events = append(events, eventToEventInfo(evt))
	}
	return events, nil
}

func (c *Client) WatchEvents(ctx context.Context) (<-chan domain.WatchEvent, error) {
	watcher, err := c.clientset.CoreV1().Events(c.namespace).Watch(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, classifyError(err, c.serverURL)
	}
	ch := make(chan domain.WatchEvent)
	go func() {
		defer close(ch)
		defer watcher.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-watcher.ResultChan():
				if !ok {
					return
				}
				evt, ok := event.Object.(*corev1.Event)
				if !ok {
					continue
				}
				info := eventToEventInfo(*evt)
				wType := domain.WatchEventType(string(event.Type))
				select {
				case ch <- domain.WatchEvent{Type: wType, Resource: "event", Event: &info}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return ch, nil
}

func eventToEventInfo(evt corev1.Event) domain.EventInfo {
	obj := fmt.Sprintf("%s/%s", evt.InvolvedObject.Kind, evt.InvolvedObject.Name)

	age := ""
	if !evt.LastTimestamp.IsZero() {
		age = formatAge(evt.LastTimestamp.Time)
	} else if !evt.EventTime.IsZero() {
		age = formatAge(evt.EventTime.Time)
	}

	createdAt := evt.LastTimestamp.Time
	if createdAt.IsZero() {
		createdAt = evt.EventTime.Time
	}

	return domain.EventInfo{
		Type:      evt.Type,
		Reason:    evt.Reason,
		Message:   evt.Message,
		Object:    obj,
		Namespace: evt.Namespace,
		Age:       age,
		Count:     evt.Count,
		CreatedAt: createdAt,
	}
}
