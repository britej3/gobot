package automation

import (
	"context"
	"net/http"
	"sync"
	"time"
)

type AutomationType string

const (
	AutomationWebhook   AutomationType = "webhook"
	AutomationSchedule  AutomationType = "schedule"
	AutomationEventType AutomationType = "event"
	AutomationAI        AutomationType = "ai"
	AutomationN8N       AutomationType = "n8n"
)

type Automation interface {
	Type() AutomationType
	Name() string
	Configure(config AutomationConfig) error
	Validate() error
	Start(ctx context.Context) error
	Stop() error
	Execute(ctx context.Context, event EventData) error
}

type AutomationConfig struct {
	Type        AutomationType  `json:"type"`
	Name        string          `json:"name"`
	Enabled     bool            `json:"enabled"`
	Endpoint    string          `json:"endpoint"`
	APIKey      string          `json:"api_key"`
	Webhooks    []WebhookConfig `json:"webhooks"`
	Schedule    ScheduleConfig  `json:"schedule"`
	Events      []EventConfig   `json:"events"`
	N8NConfig   N8NConfig       `json:"n8n_config"`
	RetryPolicy RetryPolicy     `json:"retry_policy"`
	Timeout     time.Duration   `json:"timeout"`
}

type WebhookConfig struct {
	Path    string            `json:"path"`
	Method  string            `json:"method"`
	Handler string            `json:"handler"`
	Filters map[string]string `json:"filters"`
	Enabled bool              `json:"enabled"`
}

type ScheduleConfig struct {
	CronExpression string        `json:"cron_expression"`
	Interval       time.Duration `json:"interval"`
	Timezone       string        `json:"timezone"`
	StartTime      time.Time     `json:"start_time"`
	EndTime        time.Time     `json:"end_time"`
}

type EventConfig struct {
	EventType string                 `json:"event_type"`
	Handler   string                 `json:"handler"`
	Filters   map[string]interface{} `json:"filters"`
	Enabled   bool                   `json:"enabled"`
}

type N8NConfig struct {
	BaseURL   string            `json:"base_url"`
	APIKey    string            `json:"api_key"`
	Workflows []N8NWorkflow     `json:"workflows"`
	AuthType  string            `json:"auth_type"`
	Headers   map[string]string `json:"headers"`
	Timeout   time.Duration     `json:"timeout"`
}

