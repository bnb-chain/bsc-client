package blockarchiver

type BlockArchiverConfig struct {
	RPCAddress     string
	BlockCacheSize int64
}

var DefaultBlockArchiverConfig = BlockArchiverConfig{
	BlockCacheSize: 5000,
}
