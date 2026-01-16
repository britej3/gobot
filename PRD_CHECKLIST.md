# GOBOT Autonomous Crypto Futures Trading Bot - PRD & Implementation Checklist

**Version**: 3.0  
**Status**: In Development  
**Target Launch**: Q1 2026  
**Max Capital**: 26 USDT  

---

## Executive Summary

### Product Vision
Build a fully autonomous crypto futures trading bot that intelligently selects mid-cap assets, analyzes them using QuantCrawler AI, and executes trades with API sniffing guard rails to achieve 15-25% monthly returns with minimal risk.

### Key Objectives
- **Autonomous Operation**: Fully automated trading with minimal human intervention
- **Smart Asset Selection**: Mid-cap coins with high probability setups
- **AI-Driven Analysis**: QuantCrawler integration for market insights
- **Risk-First Approach**: Comprehensive guard rails and stop-loss mechanisms
- **API Protection**: Advanced anti-detection measures to avoid rate limiting
- **Performance Target**: 15-25% monthly returns with <2% max drawdown

### Success Metrics
- **Win Rate**: 65-75%
- **Monthly Return**: 15-25%
- **Max Drawdown**: <2%
- **System Uptime**: >99.5%
- **API Rate Limit Hits**: <1 per day
- **Ghost Positions**: 0 per day

---

## Table of Contents

