package types

import "time"

// Config is a struct to hold the configuration data
type Config struct {
	Logging struct {
		OutputLevel  string `yaml:"outputLevel" envconfig:"LOGGING_OUTPUT_LEVEL"`
		OutputStderr bool   `yaml:"outputStderr" envconfig:"LOGGING_OUTPUT_STDERR"`

		FilePath  string `yaml:"filePath" envconfig:"LOGGING_FILE_PATH"`
		FileLevel string `yaml:"fileLevel" envconfig:"LOGGING_FILE_LEVEL"`
	} `yaml:"logging"`

	Server struct {
		Port string `yaml:"port" envconfig:"FRONTEND_SERVER_PORT"`
		Host string `yaml:"host" envconfig:"FRONTEND_SERVER_HOST"`
	} `yaml:"server"`

	Chain struct {
		Name             string `yaml:"name" envconfig:"CHAIN_NAME"`
		DisplayName      string `yaml:"displayName" envconfig:"CHAIN_DISPLAY_NAME"`
		GenesisTimestamp uint64 `yaml:"genesisTimestamp" envconfig:"CHAIN_GENESIS_TIMESTAMP"`
		ConfigPath       string `yaml:"configPath" envconfig:"CHAIN_CONFIG_PATH"`
		Config           ChainConfig

		// optional features
		WhiskForkEpoch *uint64 `yaml:"whiskForkEpoch" envconfig:"WHISK_FORK_EPOCH"`
	} `yaml:"chain"`

	Frontend struct {
		Enabled bool `yaml:"enabled" envconfig:"FRONTEND_ENABLED"`
		Debug   bool `yaml:"debug" envconfig:"FRONTEND_DEBUG"`
		Pprof   bool `yaml:"pprof" envconfig:"FRONTEND_PPROF"`
		Minify  bool `yaml:"minify" envconfig:"FRONTEND_MINIFY"`

		SiteDomain   string `yaml:"siteDomain" envconfig:"FRONTEND_SITE_DOMAIN"`
		SiteName     string `yaml:"siteName" envconfig:"FRONTEND_SITE_NAME"`
		SiteSubtitle string `yaml:"siteSubtitle" envconfig:"FRONTEND_SITE_SUBTITLE"`

		EthExplorerLink         string `yaml:"ethExplorerLink" envconfig:"FRONTEND_ETH_EXPLORER_LINK"`
		ValidatorNamesYaml      string `yaml:"validatorNamesYaml" envconfig:"FRONTEND_VALIDATOR_NAMES_YAML"`
		ValidatorNamesInventory string `yaml:"validatorNamesInventory" envconfig:"FRONTEND_VALIDATOR_NAMES_INVENTORY"`

		PageCallTimeout  time.Duration `yaml:"pageCallTimeout" envconfig:"FRONTEND_PAGE_CALL_TIMEOUT"`
		HttpReadTimeout  time.Duration `yaml:"httpReadTimeout" envconfig:"FRONTEND_HTTP_READ_TIMEOUT"`
		HttpWriteTimeout time.Duration `yaml:"httpWriteTimeout" envconfig:"FRONTEND_HTTP_WRITE_TIMEOUT"`
		HttpIdleTimeout  time.Duration `yaml:"httpIdleTimeout" envconfig:"FRONTEND_HTTP_IDLE_TIMEOUT"`
		AllowDutyLoading bool          `yaml:"allowDutyLoading" envconfig:"FRONTEND_ALLOW_DUTY_LOADING"`
	} `yaml:"frontend"`

	RateLimit struct {
		Enabled    bool `yaml:"enabled" envconfig:"RATELIMIT_ENABLED"`
		ProxyCount uint `yaml:"proxyCount" envconfig:"RATELIMIT_PROXY_COUNT"`
		Rate       uint `yaml:"rate" envconfig:"RATELIMIT_RATE"`
		Burst      uint `yaml:"burst" envconfig:"RATELIMIT_BURST"`
	} `yaml:"rateLimit"`

	BeaconApi struct {
		Endpoint  string           `yaml:"endpoint" envconfig:"BEACONAPI_ENDPOINT"`
		Endpoints []EndpointConfig `yaml:"endpoints"`

		LocalCacheSize       int    `yaml:"localCacheSize" envconfig:"BEACONAPI_LOCAL_CACHE_SIZE"`
		SkipFinalAssignments bool   `yaml:"skipFinalAssignments" envconfig:"BEACONAPI_SKIP_FINAL_ASSIGNMENTS"`
		AssignmentsCacheSize int    `yaml:"assignmentsCacheSize" envconfig:"BEACONAPI_ASSIGNMENTS_CACHE_SIZE"`
		RedisCacheAddr       string `yaml:"redisCacheAddr" envconfig:"BEACONAPI_REDIS_CACHE_ADDR"`
		RedisCachePrefix     string `yaml:"redisCachePrefix" envconfig:"BEACONAPI_REDIS_CACHE_PREFIX"`
	} `yaml:"beaconapi"`

	Indexer struct {
		InMemoryEpochs                  uint16 `yaml:"inMemoryEpochs" envconfig:"INDEXER_IN_MEMORY_EPOCHS"`
		CachePersistenceDelay           uint16 `yaml:"cachePersistenceDelay" envconfig:"INDEXER_CACHE_PERSISTENCE_DELAY"`
		DisableIndexWriter              bool   `yaml:"disableIndexWriter" envconfig:"INDEXER_DISABLE_INDEX_WRITER"`
		DisableSynchronizer             bool   `yaml:"disableSynchronizer" envconfig:"INDEXER_DISABLE_SYNCHRONIZER"`
		SyncEpochCooldown               uint   `yaml:"syncEpochCooldown" envconfig:"INDEXER_SYNC_EPOCH_COOLDOWN"`
		MaxParallelValidatorSetRequests uint   `yaml:"maxParallelValidatorSetRequests" envconfig:"INDEXER_MAX_PARALLEL_VALIDATOR_SET_REQUESTS"`
	} `yaml:"indexer"`

	BlobStore struct {
		PersistenceMode string `yaml:"persistenceMode" envconfig:"BLOBSTORE_PERSISTENCE_MODE"`
		NameTemplate    string `yaml:"nameTemplate" envconfig:"BLOBSTORE_NAME_TEMPLATE"`

		Fs struct {
			Path string `yaml:"path" envconfig:"BLOBSTORE_FS_PATH"`
		} `yaml:"fs"`
		Aws struct {
			AccessKey string `yaml:"accessKey" envconfig:"BLOBSTORE_AWS_ACCESSKEY"`
			SecretKey string `yaml:"secretKey" envconfig:"BLOBSTORE_AWS_SECRETKEY"`
			S3Region  string `yaml:"s3Region" envconfig:"BLOBSTORE_AWS_S3REGION"`
			S3Bucket  string `yaml:"s3Bucket" envconfig:"BLOBSTORE_AWS_S3BUCKET"`
		} `yaml:"aws"`
	} `yaml:"blobstore"`

	Database struct {
		Engine string `yaml:"engine" envconfig:"DATABASE_ENGINE"`
		Sqlite struct {
			File         string `yaml:"file" envconfig:"DATABASE_SQLITE_FILE"`
			MaxOpenConns int    `yaml:"maxOpenConns" envconfig:"DATABASE_SQLITE_MAX_OPEN_CONNS"`
			MaxIdleConns int    `yaml:"maxIdleConns" envconfig:"DATABASE_SQLITE_MAX_IDLE_CONNS"`
		} `yaml:"sqlite"`
		Pgsql struct {
			Username     string `yaml:"user" envconfig:"DATABASE_PGSQL_USERNAME"`
			Password     string `yaml:"password" envconfig:"DATABASE_PGSQL_PASSWORD"`
			Name         string `yaml:"name" envconfig:"DATABASE_PGSQL_NAME"`
			Host         string `yaml:"host" envconfig:"DATABASE_PGSQL_HOST"`
			Port         string `yaml:"port" envconfig:"DATABASE_PGSQL_PORT"`
			MaxOpenConns int    `yaml:"maxOpenConns" envconfig:"DATABASE_PGSQL_MAX_OPEN_CONNS"`
			MaxIdleConns int    `yaml:"maxIdleConns" envconfig:"DATABASE_PGSQL_MAX_IDLE_CONNS"`
		} `yaml:"pgsql"`
		PgsqlWriter struct {
			Username     string `yaml:"user" envconfig:"DATABASE_PGSQL_WRITER_USERNAME"`
			Password     string `yaml:"password" envconfig:"DATABASE_PGSQL_WRITER_PASSWORD"`
			Name         string `yaml:"name" envconfig:"DATABASE_PGSQL_WRITER_NAME"`
			Host         string `yaml:"host" envconfig:"DATABASE_PGSQL_WRITER_HOST"`
			Port         string `yaml:"port" envconfig:"DATABASE_PGSQL_WRITER_PORT"`
			MaxOpenConns int    `yaml:"maxOpenConns" envconfig:"DATABASE_PGSQL_WRITER_MAX_OPEN_CONNS"`
			MaxIdleConns int    `yaml:"maxIdleConns" envconfig:"DATABASE_PGSQL_WRITER_MAX_IDLE_CONNS"`
		} `yaml:"pgsqlWriter"`
	} `yaml:"database"`
}

type EndpointConfig struct {
	Ssh            *EndpointSshConfig `yaml:"ssh"`
	Url            string             `yaml:"url"`
	Name           string             `yaml:"name"`
	Archive        bool               `yaml:"archive"`
	SkipValidators bool               `yaml:"skipValidators"`
	Priority       int                `yaml:"priority"`
	Headers        map[string]string  `yaml:"headers"`
}

type EndpointSshConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Keyfile  string `yaml:"keyfile"`
}

type SqliteDatabaseConfig struct {
	File         string
	MaxOpenConns int
	MaxIdleConns int
}

type PgsqlDatabaseConfig struct {
	Username     string
	Password     string
	Name         string
	Host         string
	Port         string
	MaxOpenConns int
	MaxIdleConns int
}
