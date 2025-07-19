# FUD Detection System - System Architecture & Operation Flow

## Overview

This is a comprehensive Go-based Twitter/X monitoring and FUD (Fear, Uncertainty, Doubt) detection system that uses Claude AI to analyze crypto community posts and user behavior in real-time. The system monitors Twitter communities for new messages, performs two-step AI analysis to identify potential FUDders, and provides Telegram-based administrative interface.

## System Initialization

### 1. Application Startup (`main.go`)

The application starts with the following sequence:

1. **Configuration Loading**
   - Loads configuration from `.env`, `.dev.env`, or specified config file
   - Parses command-line flags (`-config`, `-help`)
   - Validates required environment variables

2. **Dependency Injection Container** (`container.go`)
   - Uses `uber-go/dig` for dependency injection
   - Builds container with all services and their dependencies
   - Ensures proper service initialization order

3. **Application Initialization** (`app.go`)
   - Initializes all services (Database, Logging, Telegram, etc.)
   - Starts cleanup scheduler for log maintenance
   - Loads system prompts for AI analysis
   - Performs initial data loading (CSV import or community loading)
   - Starts Telegram bot listener

## Core Architecture Components

### Services Layer

#### 1. **Twitter API Service** (`twitterapi/`)
- **Purpose**: Interface to Twitter API with proxy support
- **Key Functions**:
  - Community tweets monitoring
  - User timeline analysis  
  - Tweet replies retrieval
  - User followers/following data
  - Advanced search functionality
- **Features**: State tracking, rate limiting, batch requests

#### 2. **Twitter Reverse API Service** (`twitterapi_reverse/`)
- **Purpose**: Alternative Twitter API implementation to reduce costs
- **Key Functions**:
  - Direct HTTP requests to Twitter with authentication
  - Community tweets parsing
  - Fallback mechanism when main API fails
- **Authentication**: Uses authorization tokens, CSRF tokens, and cookies

#### 3. **Claude API Client** (`claude_api.go`)
- **Purpose**: Interface to Anthropic's Claude AI
- **Configuration**: Supports proxy, temperature settings, token limits
- **Models**: Uses `claude-sonnet-4-0` for analysis
- **Features**: Request/response logging, error handling

#### 4. **Database Service** (`database_service.go`)
- **Database**: SQLite with GORM ORM
- **Tables**:
  - `tweets`: Tweet content and metadata
  - `users`: User information and FUD status
  - `fud_users`: Detected FUD users with analysis details
  - `user_relations`: Follower/following relationships
  - `analysis_tasks`: Manual analysis task tracking
  - `cached_analysis`: 24-hour cached analysis results
  - `user_ticker_opinions`: User ticker mention analysis
  - Enhanced `users` table: Now includes status, analysis tracking, and FUD information

#### 5. **Logging Service** (`logging_service.go`)
- **Purpose**: Analytics and performance monitoring
- **Separate Database**: `logs.db` for analytics data
- **Tracked Data**:
  - Message activities
  - AI request performance
  - Data collection metrics
  - User activity patterns
  - Request processing timelines

#### 6. **Telegram Service** (`telegram.go`)
- **Purpose**: Administrative interface and notification system
- **Features**:
  - Bot command processing
  - FUD alert broadcasting
  - Manual analysis triggers
  - User management commands
  - Real-time status reporting

#### 7. **User Status Management** (integrated into `database_service.go`)
- **Purpose**: Database-based user analysis status tracking
- **Storage**: SQLite database with additional user status fields
- **Functions**:
  - Track analysis progress in database
  - Prevent duplicate processing
  - Friend network FUD analysis
  - Real-time status updates

## Data Flow & Processing Pipeline

### 1. **Community Monitoring** (`monitoring_handler.go`)

**Initialization Process:**
1. Check if Twitter Reverse API is enabled and configured
2. Initialize monitoring with 3 pages of existing tweets
3. Create baseline mapping of tweet IDs and reply counts

**Monitoring Loop (30-second intervals):**
1. **Tweet Retrieval**: 
   - Try Reverse API first (cost optimization)
   - Fallback to main Twitter API if needed
   - Get latest community tweets

2. **Change Detection**:
   - Compare current reply counts with stored values
   - Identify tweets with new replies
   - Track new posts in community

3. **Reply Processing**:
   - Fetch replies for tweets with increased reply counts
   - Handle nested replies (replies to replies)
   - Maintain thread context (parent → grandparent relationships)

4. **Data Storage**:
   - Store tweets and users in database
   - Log all message activities
   - Update tweet reply counts

5. **Channel Distribution**:
   - Send new messages to `NewMessageCh`
   - Messages broadcast to both First Step Handler and Twitter Bot Handler

### 2. **First Step Analysis** (`first_step_handler.go`)

**Message Reception** from `NewMessageCh`:

