package setting

import "forgejo.org/modules/log"

var Cgroup = struct {
	Enabled      bool
	FsMountpoint string
	PidLimit     uint64
}{
	Enabled:      false,
	FsMountpoint: "/sys/fs/cgroup",
	PidLimit:     100,
}

func loadCgroupFrom(rootCfg ConfigProvider) {
	err := rootCfg.Section("cgroup").MapTo(&Cgroup)
	if err != nil {
		log.Fatal("Failed to load Cgroup settings: %v", err)
	}
}
