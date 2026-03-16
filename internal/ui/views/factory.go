package views

import (
	"github.com/Azure/durabletask-cli/internal/app"
)

// Factory creates views by resource name.
type Factory struct{}

// CreateView returns a new View for the given resource name.
func (f *Factory) CreateView(a *app.App, name string, params ...string) app.View {
	switch name {
	case "home":
		return NewHomeView(a)
	case "orchestrations":
		return NewOrchestrationsView(a)
	case "orchestration-detail":
		if len(params) >= 2 {
			return NewOrchestrationDetailView(a, params[0], params[1])
		}
		if len(params) == 1 {
			return NewOrchestrationDetailView(a, params[0], "")
		}
		return NewOrchestrationsView(a)
	case "entities":
		return NewEntitiesView(a)
	case "entity-detail":
		if len(params) >= 1 {
			return NewEntityDetailView(a, params[0])
		}
		return NewEntitiesView(a)
	case "schedules":
		return NewSchedulesView(a)
	case "workers":
		return NewWorkersView(a)
	case "agents":
		return NewAgentsView(a)
	case "agent-session":
		if len(params) >= 3 {
			return NewAgentSessionView(a, params[0], params[1], params[2])
		}
		return NewAgentsView(a)
	case "help":
		return NewHelpView(a)
	default:
		return NewHomeView(a)
	}
}