**User Classification Logic:**
1. **Known FUD User**: 
   - Quick Claude AI analysis for current message
   - Immediate notification if flagged as FUD
   - Skip detailed analysis

2. **New User** (never analyzed):
   - Direct routing to detailed Second Step analysis
   - Skip first step screening

3. **Existing User** (previously analyzed, not FUD):
   - Standard first step Claude AI analysis
   - Thread context included (grandparent → parent → current)
   - Route to Second Step only if flagged

**AI Analysis Process:**
1. Build message context with full thread hierarchy
2. Send to Claude AI with first step prompt
3. Parse JSON response for FUD flag
4. Log AI request performance and results
5. Update user analysis status

### 3. **Second Step Analysis** (`second_step_handler.go`)

**Comprehensive User Analysis:**

**Cache Check:**
- Check for existing 24-hour cached analysis
- Use cached results if available (performance optimization)
- Skip full analysis for repeated requests

**Data Collection Phase:**
1. **Ticker Mentions** (max 3 pages):
   - Advanced search for user's mentions of community ticker
   - Collect up to 50,000 tokens of content
   - Fetch replied-to messages for context
   - Store ticker opinions in database

2. **Community Activity**:
   - Retrieve user's historical community posts from database
   - Group messages by conversation threads
   - Build complete activity timeline

3. **Social Network Analysis**:
   - Get user's followers and following lists
   - Save relationships to database
   - Analyze FUD connections in network

**AI Analysis:**
1. Prepare comprehensive data package for Claude
2. Include full thread context for current message
3. Send to Claude AI with second step prompt
4. Enhanced analysis for manual requests

**Result Processing:**
1. Parse Claude response with detailed FUD assessment
2. Update user FUD status in database
3. Cache analysis results for 24 hours
4. Mark user as detail-analyzed
5. Send notifications if FUD detected or forced

### 4. **Notification System** (`notification_handler.go`)

**Alert Processing:**
- Receive FUD alerts from `NotificationCh`
- Check for targeted chat delivery
- Format notifications with thread context
- Broadcast to all registered Telegram chats

**Telegram Bot Features:**
- Administrative commands for FUD management
- Manual analysis triggers
- User search and status queries
- System statistics and monitoring
- Chat registration and management

## Configuration & Environment

### Required Environment Variables:
- `twitter_api_key`: Twitter API access
- `twitter_api_base_url`: Twitter API endpoint
- `claude_api_key`: Anthropic Claude API key
- `telegram_api_key`: Telegram bot token
- `tg_admin_chat_id`: Admin chat for notifications
- `demo_community_id`: Twitter community to monitor
- `twitter_community_ticker`: Cryptocurrency ticker symbol

### Optional Configuration:
- `proxy_dsn`: HTTP proxy for Twitter API
- `proxy_claude_dsn`: HTTP proxy for Claude API
- `monitoring_method`: "incremental" (default) or "full_scan"
- `twitter_reverse_*`: Reverse API authentication
- `database_name`: SQLite database file
- `clear_analysis_on_start`: Reset analysis flags on startup

## System Monitoring & Analytics

### Performance Tracking:
- AI request response times
- Data collection metrics
- Processing pipeline performance
- Error rates and patterns

### Data Analytics:
- User activity patterns
- FUD detection accuracy
- Message volume statistics
- Community engagement metrics

### Cleanup & Maintenance:
- Automatic log rotation
- Database optimization
- Periodic status saves
- Error recovery mechanisms

## Error Handling & Resilience

### Graceful Degradation:
- API fallback mechanisms (Reverse API → Main API)
- Cached analysis for repeated requests
- Continue processing on individual failures

### Monitoring & Alerting:
- Comprehensive error logging
- Telegram notifications for system issues
- Performance monitoring dashboards

### Data Integrity:
- Transaction-based database operations
- Duplicate detection and prevention
- Consistent state management

## Concurrency & Performance

### Goroutine Architecture:
- **6 main goroutines** running concurrently:
  1. Community monitoring
  2. Message broadcasting
  3. First step analysis
  4. Second step analysis (dynamic)
  5. Notification handling
  6. Twitter bot mention processing

### Simplified Architecture:
- **Removed User Status Manager**: All user status tracking now handled directly by DatabaseService
- **Database-Centric**: User analysis status persisted in SQLite instead of JSON files
- **Real-time Updates**: Status changes immediately reflected in database

### Channel-Based Communication:
- Buffered channels for high-throughput processing
- Non-blocking message distribution
- Graceful backpressure handling

### Resource Management:
- Connection pooling for HTTP clients
- Database connection optimization
- Memory-efficient data structures
- Token limit management for AI requests

This architecture provides a robust, scalable system for real-time FUD detection in cryptocurrency communities, with comprehensive monitoring, analytics, and administrative capabilities.