1. [Product Requirements](#1-product-requirements)
2. [Technical Architecture](#2-technical-architecture)
3. [Implementation Checklist](#3-implementation-checklist)
4. [Risk Management](#4-risk-management)
5. [Testing & Validation](#5-testing--validation)
6. [Deployment](#6-deployment)
7. [Monitoring & Operations](#7-monitoring--operations)
8. [Success Criteria](#8-success-criteria)

---

## 1. Product Requirements

### 1.1 Core Features

#### FR-1: Asset Selection Engine
**Priority**: P0  
**Description**: Automatically screen and select mid-cap crypto assets for trading

**Requirements**:
- Screen assets with $10M-$100M 24h volume
- Filter assets with price range $0.01-$10
- Calculate confidence scores (0.0-1.0) for each asset
- Filter assets with confidence > 0.75
- Return top 10 assets for analysis
- Run screening every 15 minutes

**Acceptance Criteria**:
- [ ] Can screen 100+ assets in <5 seconds
- [ ] Confidence score calculation is accurate
- [ ] Filtering works correctly
- [ ] Returns assets in descending confidence order

---

#### FR-2: Capital Allocation System
**Priority**: P0  
**Description**: Intelligently allocate 100 USDT max capital across selected assets

**Requirements**:
- Max total capital: 100 USDT
- Min capital per position: 10 USDT
- Max concurrent positions: 3
- Allocation based on confidence scores
- Proportional distribution among high-confidence assets
- Respect daily/weekly loss limits
- Support multiple position sizing options from QuantCrawler

**Acceptance Criteria**:
- [ ] Never exceeds 30 USDT total exposure
- [ ] Minimum 10 USDT per position enforced
- [ ] Maximum 3 concurrent positions enforced
- [ ] Allocation proportional to confidence
- [ ] Daily loss limit ($30) enforced
- [ ] Weekly loss limit ($100) enforced
- [ ] Support QuantCrawler position sizing options

---

#### FR-3: QuantCrawler Integration
**Priority**: P0  
**Description**: Integrate with QuantCrawler for AI-powered market analysis

**Requirements**:
- Capture screenshots (1m, 5m, 15m timeframes)
- Send to QuantCrawler via N8N workflow with **structured prompt format**
- Request specific output format for structured data extraction:
  - Prompt QuantCrawler to return analysis in machine-readable format
  - Request JSON format for key trading parameters
  - Request structured entry/exit/stop/target levels
  - Request explicit confidence values and risk metrics
- Receive detailed plain text analysis including:
  - Ticker symbol and current price
  - Entry price (with order type: LIMIT/MARKET)
  - Direction (LONG/SHORT with emoji indicators)
  - Confidence score (0-100%, capped at 75% for limit orders)
  - Recommendation text explanation
  - Multiple position sizing options:
    - Option 1: Single contract (wider stop)
    - Option 2: Multiple contracts (tighter stop, POPULAR)
    - Option 3: Chart structure based (PRO TRADER FAVORITE)
  - Contract specifications (tick size, tick value)
  - Stop and target distances (in points and price levels)
  - Risk per contract and total risk
  - Risk-reward ratios
  - Timeframe analysis (15m, 5m, 1m)
  - Key levels (support, resistance)
  - Invalidation conditions
  - Execution instructions
  - Confluence score (e.g., "3/3 timeframes agree")
- Parse plain text response into structured data
- Validate analysis confidence > 70%
- Support multiple position sizing options
- Timeout: 3 minutes per analysis

**Acceptance Criteria**:
- [ ] Screenshots captured successfully
- [ ] QuantCrawler analysis received within 3 minutes
- [ ] Plain text response parsed correctly
- [ ] All position sizing options extracted
- [ ] Contract specifications parsed correctly
- [ ] Stop and target levels extracted accurately
- [ ] Confidence validation works (convert 75% to 0.75)
- [ ] Direction parsed correctly (with emoji support)
- [ ] Entry price and order type extracted
- [ ] Fallback to hold if confidence < 70%
- [ ] No mock analysis in production
- [ ] Structured prompt format used for requests
- [ ] JSON format requested for key parameters

---

#### FR-4: Trade Execution Engine
**Priority**: P0  
**Description**: Execute approved trades with precision and speed

**Requirements**:
- Execute market orders
- Set leverage before order placement
- Calculate position size based on capital and leverage
- Validate order before submission (quantity, price, notional)
- Apply anti-sniffer jitter (5-25ms)
- Apply request delay (100-300ms)
- Set stop loss (0.5%)
- Set take profit (1.5%)
- Confirm order fill
- Handle partial fills
- Retry on temporary failures (max 3 attempts)

**Acceptance Criteria**:
- [ ] Orders execute within 500ms
- [ ] Leverage set correctly before order
- [ ] Position size calculated accurately
- [ ] Order validation passes for valid orders
- [ ] Stop loss and take profit set correctly
- [ ] Order confirmation received
- [ ] Partial fills handled correctly
- [ ] Retry logic works

---

#### FR-5: Position Management
**Priority**: P0  
**Description**: Monitor and manage open positions with AI-assisted decision making

**Requirements**:
- Monitor positions every 30 seconds
- Calculate unrealized PnL
- Assess position health with AI (0-100 score)
- Close positions if:
  - Stop loss hit (0.5% loss)
  - Take profit hit (1.5% gain)
  - AI health score < 45
  - Loss > 0.2% + AI score < 50
- Update trailing stops if enabled
- Log all position changes

**Acceptance Criteria**:
- [ ] Positions monitored every 30 seconds
- [ ] PnL calculated accurately
- [ ] AI health assessment works
- [ ] Stop loss triggers correctly
- [ ] Take profit triggers correctly
- [ ] AI-based closures work
- [ ] Trailing stops update correctly
- [ ] All changes logged

---

#### FR-6: API Sniffing Guard Rails
**Priority**: P0  
**Description**: Protect against API rate limiting and detection

**Requirements**:
- Request throttling (8 RPS, burst 16)
- Anti-sniffer jitter (5-25ms normal distribution)
- Request delay (100-300ms random)
- Signature variance (0.01)
- User agent rotation (4 different agents)
- Random IP spoofing (X-MBX-USER-IP header)
- Circuit breaker (5 failures = 30s cooldown)
- Time synchronization (<1000ms offset)

**Acceptance Criteria**:
- [ ] Rate limiting enforced
- [ ] Jitter within 5-25ms range
- [ ] Request delay within 100-300ms
- [ ] Signature variance applied
- [ ] User agents rotate correctly
- [ ] Random IP header added
- [ ] Circuit breaker triggers correctly
- [ ] Time offset <1000ms

---

#### FR-7: Risk Management System
**Priority**: P0  
**Description**: Comprehensive risk controls to protect capital

**Requirements**:
- Max risk per trade: 2%
- Max daily loss: $30
- Max weekly loss: $100
- Max concurrent positions: 3
- Stop loss: 0.5%
- Take profit: 1.5%
- Trailing stop: 0.3% (optional)
- Emergency stop (kill switch)
- Position size limits
- Leverage limits (10-25x)

**Acceptance Criteria**:
- [ ] All risk limits enforced
- [ ] Stop loss triggers at 0.5%
- [ ] Take profit triggers at 1.5%
- [ ] Daily loss limit stops trading
- [ ] Weekly loss limit stops trading
- [ ] Emergency stop works
- [ ] Position size limits enforced
- [ ] Leverage limits enforced

---

#### FR-8: Monitoring & Alerting
**Priority**: P1  
**Description**: Real-time monitoring and alerting system

**Requirements**:
- HTTP dashboard on :8080
- Metrics endpoint (/metrics)
- Health check endpoint (/health)
- Telegram alerts for:
  - Trade executions
  - P&L milestones
  - Risk breaches
  - System errors
  - Emergency stops
- Real-time P&L tracking
- Position status updates

**Acceptance Criteria**:
- [ ] Dashboard accessible on :8080
- [ ] Metrics endpoint returns correct data
- [ ] Health check returns 200
- [ ] Telegram alerts sent correctly
- [ ] P&L tracked accurately
- [ ] Position status updates work

---

### 1.2 Non-Functional Requirements

#### NFR-1: Performance
- Asset screening: <5 seconds
- QuantCrawler analysis: <3 minutes
- Order execution: <500ms
- Position monitoring: <100ms
- API response time: <200ms (95th percentile)

#### NFR-2: Reliability
- System uptime: >99.5%
- Order success rate: >99%
- API error rate: <1%
- Ghost positions: 0 per day

#### NFR-3: Scalability
- Support 100+ concurrent assets
- Handle 50+ trades per day
- Process 1000+ API requests per hour

#### NFR-4: Security
- API keys stored securely (environment variables)
- No hardcoded credentials
- HTTPS only for external APIs
- IP whitelisting support
- Kill switch for emergency stop

#### NFR-5: Maintainability
- Modular architecture
- Clear separation of concerns
- Comprehensive logging
- Easy configuration updates
- Well-documented code

---

## 2. Technical Architecture

### 2.1 System Components

```
┌─────────────────────────────────────────────────────────────────┐
│                         GOBOT SYSTEM                             │
└─────────────────────────────────────────────────────────────────┘
                                │
        ┌───────────────────────┼───────────────────────┐
        │                       │                       │
┌───────▼────────┐    ┌────────▼────────┐    ┌────────▼────────┐
│   Screener     │    │   QuantCrawler  │    │    Striker      │
│   Engine       │    │   Integration   │    │   Engine        │
└────────────────┘    └─────────────────┘    └─────────────────┘
        │                       │                       │
        │                       │                       │
┌───────▼────────┐    ┌────────▼────────┐    ┌────────▼────────┐
│   Allocator    │    │   Brain Engine   │    │   Position      │
│   Engine       │    │   (AI Decision)  │    │   Manager       │
└────────────────┘    └─────────────────┘    └─────────────────┘
        │                       │                       │
        └───────────────────────┼───────────────────────┘
                                │
                    ┌───────────▼───────────┐
                    │   Guard Rails        │
                    │   - Throttler        │
                    │   - Anti-Sniffer      │
                    │   - Circuit Breaker  │
                    │   - Time Sync         │
                    └──────────────────────┘
                                │
                    ┌───────────▼───────────┐
                    │   Binance API        │
                    │   (Futures)          │
                    └──────────────────────┘
```

### 2.2 Data Flow

```
1. Asset Screening
   Screener → Filter mid-caps → Calculate confidence → Return top 10

2. Capital Allocation
   Allocator → Distribute $100 → Max 3 positions → Return allocations

3. QuantCrawler Analysis
   Capture screenshots → Send to N8N → Receive analysis → Validate confidence

4. Trade Approval
   Validate confidence → Check direction → Approve trades

5. Trade Execution
   Apply guard rails → Set leverage → Calculate size → Execute order → Set SL/TP

6. Position Monitoring
   Get positions → Calculate PnL → AI health check → Close if needed
```

### 2.3 Technology Stack

**Backend**:
- Go 1.25.4
- Binance Futures API (go-binance/v2)
- WebSocket for real-time data

**AI/ML**:
- QuantCrawler (external service)
- Ollama (local LLM: qwen3:0.6b)
- Brain Engine (decision making)

**Automation**:
- N8N (workflow orchestration)
- Puppeteer (screenshot capture)

**Infrastructure**:
- Docker & Docker Compose
- Systemd (Linux) / launchd (macOS)
- PostgreSQL (future: historical data)

**Monitoring**:
- HTTP dashboard
- Telegram Bot API
- Structured logging (logrus)

---

## 3. Implementation Checklist

### Phase 1: Foundation & Infrastructure (Week 1)

#### 1.1 Critical Bug Fixes
- [ ] **FIX-001**: Fix Binance API signature algorithm in `infra/binance/client.go:594`
  - Replace UUID-based signature with HMAC-SHA256
  - Test authentication on testnet
  - Verify signature matches Binance requirements

- [ ] **FIX-002**: Use hardened client in main application (`cmd/cobot/main.go`)
  - Replace `binance.New()` with `binance.NewHardenedClient()`
  - Configure anti-sniffer parameters
  - Test on testnet

- [ ] **FIX-003**: Fix position sizing calculation (`internal/striker/striker.go:490`)
  - Add account balance consideration
  - Implement risk per trade (2%)
  - Add maximum position limits
  - Test with various account sizes

#### 1.2 WebSocket Integration
- [ ] **WS-001**: Integrate WebSocket stream manager
  - Import `internal/platform/ws_stream.go`
  - Add to main application
  - Configure symbols for streaming
  - Test real-time data reception

- [ ] **WS-002**: Implement real-time kline processing
  - Process WebSocket kline events
  - Update market data
  - Trigger analysis on new data
  - Test with live data

#### 1.3 Time Synchronization
- [ ] **TIME-001**: Implement time sync module
  - Create `internal/time/sync.go`
  - Sync with Binance server time
  - Calculate offset
  - Validate offset <1000ms

- [ ] **TIME-002**: Integrate time sync in API calls
  - Use synchronized timestamps
  - Update all API calls
  - Test on testnet

---

### Phase 2: Asset Selection Engine (Week 1-2)

#### 2.1 Mid-Cap Screener
- [ ] **SCREEN-001**: Create mid-cap screener
  - Create `internal/screener/midcap.go`
  - Implement screening logic
  - Add confidence calculation
  - Test screening performance

- [ ] **SCREEN-002**: Add volume filtering
  - Filter by 24h volume ($10M-$100M)
  - Test with various assets
  - Validate filtering accuracy

- [ ] **SCREEN-003**: Add price filtering
  - Filter by price range ($0.01-$10)
  - Test with various assets
  - Validate filtering accuracy

- [ ] **SCREEN-004**: Implement confidence scoring
  - Calculate volume score (30%)
  - Calculate price action score (30%)
  - Calculate volatility score (20%)
  - Calculate liquidity score (20%)
  - Test scoring accuracy

- [ ] **SCREEN-005**: Add sorting and filtering
  - Sort by confidence (descending)
  - Filter by confidence > 0.75
  - Return top 10 assets
  - Test with real data

#### 2.2 Capital Allocation
- [ ] **ALLOC-001**: Create allocator engine
  - Create `internal/allocation/allocator.go`
  - Implement allocation logic
  - Add position limits
  - Test allocation accuracy

- [ ] **ALLOC-002**: Implement proportional allocation
  - Allocate based on confidence
  - Ensure minimum $10 per position
  - Enforce max 3 positions
  - Test with various scenarios

- [ ] **ALLOC-003**: Add leverage calculation
  - Calculate leverage based on confidence
  - Enforce 10-25x range
  - Test leverage accuracy

- [ ] **ALLOC-004**: Add QuantCrawler position sizing support
  - Parse multiple position sizing options
  - Select optimal option based on risk tolerance
  - Support contract specifications
  - Test with various scenarios

---

### Phase 3: QuantCrawler Integration (Week 2)

#### 3.1 Enhanced QuantCrawler Client
- [ ] **QC-001**: Create enhanced QuantCrawler client
  - Create `services/quantcrawler/enhanced.go`
  - Implement analysis request
  - Add market data support
  - Test with N8N workflow

- [ ] **QC-001a**: Implement structured prompt formatting
  - Create prompt templates for different analysis types
  - Add JSON format request for key parameters
  - Implement prompt builder with parameters:
    - Symbol, timeframes, risk amount, preferred option
    - Output format (JSON/plain/mixed)
  - Add prompt validation and sanitization
  - Test prompt templates with various scenarios
  - Document prompt format requirements

- [ ] **QC-002**: Implement screenshot capture
  - Capture 1m, 5m, 15m timeframes
  - Handle capture failures
  - Test with various symbols

- [ ] **QC-003**: Add plain text response parser
  - Parse ticker information
  - Extract direction (with emoji)
  - Extract confidence score (convert to 0.0-1.0 range)
  - Parse entry price and order type
  - Extract position sizing options (all 3 options)
  - Parse contract specifications (tick size, tick value)
  - Extract stop and target levels (points and price)
  - Calculate risk per contract and total risk
  - Extract risk-reward ratios
  - Parse timeframe analysis
  - Extract key levels (support, resistance)
  - Parse invalidation conditions
  - Extract execution instructions
  - Parse confluence score
  - **Extract JSON blocks** (when present in response)
  - **Parse JSON format for key parameters**
  - **Implement hybrid parsing** (JSON + plain text)
  - **Fallback to plain text if JSON extraction fails**
  - **Validate consistency between JSON and plain text**
  - Test parsing with various response formats

- [ ] **QC-004**: Add analysis validation
  - Validate confidence > 70%
  - Validate all required fields present
  - Validate position sizing options valid
  - Reject invalid analyses
  - Test validation logic

- [ ] **QC-005**: Implement batch analysis
  - Analyze multiple assets in parallel
  - Handle failures gracefully
  - Test with multiple assets

#### 3.2 Workflow Integration
- [ ] **WF-001**: Create QuantCrawler workflow
  - Create `internal/workflow/quantcrawler.go`
  - Implement complete workflow
  - Add plain text parsing
  - Add position sizing option selection
  - Add trade approval logic
  - Test end-to-end

- [ ] **WF-002**: Implement position sizing option selector
  - Parse all 3 position sizing options
  - Select optimal option based on:
    - Risk tolerance (fixed $200 risk or percentage-based)
    - Trading style (scalping vs swing)
    - Risk-reward ratio
  - Validate option selection
  - Test with various scenarios

- [ ] **WF-003**: Implement periodic execution
  - Run workflow every 15 minutes
  - Handle execution errors
  - Test periodic execution

- [ ] **WF-004**: Add trade execution integration
  - Connect workflow to striker
  - Pass selected position sizing option
  - Pass contract specifications
  - Test execution flow

#### 3.3 QuantCrawler Response Format
- [ ] **QC-FORMAT-001**: Document QuantCrawler plain text format
  - Create documentation of response structure
  - Include sample response (MMTU example)
  - Document all fields and their formats
  - Add parsing examples

- [ ] **QC-FORMAT-002**: Create response parser
  - Create `services/quantcrawler/parser.go`
  - Implement plain text parsing logic
  - Extract all fields from response
  - Handle multiple position sizing options
  - Test with various response formats

---

### Phase 4: API Sniffing Guard Rails (Week 2)

#### 4.1 Anti-Sniffer System
- [ ] **AS-001**: Enhance anti-sniffer module
  - Update `pkg/stealth/anti_sniffer.go`
  - Implement normal distribution jitter (5-25ms)
  - Add request delay (100-300ms)
  - Test jitter distribution

- [ ] **AS-002**: Add user agent rotation
  - Implement 4 different user agents
  - Rotate on each request
  - Test rotation logic

- [ ] **AS-003**: Add IP spoofing
  - Generate random IP addresses
  - Add X-MBX-USER-IP header
  - Test IP generation

- [ ] **AS-004**: Add signature variance
  - Add small variance to signature timestamp
  - Test variance application

#### 4.2 Request Throttling
- [ ] **THROTTLE-001**: Create throttler
  - Create `pkg/throttle/throttler.go`
  - Implement rate limiting (8 RPS, burst 16)
  - Add minimum interval with jitter
  - Test throttling accuracy

- [ ] **THROTTLE-002**: Integrate throttler
  - Add throttler to all API calls
  - Test rate limiting
  - Verify burst handling

#### 4.3 Circuit Breaker
- [ ] **CB-001**: Integrate circuit breaker
  - Use existing `pkg/circuitbreaker/circuitbreaker.go`
  - Configure for 5 failures, 30s cooldown
  - Test circuit breaker triggers

---

### Phase 5: Trade Execution Engine (Week 2-3)

#### 5.1 Enhanced Striker
- [ ] **STRIKER-001**: Create enhanced striker
  - Create `internal/striker/enhanced.go`
  - Implement complete execution flow
  - Add all guard rails
  - Test execution accuracy

- [ ] **STRIKER-002**: Add leverage setting
  - Set leverage before order placement
  - Handle leverage errors
  - Test leverage setting

- [ ] **STRIKER-003**: Implement position sizing from QuantCrawler
  - Parse QuantCrawler position sizing options
  - Calculate contracts based on option selected
  - Validate sizing accuracy
  - Support 10 USDT minimum position
  - Test with various scenarios

- [ ] **STRIKER-004**: Add order type support
  - Support LIMIT orders (wait for pullback)
  - Support MARKET orders (immediate execution)
  - Validate order type compatibility
  - Test both order types

- [ ] **STRIKER-005**: Add contract specification handling
  - Parse tick size and tick value
  - Calculate position size in contracts
  - Validate contract specifications
  - Test with various contracts

- [ ] **STRIKER-006**: Add order validation
  - Validate before submission
  - Check quantity, price, notional
  - Test validation logic

- [ ] **STRIKER-007**: Implement stop loss & take profit
  - Set stop loss (from QuantCrawler or 0.5% default)
  - Set take profit (from QuantCrawler or 1.5% default)
  - Support point-based and percentage-based levels
  - Test SL/TP placement

- [ ] **STRIKER-008**: Add order confirmation
  - Verify order fill
  - Check fill price
  - Handle partial fills
  - Test confirmation logic

#### 5.2 Order Validation
- [ ] **VALID-001**: Create order validator
  - Create `internal/validation/order.go`
  - Implement quantity validation
  - Implement price validation
  - Implement notional validation
  - Test validation accuracy

---

### Phase 6: Position Management (Week 3)

#### 6.1 Enhanced Position Manager
- [ ] **POS-001**: Create enhanced position manager
  - Create `internal/position/enhanced.go`
  - Implement 30-second monitoring
  - Add AI health assessment
  - Test monitoring accuracy

- [ ] **POS-002**: Implement PnL calculation
  - Calculate unrealized PnL
  - Calculate PnL percentage
  - Test calculation accuracy

- [ ] **POS-003**: Add AI health assessment
  - Get market trend
  - Assess win probability
  - Calculate health score
  - Test assessment accuracy

- [ ] **POS-004**: Implement position closure
  - Close positions on SL hit
  - Close positions on TP hit
  - Close positions on AI score < 45
  - Test closure logic

- [ ] **POS-005**: Add trailing stop
  - Update trailing stops
  - Test trailing stop logic

---

### Phase 7: Main Application (Week 3)

#### 7.1 Application Orchestration
- [ ] **APP-001**: Rewrite main application
  - Rewrite `cmd/gobot/main.go`
  - Initialize all components
  - Wire components together
  - Test complete startup

- [ ] **APP-002**: Add component initialization
  - Initialize screener
  - Initialize allocator
  - Initialize QuantCrawler
  - Initialize striker
  - Initialize position manager
  - Initialize guard rails
  - Test all components

- [ ] **APP-003**: Add periodic workflows
  - Start QuantCrawler workflow (15 min)
  - Start position monitoring (30 sec)
  - Start real-time data
  - Test all workflows

- [ ] **APP-004**: Add graceful shutdown
  - Handle SIGINT/SIGTERM
  - Stop all components
  - Clean up resources
  - Test shutdown

---

### Phase 8: Testing & Validation (Week 3-4)

#### 8.1 Unit Tests
- [ ] **TEST-001**: Test screener
  - Test asset screening
  - Test confidence calculation
  - Test filtering logic

- [ ] **TEST-002**: Test allocator
  - Test capital allocation
  - Test leverage calculation
  - Test position limits

- [ ] **TEST-003**: Test QuantCrawler client
  - Test screenshot capture
  - Test analysis request
  - Test validation logic

- [ ] **TEST-004**: Test striker
  - Test position sizing
  - Test leverage setting
  - Test order execution
  - Test SL/TP placement

- [ ] **TEST-005**: Test position manager
  - Test PnL calculation
  - Test AI health assessment
  - Test position closure

- [ ] **TEST-006**: Test guard rails
  - Test throttler
  - Test anti-sniffer
  - Test circuit breaker

#### 8.2 Integration Tests
- [ ] **IT-001**: Test complete workflow
  - Test screening → allocation → analysis → execution
  - Test on testnet
  - Validate complete flow

- [ ] **IT-002**: Test order validation
  - Test order validation with various scenarios
  - Test on testnet

- [ ] **IT-003**: Test position lifecycle
  - Test order → position → SL/TP → closure
  - Test on testnet

#### 8.3 Testnet Validation
- [ ] **TN-001**: Run 48-hour testnet test
  - Deploy to testnet
  - Run for 48 hours
  - Monitor all trades
  - Validate all features

- [ ] **TN-002**: Validate performance
  - Measure win rate
  - Measure P&L
  - Measure API usage
  - Validate against targets

- [ ] **TN-003**: Validate risk controls
  - Test stop losses
  - Test take profits
  - Test daily limits
  - Test emergency stop

---

### Phase 9: Deployment (Week 4)

#### 9.1 Configuration
- [ ] **CONFIG-001**: Create production config
  - Create `config/production.yaml`
  - Set all parameters
  - Validate configuration

- [ ] **CONFIG-002**: Configure environment variables
  - Set API keys
  - Set Telegram credentials
  - Set kill switch password
  - Validate all variables

#### 9.2 Deployment Scripts
- [ ] **DEPLOY-001**: Create systemd service
  - Create `scripts/gobot.service`
  - Test service start/stop
  - Validate auto-restart

- [ ] **DEPLOY-002**: Create deployment script
  - Create `scripts/deploy.sh`
  - Test deployment process
  - Validate deployment

- [ ] **DEPLOY-003**: Create monitoring setup
  - Set up dashboard
  - Configure Telegram alerts
  - Test monitoring

#### 9.3 Pre-Deployment Checklist
- [ ] All critical bugs fixed
- [ ] All unit tests passing
- [ ] All integration tests passing
- [ ] 48-hour testnet validation complete
- [ ] Performance targets met
- [ ] Risk controls validated
- [ ] Monitoring configured
- [ ] Emergency stop tested
- [ ] API keys configured
- [ ] Documentation complete

---

## 4. Risk Management

### 4.1 Built-in Protections

#### Position Limits
- **Max Total Capital**: 100 USDT
- **Min Per Position**: 10 USDT
- **Max Per Position**: 40 USDT
- **Max Concurrent Positions**: 3
- **Max Risk Per Trade**: 2%
- **Support Multiple Position Sizing Options**: Yes (from QuantCrawler)

#### Stop Loss & Take Profit
- **Stop Loss**: 0.5% (automatic closure)
- **Take Profit**: 1.5% (automatic closure)
- **Trailing Stop**: 0.3% (optional)
- **No Manual Override**: Automatic enforcement

#### Daily/Weekly Limits
- **Max Daily Loss**: $30 (trading halt)
- **Max Daily Trades**: 5
- **Max Weekly Loss**: $100 (trading halt)

#### API Protection
- **Rate Limiting**: 8 RPS, burst 16
- **Circuit Breaker**: 5 failures = 30s cooldown
- **Anti-Sniffer**: 5-25ms jitter
- **Request Delay**: 100-300ms
- **Time Sync**: <1000ms offset

### 4.2 Emergency Controls

#### Kill Switch
- **File-based**: `/tmp/gobot_kill_switch`
- **Telegram**: `/panic` command
- **Immediate Effect**: Close all positions, halt trading

#### Safe Mode
- **Trigger**: Daily loss limit breach
- **Action**: Stop new entries, close existing positions
- **Recovery**: 24-hour cooldown

### 4.3 Risk Validation Checklist

Before each trade:
- [ ] Account balance sufficient
- [ ] Daily loss limit not breached
- [ ] Weekly loss limit not breached
- [ ] Max positions not exceeded
- [ ] Position size within limits
- [ ] Leverage within limits
- [ ] Stop loss configured
- [ ] Take profit configured
- [ ] API rate limit not exceeded
- [ ] Circuit breaker not triggered

---

## 5. Testing & Validation

### 5.1 Test Strategy

#### Unit Tests
- **Coverage Target**: >80%
- **Framework**: Go testing
- **CI/CD**: Automated on every commit

#### Integration Tests
- **Environment**: Testnet
- **Duration**: 48 hours continuous
- **Scenarios**: Normal, high volatility, API errors

#### Load Tests
- **Target**: 100 trades/hour
- **Duration**: 1 hour
- **Validation**: No errors, <500ms latency

### 5.2 Test Scenarios

#### Scenario 1: Normal Trading
- **Setup**: 100 USDT, 3 positions
- **Actions**: Execute 10 trades
- **Expected**: 6-8 wins, 2-4 losses, P&L +5-15%

#### Scenario 2: High Volatility
- **Setup**: 100 USDT, high volatility market
- **Actions**: Execute 10 trades
- **Expected**: Reduced position sizes, wider stops

#### Scenario 3: API Rate Limit
- **Setup**: 100 USDT, trigger rate limit
- **Actions**: Attempt 20 rapid trades
- **Expected**: Circuit breaker triggers, cooldown applied

#### Scenario 4: Emergency Stop
- **Setup**: 30 USDT, 3 open positions
- **Actions**: Trigger kill switch
- **Expected**: All positions closed, trading halted

### 5.3 Validation Checklist

#### Pre-Testnet Validation
- [ ] All unit tests passing
- [ ] Code review complete
- [ ] Security audit complete
- [ ] Performance profiling complete
- [ ] Documentation complete

#### Testnet Validation
- [ ] 48-hour continuous run
- [ ] 50+ trades executed
- [ ] Win rate 65-75%
- [ ] No ghost positions
- [ ] No API errors
- [ ] All SL/TP triggered correctly
- [ ] Emergency stop tested
- [ ] Monitoring validated

#### Pre-Mainnet Validation
- [ ] Testnet validation passed
- [ ] Performance targets met
- [ ] Risk controls validated
- [ ] Monitoring configured
- [ ] Deployment tested
- [ ] Team trained
- [ ] Runbooks created

---

## 6. Deployment

### 6.1 Deployment Strategy

#### Phased Rollout
1. **Phase 1**: Testnet (48 hours)
2. **Phase 2**: Mainnet - Small Capital ($10 minimum position)
3. **Phase 3**: Mainnet - Full Capital ($100)

#### Rollback Plan
- Immediate rollback if:
  - Win rate < 50%
  - Daily loss > $50
  - Ghost positions detected
  - API errors > 5%

### 6.2 Deployment Checklist

#### Pre-Deployment
- [ ] All tests passing
- [ ] Testnet validation complete
- [ ] API keys configured
- [ ] Monitoring configured
- [ ] Team trained
- [ ] Runbooks ready
- [ ] Emergency procedures tested

#### Deployment
- [ ] Backup current version
- [ ] Deploy new version
- [ ] Start monitoring
- [ ] Validate startup
- [ ] Test emergency stop
- [ ] Monitor for 24 hours

#### Post-Deployment
- [ ] Monitor for 24 hours
- [ ] Review all trades
- [ ] Check P&L accuracy
- [ ] Validate SL/TP triggers
- [ ] Check API usage
- [ ] Review system health
- [ ] Update documentation

### 6.3 Deployment Scripts

#### Deployment Script
```bash
#!/bin/bash
# scripts/deploy.sh

echo "🚀 Deploying GOBOT..."

# Backup
cp gobot gobot.backup.$(date +%Y%m%d_%H%M%S)

# Build
go build -o gobot ./cmd/gobot

# Test
./gobot --test

# Deploy
sudo systemctl stop gobot
sudo cp gobot /opt/gobot/
sudo systemctl start gobot

# Monitor
sudo journalctl -u gobot -f
```

#### Rollback Script
```bash
#!/bin/bash
# scripts/rollback.sh

echo "🔄 Rolling back GOBOT..."

# Stop
sudo systemctl stop gobot

# Restore
sudo cp gobot.backup.* /opt/gobot/gobot

# Start
sudo systemctl start gobot

# Monitor
sudo journalctl -u gobot -f
```

---

## 7. Monitoring & Operations

### 7.1 Monitoring Dashboard

#### Metrics Endpoint
- **URL**: http://localhost:8080/metrics
- **Update Frequency**: 5 seconds
- **Data**:
  - Total balance
  - Available margin
  - Unrealized P&L
  - Realized P&L
  - Open positions
  - Total trades
  - Win rate
  - System health

#### Health Check
- **URL**: http://localhost:8080/health
- **Response**: 200 OK
- **Purpose**: Load balancer health check

### 7.2 Alerts

#### Telegram Alerts
- **Trade Executions**: Every trade
- **P&L Milestones**: Every $10 gain/loss
- **Risk Breaches**: Daily/weekly limits
- **System Errors**: API errors, component failures
- **Emergency Stops**: Kill switch activation

#### Alert Levels
- **INFO**: Trade executions, P&L milestones
- **WARN**: Risk breaches, high API error rate
- **ERROR**: System failures, emergency stops
- **CRITICAL**: Multiple failures, system down

### 7.3 Logging

#### Log Files
- **Main Log**: `logs/gobot.log`
- **Trade Log**: `logs/trades_mainnet.log`
- **Audit Log**: `logs/mainnet_audit.log`
- **Error Log**: `logs/error.log`

#### Log Levels
- **DEBUG**: Detailed debugging info
- **INFO**: General information
- **WARN**: Warning messages
- **ERROR**: Error messages
- **FATAL**: Critical failures

### 7.4 Operations Runbook

#### Daily Operations
- [ ] Check system health (8 AM)
- [ ] Review trades from previous day
- [ ] Check P&L
- [ ] Verify no ghost positions
- [ ] Review API usage
- [ ] Check system logs

#### Weekly Operations
- [ ] Review weekly performance
- [ ] Analyze winning trades
- [ ] Analyze losing trades
- [ ] Update risk parameters if needed
- [ ] Review system health
- [ ] Backup logs and state

#### Monthly Operations
- [ ] Comprehensive performance review
- [ ] Strategy optimization
- [ ] System updates
- [ ] Security audit
- [ ] Documentation update
- [ ] Team training

---

## 8. Success Criteria

### 8.1 Performance Metrics

#### Monthly Targets
- **Win Rate**: 65-75%
- **Average Win**: 1.2%
- **Average Loss**: -0.5%
- **Risk-Reward Ratio**: 2.4:1
- **Monthly Trades**: 60-80
- **Monthly Return**: 15-25%
- **Max Drawdown**: <2%

#### Daily Targets
- **Max Daily Loss**: $30
- **Max Daily Trades**: 5
- **Max Concurrent Positions**: 3
- **Position Size**: $10-40 per trade
- **Leverage**: 10-25x
- **Min Position**: $10 (for small capital deployment)

### 8.2 Quality Metrics

#### System Reliability
- **Uptime**: >99.5%
- **Order Success Rate**: >99%
- **API Error Rate**: <1%
- **Ghost Positions**: 0 per day
- **Data Accuracy**: 100%

#### Performance
- **Asset Screening**: <5 seconds
- **QuantCrawler Analysis**: <3 minutes
- **Order Execution**: <500ms
- **Position Monitoring**: <100ms
- **API Response Time**: <200ms (95th percentile)

### 8.3 Risk Metrics

#### Risk Controls
- **Stop Loss Hit Rate**: 20-30%
- **Take Profit Hit Rate**: 60-70%
- **Daily Loss Limit Breaches**: 0 per month
- **Weekly Loss Limit Breaches**: 0 per month
- **Emergency Stops**: 0 per month (expected)

#### Position Management
- **Average Position Duration**: 5-15 minutes
- **Max Position Duration**: 1 hour
- **Average Leverage**: 15-20x
- **Position Size Accuracy**: 100%

### 8.4 Success Checklist

#### Launch Criteria
- [ ] All critical bugs fixed
- [ ] All tests passing
- [ ] 48-hour testnet validation complete
- [ ] Performance targets met on testnet
- [ ] Risk controls validated
- [ ] Monitoring configured
- [ ] Emergency procedures tested
- [ ] Team trained
- [ ] Documentation complete
- [ ] Deployment tested

#### Post-Launch Criteria (30 Days)
- [ ] System uptime >99%
- [ ] Win rate 65-75%
- [ ] Monthly return 15-25%
- [ ] Max drawdown <2%
- [ ] No ghost positions
- [ ] API error rate <1%
- [ ] All alerts working
- [ ] Team comfortable with operations

---

## Appendix

### A. Configuration Reference

#### Environment Variables
```bash
# Binance API
BINANCE_API_KEY=your_api_key
BINANCE_API_SECRET=your_secret
BINANCE_USE_TESTNET=false

# Telegram
TELEGRAM_TOKEN=your_bot_token
TELEGRAM_CHAT_ID=your_chat_id

# Kill Switch
KILL_SWITCH_PASSWORD=your_password

# Ollama (Local AI)
OLLAMA_BASE_URL=http://localhost:11964
OLLAMA_MODEL=qwen3:0.6b

# QuantCrawler
QUANTCRAWLER_EMAIL=your_email@gmail.com
QUANTCRAWLER_PASSWORD=your_password
```

#### Configuration File
```yaml
# config/production.yaml
binance:
  api_key: "${BINANCE_API_KEY}"
  api_secret: "${BINANCE_API_SECRET}"
  use_testnet: false
  rate_limit_rps: 8
  rate_limit_burst: 16

trading:
  initial_capital_usd: 26
  min_position_usd: 8
  max_position_usd: 13
  risk_per_trade: 0.02
  max_daily_loss: 13.0
  max_weekly_loss: 26.0
  stop_loss_percent: 0.015
  take_profit_percent: 0.05
  trailing_stop_enabled: true
  trailing_stop_percent: 0.01
  max_positions: 3
  support_limit_orders: true
  support_multiple_position_options: true

screener:
  min_volume_24h: 10_000_000
  max_volume_24h: 100_000_000
  min_price: 0.01
  max_price: 10.0
  min_confidence: 0.75
  max_assets: 10

quantcrawler:
  enabled: true
  timeout: 180

stealth:
  enabled: true
  jitter_range_ms: 20
  request_delay_min_ms: 100
  request_delay_max_ms: 300
  signature_variance: 0.01

brain:
  enabled: true
  inference_mode: "LOCAL"
  decision_timeout: 8

monitoring:
  enabled: true
  telegram_enabled: true

emergency:
  kill_switch_enabled: true
```

### B. QuantCrawler Request Format

#### Requesting Structured Output

QuantCrawler can be prompted to return information in specific formats when asked. This capability allows for more reliable parsing and structured data extraction.

**Prompt Format for Structured Output**:

```
Analyze [SYMBOL] and provide trading setup with the following structured output:

1. JSON Format for Key Parameters:
```json
{
  "ticker": "SYMBOL",
  "current_price": 0.2511,
  "entry_price": 0.252,
  "order_type": "LIMIT",
  "direction": "SHORT",
  "confidence": 75,
  "stop_loss": 0.2525,
  "take_profit": 0.241,
  "risk_per_contract": 125,
  "risk_reward_ratio": 1.5,
  "confluence_score": "3/3"
}
```

2. Position Sizing Options:
- Option 1: [description] with entry/stop/target prices
- Option 2: [description] with entry/stop/target prices
- Option 3: [description] with entry/stop/target prices

3. Contract Specifications:
- Tick Size: [value]
- Tick Value: [value]

4. Timeframe Analysis:
- 15m: [analysis]
- 5m: [analysis]
- 1m: [analysis]

5. Key Levels:
- Support: [price]
- Resistance: [price]

6. Invalidation Conditions:
- [conditions]

7. Execution Instructions:
- [instructions]
```

**Request Parameters**:

| Parameter | Type | Description | Example |
|-----------|------|-------------|---------|
| symbol | string | Ticker symbol to analyze | MMTU |
| timeframes | array | Timeframes to analyze | ["1m", "5m", "15m"] |
| risk_amount | number | Fixed risk amount per trade | 200 |
| preferred_option | number | Preferred position sizing option | 2 |
| output_format | string | Desired output format | "json" or "plain" |

**Example Request via N8N**:

```json
{
  "symbol": "MMTU",
  "timeframes": ["1m", "5m", "15m"],
  "risk_amount": 200,
  "preferred_option": 2,
  "output_format": "json",
  "prompt": "Analyze MMTU and provide trading setup with JSON format for key parameters, 3 position sizing options, contract specs, timeframe analysis, key levels, invalidation conditions, and execution instructions."
}
```

**Response Format Options**:

1. **Plain Text (Default)**: Human-readable format with emojis and sections
2. **JSON Structured**: Machine-readable format for key parameters
3. **Mixed Format**: JSON for parameters + plain text for analysis

**Best Practices**:

- Always request JSON format for key trading parameters (entry, stop, target, confidence)
- Include specific output format requirements in the prompt
- Request explicit numeric values for all price levels
- Ask for confidence as a percentage (0-100)
- Request confluence score as "X/Y" format
- Include risk amount in the request for accurate position sizing

**Prompt Templates**:

**Template 1: Basic Analysis**
```
Analyze {SYMBOL} for futures trading. Provide:
- Entry, stop, and target prices
- Direction (LONG/SHORT)
- Confidence score (0-100)
- Risk-reward ratio
- 3 position sizing options with contract counts
```

**Template 2: Structured JSON**
```
Analyze {SYMBOL} and return trading setup in JSON format with:
ticker, current_price, entry_price, order_type, direction, confidence, stop_loss, take_profit, risk_per_contract, risk_reward_ratio, confluence_score
```

**Template 3: Complete Analysis**
```
Analyze {SYMBOL} with {RISK_AMOUNT} risk. Provide:
1. JSON format for all key parameters
2. 3 position sizing options (single, multiple, structure-based)
3. Contract specs (tick size, tick value)
4. Timeframe analysis (1m, 5m, 15m)
5. Key levels (support, resistance)
6. Invalidation conditions
7. Execution instructions
```

### C. QuantCrawler Response Format

#### Response Structure
QuantCrawler returns plain text (not JSON) with the following structure:

```
🕐 Analysis generated: [timestamp]
⚠️ DISCLAIMER: [warning text]
🚨 IMPORTANT: [ticker warning]

TICKER: [SYMBOL]

🔹 TRADE SETUP
Contract: [SYMBOL] (AI specs) ([SYMBOL])
Current Price: [price]
Entry: [price] 🎯 [ORDER TYPE] - [execution instructions]
Confidence: [percentage]%
[confidence cap note]
Direction: [emoji] [DIRECTION]
RECOMMENDATION: [trade recommendation text]

🎯 POSITION SIZING OPTIONS
Contract Specs: Tick Size [points] | Tick Value $[value]

OPTION 1: [option description]
Contracts: [number]
Risk: $[amount] total
Stop Distance: [points]
Target Distance: [points]
Entry: [price] | Stop: [price] | Target: [price]
Best for: [use case]

OPTION 2: [option description] ⭐ [label]
Contracts: [number]
Risk: $[amount] per contract ($[amount] total)
Stop Distance: [points]
Target Distance: [points]
Entry: [price] | Stop: [price] | | Target: [price]
Best for: [use case]

OPTION 3: [option description] ⭐ [label]
Contracts: [number]
Stop: [stop level description]
Target: [target level description]
Stop Distance: [points]
Target Distance: [points]
Risk per Contract: ~$[amount]
Risk:Reward Ratio: [ratio]
Entry: [price] | Stop: [price] | Target: [price]
Best for: [use case]

📊 TIMEFRAME ANALYSIS
15m: [analysis text]
5m: [analysis text]
1m: [analysis text]

🔑 KEY LEVELS (from [timeframe] chart)
Support: [price]
Resistance: [price]

⚠️ RISKS
[invalidation conditions]

🎥 EXECUTION
[execution instructions]

CONFLUENCE: [score]/[total] timeframes agree

💡 Want different risk parameters? Reply with "$[amount]" (e.g., "$500") to recalculate.
```

#### Sample Response (MMTU)
```
🕐 Analysis generated: Jan 12, 10:40 PM
⚠️ DISCLAIMER: QuantCrawler may make calculation errors. All information provided is for educational purposes only and should be independently verified before trading. Trade at your own risk.

🚨 IMPORTANT: This ticker (MMTU) is not in our database. Contract specifications and calculations were provided by AI and may be inaccurate. VERIFY ALL NUMBERS independently before trading.

TICKER: MMTU

🔹 TRADE SETUP
Contract: MMTU (AI specs) (MMTU)

Current Price: 0.2511
Entry: 0.252 🎯 LIMIT ORDER - Wait for pullback to 0.252 (resistance)
Confidence: 75%
Capped at 75% for limit orders (pullback may not occur)
Direction: 🔴 SHORT
RECOMMENDATION: SHORT All timeframes confirm strong bearish momentum pressing price lower suggesting a short opportunity on a pullback to resistance.

🎯 POSITION SIZING OPTIONS
Contract Specs: Tick Size 0.25 points | Tick Value $5

OPTION 1: Single Contract - Wider Stop (Fixed $200 Risk)
Contracts: 1
Risk: $200 total
Stop Distance: 0 points
Target Distance: 0 points
Entry: 0.252 | Stop: 0 | Target: 0
Best for: Swing trades, volatile sessions

OPTION 2: Multiple Contracts - Tighter Stop (Fixed $200 Risk) ⭐ POPULAR
Contracts: 3
Risk: $67 per contract ($200 total)
Stop Distance: 0 points
Target Distance: 0 points
Entry: 0.252 | Stop: 0 | Target: 0
Best for: Scalping, day trading

OPTION 3: Chart Structure - 5m Support/Resistance ⭐ PRO TRADER FAVORITE
Contracts: 1
Stop: Above 5m Resistance (6.5)
Target: At 5m Support (-9.12)
Stop Distance: 6.248 points
Target Distance: 9.372 points
Risk per Contract: ~$125
Risk:Reward Ratio: 1:1.5
Entry: 0.252 | Stop: 6.5 | Target: -9.12
Best for: Trading structure, higher win rate

📊 TIMEFRAME ANALYSIS
15m: After a significant rally price has experienced a sharp and sustained decline indicating a strong short-term bearish trend.
5m: After a significant rally price has experienced a sharp and sustained decline indicating a strong short-term bearish trend.
1m: Strong bearish momentum is evident with large red candles and sustained selling pressure pushing price down towards support.

🔑 KEY LEVELS (from 5m chart)
Support: 0.251
Resistance: 0.252

⚠️ RISKS
Invalidation occurs if price breaks strongly above 0.2525 on sustained volume.

🎥 EXECUTION
Place a LIMIT order to short at 0.2520 waiting for a pullback up to this resistance level.

CONFLUENCE: 3/3 timeframes agree

💡 Want different risk parameters? Reply with "$[amount]" (e.g., "$500") to recalculate.

✅ Took this trade
🚫 Did not take
```

#### Parsing Requirements

**Mandatory Fields**:
- Ticker symbol
- Current price
- Entry price
- Order type (LIMIT/MARKET)
- Direction (with emoji 🔴/🟢)
- Confidence percentage
- Recommendation text

**Position Sizing Options** (at least 1, typically 3):
- Number of contracts
- Total risk amount
- Entry price
- Stop price
- Target price
- Stop distance (points)
- Target distance (points)
- Risk per contract
- Risk-reward ratio (if available)

**Contract Specifications**:
- Tick size
- Tick value

**Timeframe Analysis**:
- 15m analysis
- 5m analysis
- 1m analysis

**Key Levels**:
- Support price
- Resistance price

**Risks**:
- Invalidation conditions

**Execution**:
- Order placement instructions

**Confluence**:
- Timeframes agreement score (e.g., "3/3")

#### JSON Format Extraction (When Requested)

When QuantCrawler is prompted with structured format requests, it may return JSON blocks embedded in the plain text response. These should be extracted and parsed separately.

**JSON Block Pattern**:
```
```json
{
  "ticker": "MMTU",
  "current_price": 0.2511,
  "entry_price": 0.252,
  "order_type": "LIMIT",
  "direction": "SHORT",
  "confidence": 75,
  "stop_loss": 0.2525,
  "take_profit": 0.241,
  "risk_per_contract": 125,
  "risk_reward_ratio": 1.5,
  "confluence_score": "3/3"
}
```
```

**JSON Extraction Strategy**:
1. **Locate JSON blocks**: Search for ```json code blocks
2. **Extract JSON content**: Parse content between ```json and ```
3. **Validate JSON**: Ensure valid JSON structure
4. **Parse fields**: Extract all fields from JSON object
5. **Fallback to plain text**: If JSON extraction fails, fall back to plain text parsing

**JSON Field Mapping**:

| JSON Field | Type | Description | Plain Text Equivalent |
|------------|------|-------------|----------------------|
| ticker | string | Ticker symbol | TICKER: line |
| current_price | number | Current price | Current Price: line |
| entry_price | number | Entry price | Entry: line |
| order_type | string | Order type | Entry: line (LIMIT/MARKET) |
| direction | string | Direction | Direction: line |
| confidence | number | Confidence (0-100) | Confidence: line |
| stop_loss | number | Stop loss price | Stop: line in position options |
| take_profit | number | Take profit price | Target: line in position options |
| risk_per_contract | number | Risk per contract | Risk per Contract: line |
| risk_reward_ratio | number | Risk-reward ratio | Risk:Reward Ratio: line |
| confluence_score | string | Confluence score | CONFLUENCE: line |

**Extraction Example**:

```go
func extractJSONBlock(response string) (map[string]interface{}, error) {
    // Find JSON code blocks
    re := regexp.MustCompile("```json\\s*([\\s\\S]*?)\\s*```")
    matches := re.FindAllStringSubmatch(response, -1)

    if len(matches) == 0 {
        return nil, fmt.Errorf("no JSON block found")
    }

    // Parse first JSON block
    var result map[string]interface{}
    err := json.Unmarshal([]byte(matches[0][1]), &result)
    if err != nil {
        return nil, fmt.Errorf("failed to parse JSON: %w", err)
    }

    return result, nil
}
```

**Hybrid Parsing Strategy**:
1. First attempt to extract and parse JSON blocks
2. If JSON extraction successful, use JSON values for key parameters
3. Still parse plain text for position sizing options, timeframe analysis, etc.
4. If JSON extraction fails, use plain text parsing for all fields
5. Validate consistency between JSON and plain text (if both present)

**Benefits of JSON Format**:
- More reliable parsing of numeric values
- No ambiguity in field extraction
- Easier to validate data types
- Reduced parsing errors
- Better support for edge cases

**Fallback Handling**:
- If JSON block is malformed, log error and fall back to plain text
- If required fields missing from JSON, check plain text
- If values inconsistent between JSON and plain text, prefer JSON
- Always validate JSON values against constraints (e.g., confidence 0-100)

#### Parsing Strategy

1. **Extract Ticker**: Find line starting with "TICKER:"
2. **Extract Price**: Find line starting with "Current Price:"
3. **Extract Entry**: Find line starting with "Entry:"
4. **Extract Direction**: Find line with Direction and emoji
5. **Extract Confidence**: Find line with "Confidence:" and parse percentage
6. **Parse Position Options**: Find all "OPTION X:" sections
7. **Parse Contract Specs**: Find "Contract Specs:" line
8. **Parse Key Levels**: Find "Support:" and "Resistance:" lines
9. **Parse Risks**: Find "⚠️ RISKS" section
10. **Parse Execution**: Find "🎥 EXECUTION" section
11. **Parse Confluence**: Find "CONFLUENCE:" line

### C. API Reference

#### Binance Futures API Endpoints
- **Server Time**: GET /fapi/v1/time
- **Exchange Info**: GET /fapi/v1/exchangeInfo
- **Account**: GET /fapi/v2/account
- **Position Risk**: GET /fapi/v2/positionRisk
- **New Order**: POST /fapi/v1/order
- **Cancel Order**: DELETE /fapi/v1/order
- **Change Leverage**: POST /fapi/v1/leverage
- **Klines**: GET /fapi/v1/klines
- **24hr Ticker**: GET /fapi/v1/ticker/24hr

#### GOBOT Webhooks
- **Trade Signal**: POST /webhook/trade_signal
- **Risk Alert**: POST /webhook/risk-alert
- **Market Analysis**: POST /webhook/market-analysis
- **Capture Chart**: POST /webhook/capture-chart
- **Analyze Symbol**: POST /webhook/analyze-symbol

#### GOBOT API Endpoints
- **Metrics**: GET /metrics
- **Health**: GET /health
- **Trades**: GET /trades
- **Positions**: GET /positions

### C. Troubleshooting Guide

#### Common Issues

**Issue**: API authentication failed
- **Cause**: Wrong API key or signature
- **Solution**: Check API keys, verify signature algorithm

**Issue**: Rate limit exceeded
- **Cause**: Too many requests
- **Solution**: Wait for circuit breaker cooldown, reduce request rate

**Issue**: Order rejected
- **Cause**: Invalid order parameters
- **Solution**: Check order validation logs, fix parameters

**Issue**: Position not closing
- **Cause**: SL/TP not triggered
- **Solution**: Check position health, close manually if needed

**Issue**: QuantCrawler timeout
- **Cause**: N8N workflow slow or down
- **Solution**: Check N8N status, increase timeout

### D. Contact & Support

#### Team Contacts
- **Lead Developer**: [Contact]
- **DevOps Engineer**: [Contact]
- **System Administrator**: [Contact]

#### Emergency Contacts
- **On-Call Engineer**: [Contact]
- **Team Lead**: [Contact]

#### Documentation
- **Main Documentation**: IFLOW.md
- **API Documentation**: API_REFERENCE.md
- **Deployment Guide**: DEPLOYMENT_GUIDE.md
- **Runbook**: RUNBOOK.md

---

## Change Log

### Version 3.0 (2026-01-16)
- Initial PRD & Implementation Checklist
- Complete feature specification
- Comprehensive implementation plan
- Risk management framework
- Testing & validation strategy
- Deployment procedures
- Success criteria defined

---

**Document Status**: Draft  
**Last Updated**: 2026-01-16  
**Next Review**: 2026-01-23  
**Approved By**: [Pending]