type N8NWorkflow struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	TriggerType string                 `json:"trigger_type"`
	Endpoint    string                 `json:"endpoint"`
	InputKey    string                 `json:"input_key"`
	OutputKey   string                 `json:"output_key"`
	Enabled     bool                   `json:"enabled"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type RetryPolicy struct {
	MaxRetries int           `json:"max_retries"`
	Delay      time.Duration `json:"delay"`
	MaxDelay   time.Duration `json:"max_delay"`
	Multiplier float64       `json:"multiplier"`
}

type EventData struct {
	Type      string
	Timestamp time.Time
	Data      map[string]interface{}
	Source    string
}

type AutomationResult struct {
	Success  bool
	Output   map[string]interface{}
	Duration time.Duration
	Errors   []error
}

type AutomationRegistry interface {
	Register(name string, factory AutomationFactory) error
	Get(name string) (Automation, bool)
	List() []AutomationType
	Create(cfg AutomationConfig) (Automation, error)
}

type AutomationFactory func() Automation

type N8NAutomation struct {
	cfg      AutomationConfig
	n8nCfg   N8NConfig
	client   *http.Client
	handlers map[string]EventHandler
	stopCh   chan struct{}
}

type EventHandler func(ctx context.Context, event EventData) error

func NewN8NAutomation() *N8NAutomation {
	return &N8NAutomation{
		handlers: make(map[string]EventHandler),
		stopCh:   make(chan struct{}),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (a *N8NAutomation) Type() AutomationType {
	return AutomationN8N
}

func (a *N8NAutomation) Name() string {
	return "n8n_automation"
}

func (a *N8NAutomation) Configure(config AutomationConfig) error {
	a.cfg = config
	a.n8nCfg = config.N8NConfig
	return nil
}

func (a *N8NAutomation) Validate() error {
	if a.n8nCfg.BaseURL == "" {
		return ErrMissingN8NEndpoint
	}
	return nil
}

func (a *N8NAutomation) Start(ctx context.Context) error {
	for _, workflow := range a.n8nCfg.Workflows {
		if !workflow.Enabled {
			continue
		}
		a.registerHandler(workflow)
	}
	return nil
}

func (a *N8NAutomation) Stop() error {
	close(a.stopCh)
	return nil
}

func (a *N8NAutomation) Execute(ctx context.Context, event EventData) error {
	handler, ok := a.handlers[event.Type]
	if !ok {
		return nil
	}
	return handler(ctx, event)
}

func (a *N8NAutomation) registerHandler(workflow N8NWorkflow) {
	switch workflow.TriggerType {
	case "trade_signal":
		a.handlers["trade_signal"] = a.handleTradeSignal
	case "market_data":
		a.handlers["market_data"] = a.handleMarketData
	case "position_update":
		a.handlers["position_update"] = a.handlePositionUpdate
	case "risk_alert":
		a.handlers["risk_alert"] = a.handleRiskAlert
	}
}

func (a *N8NAutomation) handleTradeSignal(ctx context.Context, event EventData) error {
	input := event.Data["signal"]
	_, err := a.callN8NWorkflow(ctx, "trade_signal", input)
	if err != nil {
		return err
	}
	return nil
}

func (a *N8NAutomation) handleMarketData(ctx context.Context, event EventData) error {
	input := event.Data["market"]
	_, err := a.callN8NWorkflow(ctx, "market_data", input)
	return err
}

func (a *N8NAutomation) handlePositionUpdate(ctx context.Context, event EventData) error {
	input := event.Data["position"]
	_, err := a.callN8NWorkflow(ctx, "position_update", input)
	return err
}

func (a *N8NAutomation) handleRiskAlert(ctx context.Context, event EventData) error {
	input := event.Data["alert"]
	_, err := a.callN8NWorkflow(ctx, "risk_alert", input)
	return err
}

func (a *N8NAutomation) callN8NWorkflow(ctx context.Context, workflowType string, input interface{}) (map[string]interface{}, error) {
	url := a.n8nCfg.BaseURL + "/webhook/" + workflowType

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-N8N-API-KEY", a.n8nCfg.APIKey)

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ErrN8NWorkflowFailed
	}

	var result map[string]interface{}
	return result, nil
}

type AutomationEngine struct {
	registry map[AutomationType]AutomationFactory
	mu       sync.RWMutex
}

func NewAutomationEngine() *AutomationEngine {
	return &AutomationEngine{
		registry: make(map[AutomationType]AutomationFactory),
	}
}

func (e *AutomationEngine) Register(t AutomationType, factory AutomationFactory) error {
	e.registry[t] = factory
	return nil
}

func (e *AutomationEngine) Get(t AutomationType) (Automation, bool) {
	factory, ok := e.registry[t]
	if !ok {
		return nil, false
	}
	return factory(), true
}

func (e *AutomationEngine) Create(cfg AutomationConfig) (Automation, error) {
	factory, ok := e.registry[cfg.Type]
	if !ok {
		return nil, ErrUnknownAutomation
	}

	automation := factory()
	if err := automation.Configure(cfg); err != nil {
		return nil, err
	}

	return automation, nil
}

func (e *AutomationEngine) List() []AutomationType {
	types := make([]AutomationType, 0, len(e.registry))
	for t := range e.registry {
		types = append(types, t)
	}
	return types
}

var (
	ErrMissingN8NEndpoint = &AutomationError{Message: "N8N endpoint is required"}
	ErrN8NWorkflowFailed  = &AutomationError{Message: "N8N workflow execution failed"}
	ErrUnknownAutomation  = &AutomationError{Message: "unknown automation type"}
)

type AutomationError struct {
	Message string
}

func (e *AutomationError) Error() string {
	return e.Message
}
