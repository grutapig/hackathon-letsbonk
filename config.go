package main

const ENV_TWITTER_API_KEY = "twitter_api_key"
const ENV_PROXY_DSN = "proxy_dsn"
const ENV_PROXY_CLAUDE_DSN = "proxy_claude_dsn"
const ENV_TWITTER_API_BASE_URL = "twitter_api_base_url"
const ENV_DEMO_COMMUNITY_ID = "demo_community_id"
const ENV_DEMO_TWEET_ID = "demo_tweet_id"
const ENV_DEMO_USER_NAME = "demo_user_name"
const ENV_DEMO_USER_ID = "demo_user_id"
const ENV_TWITTER_COMMUNITY_TICKER = "twitter_community_ticker"
const ENV_MONITORING_METHOD = "monitoring_method" // "incremental" or "full_scan"
const ENV_CLAUDE_API_KEY = "claude_api_key"
const ENV_TELEGRAM_API_KEY = "telegram_api_key"
const ENV_TELEGRAM_ADMIN_CHAT_ID = "tg_admin_chat_id"
const ENV_TARGET_USERS = "target_users"
const ENV_DATABASE_NAME = "database_name"
const ENV_IMPORT_CSV_PATH = "import_csv_path"
const ENV_NOTIFICATION_USERS = "notification_users"
const ENV_CLEAR_ANALYSIS_ON_START = "clear_analysis_on_start"
const ENV_SOLANA_RPC_URL = "solana_rpc"
const ENV_LOGGING_DATABASE_PATH = "logging_database_path"

// Twitter Reverse API constants
const ENV_TWITTER_REVERSE_AUTHORIZATION = "twitter_reverse_authorization"
const ENV_TWITTER_REVERSE_CSRF_TOKEN = "twitter_reverse_csrf_token"
const ENV_TWITTER_REVERSE_COOKIE = "twitter_reverse_cookie"
const ENV_TWITTER_REVERSE_ENABLED = "twitter_reverse_enabled"
const ENV_TWITTER_AUTH = "twitter_auth"
const ENV_TWITTER_BOT_TAG = "twitter_bot_tag"

// Monitoring method constants
const MONITORING_METHOD_INCREMENTAL = "incremental"
const MONITORING_METHOD_FULL_SCAN = "full_scan"

// Message processing constants
const PROCESSING_TYPE_DETAILED = "detailed" // Detailed user analysis (current second step)
const PROCESSING_TYPE_FAST = "fast"         // Fast notification without detailed analysis

// Tweet source type constants
const TWEET_SOURCE_COMMUNITY = "community"         // Tweet from community monitoring
const TWEET_SOURCE_TICKER_SEARCH = "ticker_search" // Tweet from ticker mention search
const TWEET_SOURCE_CONTEXT = "context"             // Tweet loaded for context (replies)
const TWEET_SOURCE_MONITORING = "monitoring"       // Tweet from general monitoring

// User relation type constants
const RELATION_TYPE_FOLLOWER = "follower"   // User is a follower of another user
const RELATION_TYPE_FOLLOWING = "following" // User is following another user
