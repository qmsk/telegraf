package docker

import (
    "github.com/fsouza/go-dockerclient"
    "fmt"
    "log"
    "os"
)

// monitored container state
type monitorContainer struct {
    name        string
    log         *log.Logger
    tags         map[string]string
    statsChan    chan *docker.Stats
    stats        *docker.Stats

    // calculating deltas in gather()
    prevStats  *docker.Stats
}

func (self monitorContainer) String() string {
    return self.name
}

// call as a new goroutine to start monitoring stats
// closes statsChan on errors
func (self *monitorContainer) start(dockerClient *docker.Client, listContainer docker.APIContainers) {
    statsOptions := docker.StatsOptions{
        ID:     listContainer.ID,
        Stats:  self.statsChan,
        Stream: true,
    }

    self.log.Printf("Start %#v...\n", statsOptions)

    // this is a blocking operation
    if err := dockerClient.Stats(statsOptions); err != nil {
        self.log.Printf("Error: %v\n", err)
    } else {
        self.log.Printf("End\n")
    }
}

// maintain latests stats
func (self *monitorContainer) run() {
    for dockerStats := range self.statsChan {
        self.log.Printf("Stats\n")

        self.stats = dockerStats
    }

    self.stats = nil
}

// starts monitoring a container's stats
func newMonitorContainer(dockerClient *docker.Client, listContainer docker.APIContainers) *monitorContainer {
    containerName := listContainer.Names[0][1:]

    monitorContainer := &monitorContainer{
        name:       containerName,
        log:        log.New(os.Stderr, fmt.Sprintf("plugins/docker %s: ", listContainer.ID), 0),
        tags:       map[string]string{
            "id":       listContainer.ID,
            "image":    listContainer.Image,
            "name":     containerName,
        },
        statsChan:  make(chan *docker.Stats),
    }

    go monitorContainer.start(dockerClient, listContainer)
    go monitorContainer.run()

    return monitorContainer
}

func (self *monitorContainer) GatherNetwork() map[string]interface{} {
    fields := make(map[string]interface{})

    stats := self.stats

    fields["rx_bytes"]      = stats.Network.RxBytes
    fields["rx_dropped"]    = stats.Network.RxDropped
    fields["rx_errors"]     = stats.Network.RxErrors
    fields["rx_packets"]    = stats.Network.RxPackets
    fields["tx_bytes"]      = stats.Network.TxBytes
    fields["tx_dropped"]    = stats.Network.TxDropped
    fields["tx_errors"]     = stats.Network.TxErrors
    fields["tx_packets"]    = stats.Network.TxPackets

    return fields
}

func (self *monitorContainer) GatherMemory() map[string]interface{} {
    fields := make(map[string]interface{})

    stats := self.stats

    // omit memory stats if cgroup is missing
    if stats.MemoryStats.Usage != 0 && stats.MemoryStats.MaxUsage != 0 {
        fields["cache"]     = stats.MemoryStats.Stats.Cache
        fields["rss"]       = stats.MemoryStats.Stats.Rss
        fields["max_usage"] = stats.MemoryStats.MaxUsage
        fields["usage"]     = stats.MemoryStats.Usage
        fields["failcnt"]   = stats.MemoryStats.Failcnt
        fields["limit"]     = stats.MemoryStats.Limit
    }

    return fields
}

// return fields for docker2_cpu metric
// computes usage deltas from the previous gather cycle for accounting accuracy over our entire interval
func (self *monitorContainer) GatherCPU() map[string]interface{} {
    fields := make(map[string]interface{})

    stats := self.stats
    prevStats := self.prevStats

    fields["count"]         = len(stats.CPUStats.CPUUsage.PercpuUsage)

    // XXX: are these fields even useful...?
    fields["total_usage"]   = stats.CPUStats.CPUUsage.TotalUsage
    fields["user_usage"]    = stats.CPUStats.CPUUsage.UsageInUsermode
    fields["kernel_usage"]  = stats.CPUStats.CPUUsage.UsageInKernelmode
    fields["system_usage"]  = stats.CPUStats.SystemCPUUsage

    // only calculate deltas if we have prev stats
    if prevStats != nil && prevStats.Read != stats.Read {
        fields["total_delta"]   = stats.CPUStats.CPUUsage.TotalUsage         - prevStats.CPUStats.CPUUsage.TotalUsage
        fields["user_delta"]    = stats.CPUStats.CPUUsage.UsageInUsermode    - prevStats.CPUStats.CPUUsage.UsageInUsermode
        fields["kernel_delta"]  = stats.CPUStats.CPUUsage.UsageInKernelmode  - prevStats.CPUStats.CPUUsage.UsageInKernelmode
        fields["system_delta"]  = stats.CPUStats.SystemCPUUsage              - prevStats.CPUStats.SystemCPUUsage
    }

    self.prevStats = stats

    return fields
}
